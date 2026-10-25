package scene

import (
	"math"
	"testing"

	"galaxy/internal/sim"
)

// TestUnitsMatchInfo: the length unit L = v·t of every scene with units agrees with its info line.
func TestUnitsMatchInfo(t *testing.T) {
	want := map[string]float64{ // pc per unit, from the info lines
		"Spiral galaxy": 2500, "Galaxy collision": 2500, "Star cluster": 1,
		"Black hole binary in a cluster": 1, "Local Group": 100_000, "S-stars around Sgr A*": 1000 / 206_264.8,
		"Horsehead Nebula": 1,
	}
	for _, p := range Presets {
		v, l, ok := p.Spec.Units()
		if !ok {
			t.Errorf("%s has no physical units", p.Name)
			continue
		}
		if w, known := want[p.Name]; known && math.Abs(l/w-1) > 0.02 {
			t.Errorf("%s: %.4g pc per unit (velocity unit %.4g km/s), want %.4g", p.Name, l, v, w)
		}
	}
}

func TestTemplateGalaxyInOtherUnits(t *testing.T) {
	var lg *Spec
	for _, p := range Presets {
		if p.Name == "Local Group" {
			lg = p.Spec
		}
	}
	s := sim.NewState(0)
	if !lg.AddTemplateGalaxy(s, 3000, 2e10, 7) {
		t.Fatal("Local Group has units")
	}
	mu, _ := lg.MassUnit()
	var m float64
	var lz float64
	for i := range s.Len() {
		m += s.Mass[i]
		lz += s.Mass[i] * (s.PosX[i]*s.VelY[i] - s.PosY[i]*s.VelX[i])
	}
	if math.Abs(m*mu/2e10-1) > 0.05 {
		t.Errorf("total mass %.3g Msun, want 2e10", m*mu)
	}
	if lz <= 0 {
		t.Errorf("disk must spin around +z, Lz = %v", lz)
	}
	if TemplateGalaxyMsun() < 1e11 {
		t.Errorf("template %.3g Msun", TemplateGalaxyMsun())
	}
}
