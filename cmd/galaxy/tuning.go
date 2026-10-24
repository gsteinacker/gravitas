package main

import (
	"fmt"
	"log"
	"time"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/mac"
	"galaxy/internal/perf"
	"galaxy/internal/prefs"
	"galaxy/internal/scene"
	"galaxy/internal/sim/bh"
	"galaxy/internal/ui"
)

// defaultStars is the star count of screenshots without -n and of uncalibrated machines.
const defaultStars = 50000

// tuning chooses star count and GPU use from the settings and the calibration of this machine.
type tuning struct {
	settings *prefs.Prefs
	noGPU    bool // -gpu=false: calibrate and run on the CPU only
	hw       perf.Hardware
	cal      prefs.Calibration
	// correction is the measured step time of a scene relative to the prediction (T-047); it
	// corrects the star count when the scene is loaded again in this session.
	correction map[*scene.Spec]float64
	running    *scene.Spec
	since      time.Time
}

// newTuning probes the machine and calibrates it once; the result is cached in the settings.
func newTuning(settings *prefs.Prefs, noGPU bool) *tuning {
	t := &tuning{settings: settings, hw: perf.Probe(), noGPU: noGPU}
	if c, ok := settings.Calibration[t.key()]; ok && c.SecPerNL > 0 {
		t.cal = c
		return t
	}
	t0 := time.Now()
	t.calibrate()
	log.Printf("calibrated %s in %v: %d stars for 30 steps/s", t.hw, time.Since(t0).Round(time.Millisecond), perf.StarsFor(t.cal, perf.StepBudget))
	return t
}

// gpuAllowed reports whether the GPU may be used at all.
func (t *tuning) gpuAllowed() bool { return !t.noGPU && t.settings.GPU != prefs.GPUOff }

// key identifies the cached calibration: the machine, and whether the GPU took part.
func (t *tuning) key() string {
	if !t.gpuAllowed() {
		return t.hw.Key() + "|cpu"
	}
	return t.hw.Key()
}

func (t *tuning) calibrate() {
	t.cal = perf.Calibrate(t.gpuAllowed())
	if t.settings.Calibration == nil {
		t.settings.Calibration = map[string]prefs.Calibration{}
	}
	t.settings.Calibration[t.key()] = t.cal
}

// stars is the star count of scene sp.
func (t *tuning) stars(sp *scene.Spec) int {
	if t.settings.Stars > 0 {
		return t.settings.Stars
	}
	corr := 1.0
	if c, ok := t.correction[sp]; ok {
		corr = c
	}
	return perf.ProfileByID(t.settings.Profile).SceneStars(t.cal, sp, corr, defaultStars)
}

// gpu reports whether the tree walk should run on the GPU.
func (t *tuning) gpu() bool {
	switch {
	case !t.gpuAllowed():
		return false
	case t.settings.GPU == prefs.GPUOn:
		return bh.GPUAvailable() == nil
	}
	return t.cal.UseGPU && bh.GPUAvailable() == nil
}

// describe is the machine and its measured speed for the settings window.
func (t *tuning) describe() string {
	return fmt.Sprintf(i18n.T("%s, schafft %s Sterne bei 30 Schritten/s", "%s, runs %s stars at 30 steps/s"),
		t.hw, mac.GroupDigits(perf.StarsFor(t.cal, perf.StepBudget)))
}

// apply hands changed settings to the running game.
func (t *tuning) apply(game *ui.Game) {
	game.N = t.stars(game.Scene)
	game.SetGPU(t.gpu())
	p := perf.ProfileByID(t.settings.Profile)
	game.SetProfile(p.TargetSPS, p.MaxRenderScale)
}

// recalibrate measures the machine again while the simulation is paused (settings window).
func recalibrate(game *ui.Game, t *tuning) {
	paused := game.Runner.Paused()
	game.SetPaused(true)
	game.Notify(i18n.T("Leistung wird gemessen …", "Measuring performance…"))
	gpuAllowed := t.gpuAllowed()
	key := t.key()
	go func() {
		time.Sleep(200 * time.Millisecond) // let the running step finish
		cal := perf.Calibrate(gpuAllowed)
		game.Post(func(*app.Controller) {
			t.cal = cal
			t.settings.Calibration[key] = cal
			mac.SetHardware(t.describe())
			t.apply(game)
			game.SavePrefs()
			game.SetPaused(paused)
			game.Notify(t.describe())
		})
	}()
}

// observe compares the measured step rate of the running scene with the prediction (called about
// twice per second). Only unthrottled runs count: a step limit or pause says nothing about cost.
func (t *tuning) observe(game *ui.Game) {
	if game.Scene != t.running {
		t.running, t.since = game.Scene, time.Now()
		return
	}
	sps := game.Runner.StepsPerSecond()
	if time.Since(t.since) < 5*time.Second || game.Runner.Paused() || game.Runner.TargetSPS() != 0 || sps <= 0 {
		return // the first seconds are not yet evolved, limited runs are not informative
	}
	predicted := perf.StepSeconds(t.cal, float64(game.N), perf.GasFraction(game.Scene))
	if predicted <= 0 {
		return
	}
	ratio := (1 / sps) / predicted
	if t.correction == nil {
		t.correction = map[*scene.Spec]float64{}
	}
	old, ok := t.correction[game.Scene]
	if !ok {
		old = 1
	}
	t.correction[game.Scene] = 0.9*old + 0.1*ratio
}
