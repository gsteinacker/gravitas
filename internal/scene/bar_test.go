package scene

import (
	"testing"

	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
)

// barAfter evolves the galaxy of the spiral scene without gas, changed by tune, and returns its
// bar strength at time t.
func barAfter(t *testing.T, n int, until float64, tune func(*Component)) float64 {
	t.Helper()
	sp := *Presets[0].Spec
	c := sp.Components[0]
	c.GasFraction, c.InfallMsunPerYear, c.Weight = 0, 0, 1
	tune(&c)
	sp.Components = []Component{c}
	s, p := sp.Build(n)
	p.Dt *= 2 // the bar grows within a few rotations; accuracy of single orbits is not needed
	m := sim.New(s, &bh.Solver{}, p)
	for m.Time < until {
		m.Step()
	}
	return sim.MeasureBar(m.State).A2
}

// A disk that dominates the inner galaxy (extended halo, small bulge, Q = 1) forms a bar within a
// few rotations; the halo-dominated disk of the spiral scene does not (T-090).
func TestDiskDominatedGalaxyFormsABar(t *testing.T) {
	if raceEnabled {
		t.Skip("long physics run, too slow under the race detector")
	}
	barred := barAfter(t, 6000, 25, func(c *Component) { c.HaloScale, c.BulgeFrac, c.ToomreQ = 8, 0.05, 1.0 })
	plain := barAfter(t, 6000, 25, func(*Component) {})
	t.Logf("A2 after 330 Myr: disk-dominated %.3f, spiral scene %.3f", barred, plain)
	if barred < 0.15 || barred < 2*plain {
		t.Errorf("disk-dominated galaxy: A2 = %.3f, want a bar (> 0.15 and 2× the spiral scene's)", barred)
	}
	if plain > 0.1 {
		t.Errorf("spiral scene: A2 = %.3f, want no bar (< 0.1)", plain)
	}
}
