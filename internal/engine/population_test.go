package engine

import (
	"math"
	"testing"

	"galaxy/internal/app"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
)

// TestPopulationOnlyForGalaxies checks that particles of galaxy scenes are drawn as populations
// with star-forming gas, while star clusters keep single stars.
func TestPopulationOnlyForGalaxies(t *testing.T) {
	specs, err := scene.Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, sp := range specs {
		e := &Engine{Controller: &app.Controller{Scene: sp}}
		s, _ := sp.Build(20000)
		var mass float64
		var n int
		for i := range s.Len() {
			if s.Kind[i] == sim.Star {
				mass += s.Mass[i]
				n++
			}
		}
		if n == 0 {
			continue
		}
		p := e.population(mass / float64(n))
		msun, ok := sp.MassUnit()
		galaxy := ok && mass/float64(n)*msun >= minPopulationMsun
		if galaxy != (p.Radius > 0) {
			t.Errorf("%s: mean particle %.3g Msun, population radius %v", sp.Name, mass/float64(n)*msun, p.Radius)
		}
		if p.Radius > 0 && sp.Params.Gas.SFDensity > 0 && p.SFLight <= 0 {
			t.Errorf("%s: star-forming gas without light", sp.Name)
		}
	}
}

func TestStarFormingLight(t *testing.T) {
	p := Population{Radius: 0.06, SFDensity: 0.002, SFLight: 10}
	if w := p.starFormingLight(0.002, 0); w != 0 {
		t.Errorf("at the threshold: %v, want 0 (no jump)", w)
	}
	if w := p.starFormingLight(0.0021, 0); w <= 0 || w > 0.05 {
		t.Errorf("just above the threshold: %v, want small and positive", w)
	}
	if w := p.starFormingLight(0.1, 0.6); w != 0 {
		t.Errorf("ionized gas: %v, want 0", w)
	}
	// Above twice the threshold the light grows as √ρ (Kennicutt–Schmidt per mass) up to the cap.
	w1, w2 := p.starFormingLight(0.01, 0), p.starFormingLight(0.04, 0)
	if math.Abs(w2/w1-2) > 1e-6 {
		t.Errorf("light ratio %v for 4× density, want 2", w2/w1)
	}
	if w := p.starFormingLight(1e6, 0); w != maxStarFormingLight {
		t.Errorf("cap: %v", w)
	}
}

func TestYoungBoost(t *testing.T) {
	if a, s := youngBoost(-1, 5); a != 1 || s != 1 {
		t.Errorf("initial star: %v %v", a, s)
	}
	if a, s := youngBoost(5, 5); a != 1 || s != 1 {
		t.Errorf("old star: %v %v", a, s)
	}
	// A newborn population is about ten times brighter per mass (alpha × size²).
	if a, s := youngBoost(0, 5); a*s*s < 8 || a*s*s > 15 {
		t.Errorf("newborn: light factor %v", a*s*s)
	}
}
