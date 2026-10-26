package sim

import (
	"math"
	"testing"

	"galaxy/internal/vec"
)

func TestKerrRadii(t *testing.T) {
	const m, c = 1.0, 1.0 // r_g = 1
	cases := []struct {
		name      string
		got, want float64
		tolerance float64
	}{
		{"horizon a=0", KerrHorizon(m, 0, c), 2, 1e-12},
		{"horizon a=1", KerrHorizon(m, 1, c), 1, 1e-12},
		{"ISCO a=0", KerrISCO(m, 0, c, true), 6, 1e-9},
		{"ISCO a=1 prograde", KerrISCO(m, 1, c, true), 1, 1e-6},
		{"ISCO a=1 retrograde", KerrISCO(m, 1, c, false), 9, 1e-6},
		{"ISCO a=0.9 prograde", KerrISCO(m, 0.9, c, true), 2.3209, 1e-3},
		{"ISCO a=0.9 retrograde", KerrISCO(m, 0.9, c, false), 8.7174, 1e-3},
		{"photon orbit a=0", KerrPhotonOrbit(m, 0, c, true), 3, 1e-12},
		{"photon orbit a=1 prograde", KerrPhotonOrbit(m, 1, c, true), 1, 1e-9},
		{"photon orbit a=1 retrograde", KerrPhotonOrbit(m, 1, c, false), 4, 1e-9},
	}
	for _, k := range cases {
		if math.Abs(k.got-k.want) > k.tolerance {
			t.Errorf("%s: %.6g, want %.6g", k.name, k.got, k.want)
		}
	}
	if r := KerrISCO(m, 0, 2, true); math.Abs(r-1.5) > 1e-9 {
		t.Errorf("ISCO scales with 1/c²: %v", r)
	}
}

// angularMomentum is Σ m r × u plus the spins of the holes.
func angularMomentum(s *State) vec.Vec3 {
	var l vec.Vec3
	for i := range s.Len() {
		x, y, z := cross(s.Mass[i], s.PosX[i], s.PosY[i], s.PosZ[i], s.VelX[i], s.VelY[i], s.VelZ[i])
		l = l.Add(vec.Vec3{X: x, Y: y, Z: z}).Add(s.Spin(i))
	}
	return l
}

// TestAccretionConservesAngularMomentum: what a hole swallows ends up in its spin.
func TestAccretionConservesAngularMomentum(t *testing.T) {
	s, p := accretionScene()
	s.SpinZ[0] = 0.05 // a* = 0.25
	m := New(s, pairSolver{}, p)
	l0 := angularMomentum(s)
	n0 := s.Len()
	m.accrete()
	if s.Len() == n0 {
		t.Fatal("nothing was swallowed")
	}
	if d := angularMomentum(s).Sub(l0).Len(); d > 1e-14 {
		t.Errorf("angular momentum changed by %.3g", d)
	}
	if a := SpinParameter(s.Spin(0), s.Mass[0], p.C); a <= 0 || a > MaxSpin {
		t.Errorf("spin a* = %v", a)
	}
}

// TestSpinIsCappedAtTheThorneLimit: a hole swallowing a lot of angular momentum stops at a* = 0.998.
func TestSpinIsCappedAtTheThorneLimit(t *testing.T) {
	p := Params{Dt: 1e-3, Softening: 0.05, C: 5}
	s := NewState(2)
	s.Add(Body{Mass: 1, Kind: BlackHole, Spin: vec.Vec3{Z: 0.19}})
	s.Add(Body{Pos: vec.Vec3{X: 0.05}, Vel: vec.Vec3{Y: 50}, Mass: 0.1})
	m := New(s, pairSolver{}, p)
	m.accrete()
	if a := SpinParameter(s.Spin(0), s.Mass[0], p.C); math.Abs(a-MaxSpin) > 1e-12 {
		t.Errorf("a* = %v, want %v", a, MaxSpin)
	}
}

// TestAccretionRateTracksSteadyFeeding: a hole swallowing q per step shows Ṁ = q/dt after a while
// and the rate fades once the feeding stops (T-065).
func TestAccretionRateTracksSteadyFeeding(t *testing.T) {
	p := Params{Dt: 0.01, Softening: 0.05, C: 5}
	s := NewState(1)
	h := s.Add(Body{Mass: 1, Kind: BlackHole})
	m := New(s, pairSolver{}, p)
	id := s.ID[h]
	const q = 1e-4
	for range 500 {
		i := s.Add(Body{Pos: vec.Vec3{X: 1e-3}, Mass: q})
		m.recordAccretion(h, i)
		s.Alive[i] = false
		s.Compact()
		m.decayAccretion(p.Dt)
	}
	if got, want := m.AccretionRate(id), q/p.Dt; math.Abs(got/want-1) > 0.05 {
		t.Errorf("steady rate %.4g, want %.4g", got, want)
	}
	for range 500 {
		m.decayAccretion(p.Dt)
	}
	if r := m.AccretionRate(id); r > 1e-3*q/p.Dt {
		t.Errorf("rate did not fade: %.3g", r)
	}
}
