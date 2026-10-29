package sim

import (
	"math"
	"testing"
)

// youngCluster is a ball of gas with stars born at time 0 among it, plus old stars that never explode.
func youngCluster(young, old int) *State {
	s := denseBall(400, 0.5, 3)
	for i := range s.Len() {
		s.Kind[i], s.U[i] = Gas, 0.01
	}
	for k := range young + old {
		i := s.Add(Body{Pos: s.Body(k).Pos, Mass: 0.01, Kind: Star})
		if k < young {
			s.Born[i] = 0
		}
	}
	return s
}

func TestSupernovaeExplodeOnceAndBookEnergy(t *testing.T) {
	const young, old = 20, 10
	p := Params{Dt: 1e-3, Softening: 0.05, Gas: GasParams{SNEnergy: 2, SNDelayMin: 0.01, SNDelayMax: 0.05}}
	s := youngCluster(young, old)
	m := New(s, pairSolver{}, p)
	e0 := thermalEnergy(s) + exactEnergy(s, p)
	for range 100 { // t = 0.1, well beyond the latest explosion
		m.Step()
	}

	if m.Supernovae != young {
		t.Fatalf("%d supernovae, want %d (once per young population, none for old stars)", m.Supernovae, young)
	}
	if d := m.SNMass - young*0.01; math.Abs(d) > 1e-12 {
		t.Errorf("exploded mass off by %.3g", d)
	}
	want := 2 * young * 0.01
	if math.Abs(m.ExternalEnergy-want) > 1e-12 {
		t.Errorf("booked %.6g, want %.6g", m.ExternalEnergy, want)
	}
	// Without hydrodynamics the internal energy only changes by the explosions.
	e1 := thermalEnergy(s) + exactEnergy(s, p)
	if d := e1 - e0 - m.ExternalEnergy; math.Abs(d) > 1e-3*math.Abs(e0) {
		t.Errorf("energy changed by %.6g, booked %.6g", e1-e0, m.ExternalEnergy)
	}
}

func TestSupernovaDelayInRangeAndReproducible(t *testing.T) {
	var lo, hi float64 = math.Inf(1), 0
	for id := range uint64(2000) {
		d := SupernovaDelay(id, 3, 40)
		if d != SupernovaDelay(id, 3, 40) {
			t.Fatal("delay not reproducible")
		}
		lo, hi = min(lo, d), max(hi, d)
	}
	if lo < 3 || hi >= 40 || lo > 3.5 || hi < 39.5 {
		t.Errorf("delays span [%.2f, %.2f], want about [3, 40)", lo, hi)
	}
}

func TestNoSupernovaeWithoutEnergy(t *testing.T) {
	m := New(youngCluster(5, 0), pairSolver{}, Params{Dt: 1e-3, Softening: 0.05})
	for range 50 {
		m.Step()
	}
	if m.Supernovae != 0 || m.ExternalEnergy != 0 {
		t.Errorf("%d supernovae, %.3g energy without snEnergy", m.Supernovae, m.ExternalEnergy)
	}
}
