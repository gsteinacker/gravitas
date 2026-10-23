package sph

import (
	"math"
	"testing"

	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/direct"
	"galaxy/internal/vec"
)

// gasSphere is a self-gravitating Plummer sphere of gas, made hot enough to be roughly supported
// by pressure.
func gasSphere(n int) *sim.State {
	s := sim.NewState(n)
	scene.AddPlummer(s, n, 1, 1, vec.Vec3{}, vec.Vec3{}, 9)
	for i := range s.Len() {
		s.Kind[i], s.U[i] = sim.Gas, 0.2
		s.VelX[i], s.VelY[i], s.VelZ[i] = 0.3*s.VelX[i], 0.3*s.VelY[i], 0.3*s.VelZ[i]
	}
	return s
}

func totalEnergy(m *sim.Simulation) float64 {
	e, _ := sim.TotalEnergy(m)
	return e
}

func TestAdiabaticGasConservesTotalEnergy(t *testing.T) {
	m := sim.New(gasSphere(2000), &direct.Solver{}, sim.Params{Dt: 0.002, Softening: 0.05})
	m.Hydro = &Solver{}
	m.Step()
	e0 := totalEnergy(m)
	for range 300 {
		m.Step()
	}
	if d := math.Abs((totalEnergy(m) - e0) / e0); d > 1e-2 {
		t.Errorf("relative energy change %.3g over 300 steps (adiabatic)", d)
	}
}

func TestCoolingAndStarFormation(t *testing.T) {
	s := gasSphere(2000)
	var mass0 float64
	for i := range s.Len() {
		s.U[i] = 1 // hot
		mass0 += s.Mass[i]
	}
	p := sim.Params{Dt: 0.002, Softening: 0.05, Gas: sim.GasParams{UFloor: 0.01, CoolTime: 0.02, SFDensity: 0.3, SFEfficiency: 0.5}}
	run := func() *sim.Simulation {
		m := sim.New(gasSphereCopy(s), &direct.Solver{}, p)
		m.Hydro = &Solver{}
		for range 400 {
			m.Step()
		}
		return m
	}
	a, b := run(), run()
	var mass, maxU float64
	stars := 0
	for i := range a.State.Len() {
		mass += a.State.Mass[i]
		if a.State.Kind[i] == sim.Star {
			stars++
		} else {
			maxU = math.Max(maxU, a.State.U[i])
		}
	}
	if math.Abs(mass-mass0) > 1e-12 {
		t.Errorf("mass changed by %.3g", mass-mass0)
	}
	if stars == 0 || stars != a.StarsFormed {
		t.Errorf("stars formed %d (counter %d), want > 0", stars, a.StarsFormed)
	}
	if maxU > 0.5 {
		t.Errorf("gas did not cool: max u = %.3f", maxU)
	}
	if b.StarsFormed != a.StarsFormed {
		t.Errorf("star formation not reproducible: %d vs %d", a.StarsFormed, b.StarsFormed)
	}
	t.Logf("formed %d stars from 2000 gas particles, max u %.3f", stars, maxU)
}

func gasSphereCopy(s *sim.State) *sim.State {
	c := sim.NewState(s.Len())
	for i := range s.Len() {
		c.Add(s.Body(i))
	}
	return c
}
