package sim

import (
	"context"
	"math"
	"testing"
	"time"

	"galaxy/internal/vec"
)

func waitFor(t *testing.T, r *Runner, cond func(*Snapshot) bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ok := false
		r.View(func(sn *Snapshot) { ok = cond(sn) })
		if ok {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not reached")
}

func TestPausedRunnerExecutesSingleStepsAndAdds(t *testing.T) {
	s := NewState(2)
	s.Add(Body{Mass: 1})
	m := New(s, nopSolver{}, Params{Dt: 0.1})
	r := NewRunner(m)
	r.SetPaused(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	r.Do(func(m *Simulation) { m.Step() })
	waitFor(t, r, func(sn *Snapshot) bool { return sn.Steps == 1 })

	r.Do(func(m *Simulation) {
		m.AddBody(Body{Pos: vec.Vec3{X: 1}, Vel: vec.Vec3{Y: 2}, Mass: 3, Kind: BlackHole})
	})
	waitFor(t, r, func(sn *Snapshot) bool { return sn.Len() == 2 && sn.Kind[1] == BlackHole })

	time.Sleep(50 * time.Millisecond)
	r.View(func(sn *Snapshot) {
		if sn.Steps != 1 {
			t.Errorf("paused runner kept stepping: %d steps", sn.Steps)
		}
	})
}

func TestRunnerReportsEnergyDriftAndRebaselines(t *testing.T) {
	// Two bodies on a circular orbit with the exact solver substitute below.
	s := NewState(2)
	s.Add(Body{Pos: vec.Vec3{X: -0.5}, Vel: vec.Vec3{Y: -0.5}, Mass: 0.5})
	s.Add(Body{Pos: vec.Vec3{X: 0.5}, Vel: vec.Vec3{Y: 0.5}, Mass: 0.5})
	m := New(s, pairSolver{}, Params{Dt: 1e-3})
	r := NewRunner(m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	deadline := time.Now().Add(3 * time.Second)
	for d, _ := r.EnergyDrift(); math.IsNaN(d) && time.Now().Before(deadline); d, _ = r.EnergyDrift() {
		time.Sleep(10 * time.Millisecond)
	}
	if d, q := r.EnergyDrift(); math.IsNaN(d) || math.Abs(d) > 1e-6 || q != EnergyExact {
		t.Fatalf("drift = %v (quality %v), want small and exact", d, q)
	}
	epoch := m.Epoch
	r.Do(func(m *Simulation) { m.AddBody(Body{Pos: vec.Vec3{Z: 50}, Mass: 0.1}) })
	waitFor(t, r, func(sn *Snapshot) bool { return sn.Len() == 3 })
	r.Do(func(m *Simulation) {
		if m.Epoch == epoch {
			t.Error("adding a body must start a new energy epoch")
		}
	})
}

// pairSolver is an exact star–star solver for tests inside package sim (which cannot import
// direct). Like every Solver it leaves pairs involving holes to the integrator.
type pairSolver struct{}

func (pairSolver) Accelerations(s *State, p Params) {
	n := s.Len()
	for i := range n {
		s.AccX[i], s.AccY[i], s.AccZ[i] = 0, 0, 0
		if s.Kind[i] == BlackHole {
			continue
		}
		for j := range n {
			if i == j || s.Kind[j] == BlackHole {
				continue
			}
			dx, dy, dz := s.PosX[j]-s.PosX[i], s.PosY[j]-s.PosY[i], s.PosZ[j]-s.PosZ[i]
			r2 := dx*dx + dy*dy + dz*dz + p.Softening*p.Softening
			f := s.Mass[j] / (r2 * math.Sqrt(r2))
			s.AccX[i] += f * dx
			s.AccY[i] += f * dy
			s.AccZ[i] += f * dz
		}
	}
}

func TestSlowRendererDoesNotBlockStepping(t *testing.T) {
	s := NewState(1)
	s.Add(Body{Mass: 1})
	r := NewRunner(New(s, nopSolver{}, Params{Dt: 1e-3}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	var before, after int
	r.View(func(sn *Snapshot) {
		before = sn.Steps
		time.Sleep(200 * time.Millisecond) // a very slow frame
	})
	r.View(func(sn *Snapshot) { after = sn.Steps })
	if after-before < 100 {
		t.Errorf("only %d steps during a 200 ms frame; rendering blocks the simulation", after-before)
	}
}
