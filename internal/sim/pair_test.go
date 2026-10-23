package sim

import (
	"math"
	"testing"
)

func TestPairForceIsGradientOfPotential(t *testing.T) {
	const m, eps2, h = 0.7, 0.01 * 0.01, 1e-7
	for _, rs := range []float64{0, 0.05} {
		for _, r := range []float64{0.06, 0.074, 0.08, 0.15, 0.4, 2} { // below and above 1.5 rs
			dU := (PairPotential(1, m, (r+h)*(r+h), eps2, rs) - PairPotential(1, m, (r-h)*(r-h), eps2, rs)) / (2 * h)
			if f := PairForce(m, r*r, eps2, rs) * r; math.Abs(f-dU) > 1e-6*math.Abs(f) {
				t.Errorf("rs=%v r=%v: force %.9g, dU/dr %.9g", rs, r, f, dU)
			}
		}
	}
}

func TestPairPotentialLimits(t *testing.T) {
	if got, want := PairPotential(2, 3, 4, 0, 0), -3.0; got != want {
		t.Errorf("Newtonian: %v, want %v", got, want)
	}
	if got, want := PairPotential(1, 1, 1, 0, 0.1), -1/0.9; math.Abs(got-want) > 1e-15 {
		t.Errorf("PW: %v, want %v", got, want)
	}
	if PairRS(Star, 1, Star, 1, 10) != 0 || PairRS(BlackHole, 1, BlackHole, 1, 10) != 0 || PairRS(Star, 1, BlackHole, 5, 10) != 0.1 {
		t.Error("PairRS must apply only to star–hole pairs, with the hole's radius")
	}
}

func TestDarkMatterPairsAreNewtonianExceptWithHoles(t *testing.T) {
	if PairRS(DarkMatter, 1, Star, 1, 10) != 0 || PairRS(Star, 1, DarkMatter, 1, 10) != 0 || PairRS(DarkMatter, 1, DarkMatter, 1, 10) != 0 {
		t.Error("star–dark matter and dark matter pairs must be Newtonian")
	}
	if PairRS(DarkMatter, 1, BlackHole, 5, 10) != 0.1 {
		t.Error("dark matter must feel the hole's Paczyński-Wiita potential")
	}
}
