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
	hw       perf.Hardware
	cal      prefs.Calibration
}

// newTuning probes the machine and calibrates it once; the result is cached in the settings.
func newTuning(settings *prefs.Prefs) *tuning {
	t := &tuning{settings: settings, hw: perf.Probe()}
	if c, ok := settings.Calibration[t.hw.Key()]; ok && c.SecPerNL > 0 {
		t.cal = c
		return t
	}
	t0 := time.Now()
	t.calibrate()
	log.Printf("calibrated %s in %v: %d stars for 30 steps/s", t.hw, time.Since(t0).Round(time.Millisecond), perf.StarsFor(t.cal, perf.StepBudget))
	return t
}

func (t *tuning) calibrate() {
	t.cal = perf.Calibrate(t.settings.GPU != prefs.GPUOff)
	if t.settings.Calibration == nil {
		t.settings.Calibration = map[string]prefs.Calibration{}
	}
	t.settings.Calibration[t.hw.Key()] = t.cal
}

// stars is the star count of scene sp.
func (t *tuning) stars(sp *scene.Spec) int {
	if t.settings.Stars > 0 {
		return t.settings.Stars
	}
	return perf.ProfileByID(t.settings.Profile).SceneStars(t.cal, sp, defaultStars)
}

// gpu reports whether the tree walk should run on the GPU.
func (t *tuning) gpu() bool {
	switch t.settings.GPU {
	case prefs.GPUOff:
		return false
	case prefs.GPUOn:
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
}

// recalibrate measures the machine again while the simulation is paused (settings window).
func recalibrate(game *ui.Game, t *tuning) {
	paused := game.Runner.Paused()
	game.SetPaused(true)
	game.Notify(i18n.T("Leistung wird gemessen …", "Measuring performance…"))
	gpuAllowed := t.settings.GPU != prefs.GPUOff
	go func() {
		time.Sleep(200 * time.Millisecond) // let the running step finish
		cal := perf.Calibrate(gpuAllowed)
		game.Post(func(*app.Controller) {
			t.cal = cal
			t.settings.Calibration[t.hw.Key()] = cal
			mac.SetHardware(t.describe())
			t.apply(game)
			game.SavePrefs()
			game.SetPaused(paused)
			game.Notify(t.describe())
		})
	}()
}
