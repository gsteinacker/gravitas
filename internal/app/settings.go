package app

import (
	"maps"
	"math"

	"galaxy/internal/i18n"
	"galaxy/internal/learn"
	"galaxy/internal/prefs"
	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
)

// Absolute setters for widgets such as sliders; the actions above step relative to the current
// value.

// SetC sets the speed of light, clamped to [CMin, CMax]; ignored in a Newtonian run.
func (c *Controller) SetC(v float64) {
	v = math.Max(CMin, math.Min(CMax, v))
	if c.sim.C > 0 {
		c.sim.C = v
	}
	c.Runner.Do(func(m *sim.Simulation) {
		if m.Params.C > 0 {
			m.Params.C = v
			m.Epoch++ // kinetic energy depends on c
		}
	})
}

// SetSpeed selects SpeedLevels[i].
func (c *Controller) SetSpeed(i int) { c.Speed = max(0, min(len(SpeedLevels)-1, i)) }

// SetPaused pauses or resumes the simulation.
func (c *Controller) SetPaused(p bool) { c.Runner.SetPaused(p) }

// SetDarkMatterOff switches the gravity of dark matter off (test particles) or on.
func (c *Controller) SetDarkMatterOff(off bool) {
	c.sim.DarkMatterOff = off
	c.Runner.Do(func(m *sim.Simulation) { m.SetDarkMatterOff(off) })
}

// Theta bounds of the Barnes-Hut opening angle offered to the user; see perf-log T-028/T-034.
const (
	ThetaMin = 0.5
	ThetaMax = 1.0
)

// SetTheta sets the user's opening angle of a Barnes-Hut solver; the governor may raise the
// applied one (ApplyTheta).
func (c *Controller) SetTheta(theta float64) {
	c.ThetaUser = math.Max(ThetaMin, math.Min(ThetaMax, theta))
	c.ApplyTheta(c.ThetaUser)
}

// ApplyTheta changes the opening angle in use without changing the user's choice.
func (c *Controller) ApplyTheta(theta float64) {
	theta = math.Max(ThetaMin, math.Min(ThetaMax, theta))
	c.Theta = theta
	c.Runner.Do(func(m *sim.Simulation) {
		if s, ok := m.Solver.(*bh.Solver); ok {
			s.Theta = theta
		}
	})
}

// SetGPU runs the tree walk on the GPU (if available) or the CPU.
func (c *Controller) SetGPU(on bool) {
	on = on && bh.GPUAvailable() == nil
	c.GPU = on
	c.Runner.Do(func(m *sim.Simulation) {
		if s, ok := m.Solver.(*bh.Solver); ok {
			s.GPU = on
		}
	})
}

// SetLevel selects the audience of the learning mode and remembers it.
func (c *Controller) SetLevel(l learn.Level) {
	c.Level = l
	if c.Prefs != nil {
		c.Prefs.Level = int(l)
		c.SavePrefs()
	}
}

// SavePrefs writes the settings in the background.
func (c *Controller) SavePrefs() {
	if c.Prefs == nil {
		return
	}
	p := *c.Prefs
	p.Calibration = maps.Clone(p.Calibration)
	path := c.PrefsPath
	go func() {
		if err := p.Save(path); err != nil {
			c.Notify(i18n.T("Einstellungen: ", "settings: ") + err.Error())
		}
	}()
}

// ApplyPrefs takes over the settings that act on a running app.
func (c *Controller) ApplyPrefs(p *prefs.Prefs) {
	c.Prefs = p
	c.Level = learn.Level(max(0, min(int(learn.Physics), p.Level)))
}
