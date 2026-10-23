package sim

import (
	"math"
	"testing"

	"galaxy/internal/vec"
)

// constSolver applies the same acceleration to every body.
type constSolver struct{ ax float64 }

func (c constSolver) Accelerations(s *State, _ Params) {
	for i := range s.Len() {
		s.AccX[i], s.AccY[i], s.AccZ[i] = c.ax, 0, 0
	}
}

func TestSpeedStaysBelowCUnderExtremeAcceleration(t *testing.T) {
	const c, a, dt = 1.0, 1e6, 0.01
	s := NewState(1)
	s.Add(Body{Mass: 1})
	m := New(s, constSolver{a}, Params{Dt: dt, C: c})
	prevX := 0.0
	for step := 1; step <= 1000; step++ {
		m.Step()
		if v := (s.PosX[0] - prevX) / dt; v >= c {
			t.Fatalf("step %d: coordinate speed %.15f >= c", step, v)
		}
		prevX = s.PosX[0]
		if u := s.VelX[0]; math.Abs(u-a*dt*float64(step)) > 1e-6*u {
			t.Fatalf("step %d: proper velocity %v, want a·t = %v", step, u, a*dt*float64(step))
		}
	}
	if v := CoordinateVelocity(s.Body(0).Vel, c).Len(); v >= c || v < 0.999999*c {
		t.Errorf("final speed %v should approach but stay below c", v)
	}
}

func TestNewtonianLimit(t *testing.T) {
	run := func(c float64) *State {
		s := NewState(2)
		s.Add(Body{Pos: vec.Vec3{X: -0.5}, Vel: vec.Vec3{Y: -0.5}, Mass: 0.5})
		s.Add(Body{Pos: vec.Vec3{X: 0.5}, Vel: vec.Vec3{Y: 0.5}, Mass: 0.5})
		m := New(s, pairSolver{}, Params{Dt: 1e-3, C: c})
		for range 2000 {
			m.Step()
		}
		return s
	}
	a, b := run(0), run(1e9)
	if d := a.Body(0).Pos.Sub(b.Body(0).Pos).Len(); d > 1e-9 {
		t.Errorf("c = 1e9 deviates from Newtonian by %.3g", d)
	}
}

func TestRelativisticOrbitConservesEnergy(t *testing.T) {
	// Light test body around a heavy one at v ≈ 0.4c: the relativistic Hamiltonian must be
	// conserved by the leapfrog, the Newtonian energy must not (sanity check of the test).
	const c = 2.5
	s := NewState(2)
	s.Add(Body{Mass: 1})
	s.Add(Body{Pos: vec.Vec3{X: 1}, Vel: ProperVelocity(vec.Vec3{Y: 1.0}, c), Mass: 1e-6})
	m := New(s, pairSolver{}, Params{Dt: 1e-3, C: c})
	energy := func(c float64) float64 {
		_, u := Energy(s, 0)
		return KineticEnergy(s, c) + u
	}
	e0, n0 := energy(c), energy(0)
	var maxRel, maxNewton float64
	for i := range 60000 { // ~10 orbits
		m.Step()
		if i%100 == 0 {
			maxRel = math.Max(maxRel, math.Abs((energy(c)-e0)/e0))
			maxNewton = math.Max(maxNewton, math.Abs((energy(0)-n0)/n0))
		}
	}
	if maxRel > 1e-5 {
		t.Errorf("relativistic energy drift %.3g > 1e-5", maxRel)
	}
	if maxNewton < 100*maxRel {
		t.Errorf("Newtonian energy should not be conserved here: %.3g vs %.3g", maxNewton, maxRel)
	}
}

func TestProperVelocityRoundTripAndCap(t *testing.T) {
	const c = 3.0
	v := vec.Vec3{X: 1, Y: -2}
	if got := CoordinateVelocity(ProperVelocity(v, c), c); got.Sub(v).Len() > 1e-12 {
		t.Errorf("round trip: %v, want %v", got, v)
	}
	if got := CoordinateVelocity(ProperVelocity(vec.Vec3{X: 10}, c), c).Len(); math.Abs(got-0.99*c) > 1e-12 {
		t.Errorf("speed above c should be capped to 0.99c, got %v", got)
	}
	if got := ProperVelocity(v, 0); got != v {
		t.Errorf("c = 0 must be Newtonian, got %v", got)
	}
}
