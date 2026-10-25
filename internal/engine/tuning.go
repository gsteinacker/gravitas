package engine

import (
	"time"

	"galaxy/internal/app"
	"galaxy/internal/perf"
	"galaxy/internal/prefs"
	"galaxy/internal/scene"
)

// Settings of the performance (ported from cmd/galaxy/tuning.go).

// StarChoices are the star counts offered in the settings; 0 = from the profile.
var StarChoices = []int{0, 10_000, 25_000, 50_000, 100_000, 200_000, 500_000, 1_000_000}

// GPUModes are the GPU settings in the order of the settings window.
var GPUModes = []string{prefs.GPUAuto, prefs.GPUOn, prefs.GPUOff}

// Settings returns the persistent settings (profile, stars, GPU, level).
func (e *Engine) Settings() prefs.Prefs { return *e.settings }

// SetStars fixes the stars of new scenes; 0 returns to the profile.
func (e *Engine) SetStars(n int) {
	e.settings.Stars = max(0, n)
	e.StarsFor = e.stars
	e.N = e.stars(e.Scene)
	e.SavePrefs()
}

// SetGPUMode chooses prefs.GPUAuto, GPUOn or GPUOff and applies it at once.
func (e *Engine) SetGPUMode(mode string) {
	e.settings.GPU = mode
	e.SetGPU(e.gpu())
	e.SavePrefs()
}

// Recalibrate measures the machine again while the simulation is paused (about 0.1–0.3 s) and
// applies the result to the next scene.
func (e *Engine) Recalibrate() {
	paused := e.Runner.Paused()
	e.Runner.SetPaused(true)
	time.Sleep(50 * time.Millisecond) // let the running step finish
	e.cal = perf.Calibrate(e.settings.GPU != prefs.GPUOff)
	e.settings.Calibration[e.hw.Key()] = e.cal
	e.correction = nil
	e.N = e.stars(e.Scene)
	e.SetGPU(e.gpu())
	e.SavePrefs()
	e.Runner.SetPaused(paused)
}

// Capacity is the star count the calibration allows for 30 steps per second.
func (e *Engine) Capacity() int { return perf.StarsFor(e.cal, perf.StepBudget) }

// observe compares the measured step rate of the running scene with the prediction (about twice
// per second) and corrects the star count of the scene for its next load. Only unthrottled runs
// count: a step limit or a pause says nothing about the cost.
func (e *Engine) observe() {
	if e.Scene != e.running {
		e.running, e.since = e.Scene, time.Now()
		return
	}
	sps := e.Runner.StepsPerSecond()
	if time.Since(e.since) < 5*time.Second || e.Runner.Paused() || e.Runner.TargetSPS() != 0 || sps <= 0 {
		return
	}
	predicted := perf.StepSeconds(e.cal, float64(e.N), perf.GasFraction(e.Scene))
	if predicted <= 0 {
		return
	}
	if e.correction == nil {
		e.correction = map[*scene.Spec]float64{}
	}
	old, ok := e.correction[e.Scene]
	if !ok {
		old = 1
	}
	e.correction[e.Scene] = 0.9*old + 0.1*(1/sps)/predicted
}

// LaunchSpeed is the speed a drag from (x0, y0) to (x1, y1) gives a launched black hole.
func (e *Engine) LaunchSpeed(x0, y0, x1, y1 float64) float64 {
	return app.LaunchVelocity(&e.Cam, x0, y0, x1, y1).Len()
}
