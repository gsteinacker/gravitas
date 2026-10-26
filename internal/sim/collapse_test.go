package sim

import (
	"math"
	"math/rand/v2"
	"testing"

	"galaxy/internal/vec"
)

// denseBall is a Plummer-like ball of n bodies (total mass 1, scale a) with small velocities.
func denseBall(n int, a float64, seed uint64) *State {
	rng := rand.New(rand.NewPCG(seed, 9))
	s := NewState(n + 1)
	for range n {
		x := rng.Float64()
		r := a / math.Sqrt(math.Pow(x, -2.0/3)-1)
		dir := vec.Vec3{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()}
		vel := vec.Vec3{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()}.Scale(0.05)
		s.Add(Body{Pos: dir.Scale(math.Min(r, 10*a) / dir.Len()), Vel: vel, Mass: 1 / float64(n)})
	}
	return s
}

func TestCollapseFormsHoleAndConserves(t *testing.T) {
	p := Params{Dt: 1e-3, Softening: 0.02, C: 3}
	s := denseBall(400, 0.05, 1)
	m := New(s, pairSolver{}, p)
	mass0, p0, l0, e0 := totalMass(s), Momentum(s), angularMomentum(s), exactEnergy(s, p)

	m.collapse()

	if m.HolesFormed != 1 {
		t.Fatalf("%d holes formed, want 1", m.HolesFormed)
	}
	holes := 0
	for _, k := range s.Kind {
		if k == BlackHole {
			holes++
		}
	}
	if holes != 1 || s.Len() >= 400 {
		t.Errorf("%d holes, %d bodies left", holes, s.Len())
	}
	if d := totalMass(s) - mass0; math.Abs(d) > 1e-12 {
		t.Errorf("mass changed by %.3g", d)
	}
	if d := Momentum(s).Sub(p0).Len(); d > 1e-12 {
		t.Errorf("momentum changed by %.3g", d)
	}
	if d := angularMomentum(s).Sub(l0).Len(); d > 1e-12 {
		t.Errorf("angular momentum changed by %.3g (spin below the cap expected)", d)
	}
	if e1 := exactEnergy(s, p); math.Abs(e1-e0-m.ExternalEnergy) > 1e-9*math.Abs(e0) {
		t.Errorf("energy changed by %.6g, booked %.6g", e1-e0, m.ExternalEnergy)
	}
}

func TestNoCollapseWithoutResolutionOrMass(t *testing.T) {
	// Too few bodies: heavy enough, but not resolved by minCollapseBodies.
	s := NewState(10)
	for i := range 10 {
		s.Add(Body{Pos: vec.Vec3{X: 0.001 * float64(i)}, Mass: 1})
	}
	m := New(s, pairSolver{}, Params{Dt: 1e-3, Softening: 0.02, C: 3})
	m.collapse()
	if m.HolesFormed != 0 {
		t.Error("an unresolved clump collapsed")
	}
	// The same dense ball with the real speed of light of a parsec scene (c ≈ 14500) stays.
	s = denseBall(400, 0.05, 1)
	m = New(s, pairSolver{}, Params{Dt: 1e-3, Softening: 0.02, C: 14500})
	m.collapse()
	if m.HolesFormed != 0 {
		t.Error("collapse with the real speed of light")
	}
}
