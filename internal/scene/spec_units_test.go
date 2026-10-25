package scene

import (
	"math"
	"testing"
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
