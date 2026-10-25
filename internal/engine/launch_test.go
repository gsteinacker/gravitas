package engine

import (
	"math"
	"testing"
	"time"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// kepler is a plan around a point mass M at the origin, disk in the xy plane, start at (R, 0, 0).
func kepler(m, r float64) *Engine {
	e := &Engine{}
	e.launch = Launch{
		Active: true, Normal: vec.Vec3{Z: 1}, diskNormal: vec.Vec3{Z: 1}, Foot: vec.Vec3{X: r},
		field: []vec.Vec3{{}}, fieldMass: []float64{m},
	}
	e.planLaunch()
	return e
}

func TestLaunchReferencesAroundPointMass(t *testing.T) {
	e := kepler(4, 2)
	l := e.LaunchPlan()
	if math.Abs(l.Circular-math.Sqrt(4.0/2)) > 1e-9 || math.Abs(l.Escape-math.Sqrt(2*4.0/2)) > 1e-9 {
		t.Errorf("circular %v, escape %v; want √2, 2", l.Circular, l.Escape)
	}
	e.LaunchPreset(PresetCircular)
	l = e.LaunchPlan()
	// Prograde for a disk with angular momentum along +z: (R,0,0) moves along +y.
	if v := l.Velocity(); math.Abs(v.Y-l.Circular) > 1e-9 || math.Abs(v.X) > 1e-9 || math.Abs(v.Z) > 1e-9 {
		t.Errorf("circular velocity %v", v)
	}
	// The preview spans 1.5 orbits: after one period (step 400 of 600) it is back at the start.
	if d := l.path[previewSteps*2/3].Sub(l.path[0]).Len(); d > 0.02*2 {
		t.Errorf("circular orbit does not close: %v away after one period", d)
	}
	for _, p := range l.path {
		if r := p.Len(); math.Abs(r-2) > 0.02 {
			t.Fatalf("circular orbit leaves the radius: %v", r)
		}
	}
	if b := e.LaunchPlan().Impact; math.Abs(b-2) > 1e-9 {
		t.Errorf("tangential start: impact parameter %v, want the radius 2", b)
	}
	e.LaunchPreset(PresetCenter)
	if b := e.LaunchPlan().Impact; b > 1e-9 {
		t.Errorf("aimed at the center: impact parameter %v", b)
	}
	e.LaunchPreset(PresetEscape)
	if v := e.LaunchPlan().Velocity(); math.Abs(v.X-2) > 1e-9 {
		t.Errorf("escape velocity %v, want radially outward with 2", v)
	}
	e.SetLaunch("height", 0.5)
	if p := e.LaunchPlan().Position(); p != (vec.Vec3{X: 2, Z: 0.5}) {
		t.Errorf("height: position %v", p)
	}
	e.SetLaunch("elevation", math.Pi/2)
	if v := e.LaunchPlan().Velocity(); math.Abs(v.Z-e.LaunchPlan().Speed) > 1e-9 {
		t.Errorf("elevation 90°: velocity %v", v)
	}
}

func TestLaunchInScene(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := New(0, 5000) // spiral galaxy, 1 unit = 2.5 kpc, 185 km/s
	defer e.Close()
	e.Runner.SetPaused(true)
	e.SetViewport(1440, 900)
	time.Sleep(50 * time.Millisecond)
	e.Cam.Pitch = math.Pi / 2   // top-down
	e.BeginLaunch(720+300, 450) // right of the center
	l := e.LaunchPlan()
	if !l.Active || l.Radius <= 0 {
		t.Fatalf("plan %+v", l)
	}
	// The disk is the xy plane (up to the scatter of the stars): the start lies in it.
	if math.Abs(l.Normal.Z) < 0.95 || math.Abs(l.Position().Sub(l.Center).Dot(l.Normal)) > 1e-6 {
		t.Errorf("normal %v, height above plane %v", l.Normal, l.Position().Sub(l.Center).Dot(l.Normal))
	}
	// A galaxy like the Milky Way rotates at roughly 150–300 km/s in its disk.
	kmPerS, _, _ := e.Scene.Units()
	if vc := l.Circular * kmPerS; vc < 80 || vc > 400 || l.Escape <= l.Circular {
		t.Errorf("circular %.0f km/s, escape %.0f km/s", vc, l.Escape*kmPerS)
	}
	e.LaunchPreset(PresetCircular)
	before := 0
	e.Runner.View(func(sn *sim.Snapshot) { before = sn.Len() })
	e.GoLaunch()
	if e.LaunchPlan().Active {
		t.Error("launch must end the plan")
	}
	e.Runner.SetPaused(false)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		n := 0
		e.Runner.View(func(sn *sim.Snapshot) { n = sn.Len() })
		if n == before+1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("black hole not added")
}
