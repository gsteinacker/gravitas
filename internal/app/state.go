package app

import (
	"galaxy/internal/learn"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/view"
)

// State is a copy of everything a native menu or inspector shows. The controller publishes it
// after each update; other goroutines read it with State.
type State struct {
	Paused        bool
	Speed         int // index into SpeedLevels
	C             float64
	Gravity       sim.GravityMode
	DarkMatterOff bool
	Tool          Tool
	Mass          float64
	FX            view.Effects
	ShowHUD       bool
	ShowHelp      bool
	Explain       bool
	Level         learn.Level
	Scene         int // index of the built-in scene, -1 for others
	SceneName     string
	Sim           SimState
	StepsPerSec   float64
	FPS           float64
}

// Publish stores the current state for State; call it at the end of each update.
func (c *Controller) Publish() {
	s := &State{
		Paused: c.Runner.Paused(), Speed: c.Speed, C: c.sim.C, Gravity: c.sim.Gravity,
		DarkMatterOff: c.sim.DarkMatterOff, Tool: c.Tool, Mass: c.Mass, FX: c.FX,
		ShowHUD: c.ShowHUD, ShowHelp: c.ShowHelp, Explain: c.Explain, Level: c.Level,
		Scene: -1, Sim: c.sim, StepsPerSec: c.Runner.StepsPerSecond(), FPS: c.FPS,
	}
	if c.Scene != nil {
		s.SceneName = c.Scene.Name
		for i, p := range scene.Presets {
			if p.Spec == c.Scene {
				s.Scene = i
			}
		}
	}
	c.state.Store(s)
}

// State returns the last published state; safe to call from any goroutine.
func (c *Controller) State() State { return *c.state.Load() }
