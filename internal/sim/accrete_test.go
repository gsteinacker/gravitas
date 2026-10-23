package sim

import (
	"math"
	"math/rand/v2"
	"testing"

	"galaxy/internal/vec"
)

// exactEnergy is the relativistic kinetic plus the exact pair-model potential energy.
func exactEnergy(s *State, p Params) float64 {
	return KineticEnergy(s, p.C) + PotentialEnergyParallel(s, p)
}

func totalMass(s *State) float64 {
	var m float64
	for i := range s.Len() {
		m += s.Mass[i]
	}
	return m
}

// accretionScene: a black hole with r_s = 0.08 (M = 1, c = 5), some stars inside the horizon,
// some outside, plus a second hole far away.
func accretionScene() (*State, Params) {
	p := Params{Dt: 1e-3, Softening: 0.05, C: 5}
	rng := rand.New(rand.NewPCG(1, 2))
	s := NewState(64)
	s.Add(Body{Vel: vec.Vec3{X: 0.1}, Mass: 1, Kind: BlackHole})
	s.Add(Body{Pos: vec.Vec3{X: 3}, Vel: vec.Vec3{Y: 0.2}, Mass: 0.3, Kind: BlackHole})
	for range 40 {
		r := 0.02 + 0.5*rng.Float64()
		dir := vec.Vec3{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()}
		s.Add(Body{
			Pos:  dir.Scale(r / dir.Len()),
			Vel:  vec.Vec3{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()},
			Mass: 0.001 + 0.002*rng.Float64(),
		})
	}
	return s, p
}

func TestAccretionConservesMassMomentumAndBooksEnergy(t *testing.T) {
	s, p := accretionScene()
	rs := SchwarzschildRadius(1, p.C)
	inside := 0
	for i := 2; i < s.Len(); i++ {
		if math.Sqrt(dist2(s, 0, i)) < rs {
			inside++
		}
	}
	if inside == 0 {
		t.Fatal("test scene has no star inside the horizon")
	}
	m := New(s, pairSolver{}, p)
	mass0, p0, e0 := totalMass(s), Momentum(s), exactEnergy(s, p)

	m.accrete()

	if got := s.Len(); got > 42-inside {
		t.Errorf("%d bodies left, want at most %d (%d inside r_s)", got, 42-inside, inside)
	}
	if d := totalMass(s) - mass0; math.Abs(d) > 1e-14 {
		t.Errorf("mass changed by %.3g", d)
	}
	if d := Momentum(s).Sub(p0).Len(); d > 1e-14 {
		t.Errorf("momentum changed by %.3g", d)
	}
	if e1 := exactEnergy(s, p); math.Abs(e1-e0-m.ExternalEnergy) > 1e-12*math.Abs(e0) {
		t.Errorf("energy changed by %.6g, booked %.6g", e1-e0, m.ExternalEnergy)
	}
	for i := range s.Len() {
		if s.Kind[i] == Star && math.Sqrt(dist2(s, 0, i)) < SchwarzschildRadius(s.Mass[0], p.C) {
			t.Errorf("star %d still inside the horizon", i)
		}
	}
}

func TestMomentumConservedThroughHeavyAccretion(t *testing.T) {
	s, p := accretionScene()
	m := New(s, pairSolver{}, p)
	p0 := Momentum(s)
	for range 50 {
		m.Step()
	}
	if d := Momentum(s).Sub(p0).Len(); d > 1e-12 {
		t.Errorf("momentum changed by %.3g through accretion steps", d)
	}
}

func TestBlackHolesMergeWhenHorizonsTouch(t *testing.T) {
	p := Params{Dt: 1e-3, Softening: 0.05, C: 5} // r_s = 0.08 per unit mass
	s := NewState(3)
	s.Add(Body{Pos: vec.Vec3{X: -0.05}, Vel: vec.Vec3{Y: 1}, Mass: 1, Kind: BlackHole})
	s.Add(Body{Pos: vec.Vec3{X: 0.05}, Vel: vec.Vec3{Y: -1}, Mass: 0.5, Kind: BlackHole})
	s.Add(Body{Pos: vec.Vec3{X: 2}, Mass: 0.01})
	m := New(s, pairSolver{}, p)
	e0 := exactEnergy(s, p)
	m.accrete()
	if s.Len() != 2 || s.Kind[0] != BlackHole || s.Mass[0] != 1.5 {
		t.Fatalf("want one merged hole of mass 1.5 plus the star, got %d bodies, masses %v", s.Len(), s.Mass)
	}
	if b := s.Body(0); math.Abs(b.Pos.X+0.05/3) > 1e-15 || math.Abs(b.Vel.Y-0.5/1.5) > 1e-15 {
		t.Errorf("merged hole at %v with velocity %v, want center of mass and momentum", b.Pos, b.Vel)
	}
	if e1 := exactEnergy(s, p); math.Abs(e1-e0-m.ExternalEnergy) > 1e-12*math.Abs(e0) {
		t.Errorf("energy changed by %.6g, booked %.6g", e1-e0, m.ExternalEnergy)
	}
}

func TestNoAccretionWithoutSpeedOfLight(t *testing.T) {
	s, p := accretionScene()
	p.C = 0
	m := New(s, pairSolver{}, p)
	m.Step()
	if s.Len() != 42 {
		t.Errorf("bodies absorbed although c = 0: %d left", s.Len())
	}
}

func TestSubstepsFollowClosestEncounter(t *testing.T) {
	p := Params{Dt: 0.004, Softening: 0.01, C: 0}
	build := func(r float64) *Simulation {
		s := NewState(3)
		s.Add(Body{Mass: 2, Kind: BlackHole})
		s.Add(Body{Pos: vec.Vec3{X: r}, Mass: 1e-6})
		s.Add(Body{Pos: vec.Vec3{X: 50}, Mass: 1e-6})
		return New(s, pairSolver{}, p)
	}
	far, close := build(5), build(0.05)
	far.Step()
	close.Step()
	if far.Substeps != 1 {
		t.Errorf("distant star: %d substeps, want 1", far.Substeps)
	}
	// τ = √(r³/M) ≈ 0.0079 at r = 0.05 → 20·dt/τ ≈ 10 → next power of two.
	if close.Substeps != 16 {
		t.Errorf("close star: %d substeps, want 16", close.Substeps)
	}
	if !close.isNear[1] || close.isNear[2] {
		t.Errorf("near flags %v, want only the close star near", close.isNear)
	}
}
