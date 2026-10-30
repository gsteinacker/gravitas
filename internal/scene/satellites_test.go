package scene

import (
	"math"
	"slices"
	"testing"

	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
)

// satelliteScene returns the Milky Way satellites scene and its component named name.
func satelliteScene(t *testing.T, name string) (*Spec, Component) {
	t.Helper()
	for _, p := range Presets {
		if p.Spec.Name != "Milky Way satellites" {
			continue
		}
		for _, c := range p.Spec.Components {
			if c.Name == name {
				return p.Spec, c
			}
		}
	}
	t.Fatalf("no component %q in the satellites scene", name)
	return nil, Component{}
}

// selfBound returns the half-mass radius of the self-bound stars among bodies [lo, hi) and their
// share of all stars there, from the energy in the potential of the bound bodies (iterated).
func selfBound(s *sim.State, lo, hi int, eps float64) (rh, frac float64) {
	bound := make([]bool, hi-lo)
	for k := range bound {
		bound[k] = true
	}
	var c, v [3]float64
	for range 4 {
		var m float64
		c, v = [3]float64{}, [3]float64{}
		for i := lo; i < hi; i++ {
			if bound[i-lo] {
				m += s.Mass[i]
				c[0], c[1], c[2] = c[0]+s.Mass[i]*s.PosX[i], c[1]+s.Mass[i]*s.PosY[i], c[2]+s.Mass[i]*s.PosZ[i]
				v[0], v[1], v[2] = v[0]+s.Mass[i]*s.VelX[i], v[1]+s.Mass[i]*s.VelY[i], v[2]+s.Mass[i]*s.VelZ[i]
			}
		}
		for k := range 3 {
			c[k], v[k] = c[k]/m, v[k]/m
		}
		next := make([]bool, hi-lo)
		for i := lo; i < hi; i++ {
			phi := 0.0
			for j := lo; j < hi; j++ {
				if j != i && bound[j-lo] {
					dx, dy, dz := s.PosX[i]-s.PosX[j], s.PosY[i]-s.PosY[j], s.PosZ[i]-s.PosZ[j]
					phi -= s.Mass[j] / math.Sqrt(dx*dx+dy*dy+dz*dz+eps*eps)
				}
			}
			ux, uy, uz := s.VelX[i]-v[0], s.VelY[i]-v[1], s.VelZ[i]-v[2]
			next[i-lo] = 0.5*(ux*ux+uy*uy+uz*uz)+phi < 0
		}
		bound = next
	}
	var radii []float64
	stars := 0
	for i := lo; i < hi; i++ {
		if s.Kind[i] != sim.Star {
			continue
		}
		stars++
		if bound[i-lo] {
			dx, dy, dz := s.PosX[i]-c[0], s.PosY[i]-c[1], s.PosZ[i]-c[2]
			radii = append(radii, math.Sqrt(dx*dx+dy*dy+dz*dz))
		}
	}
	slices.Sort(radii)
	return radii[len(radii)/2], float64(len(radii)) / float64(stars)
}

// The satellites are resolved more finely than the host (ADR-0019) so that two-body relaxation
// does not dissolve them: an isolated Sagittarius progenitor keeps its size for many crossing
// times (at the host's particle masses its stellar radius grows by ~40 % in the same time).
func TestFinelyResolvedDwarfKeepsItsSize(t *testing.T) {
	if raceEnabled {
		t.Skip("long physics run, too slow under the race detector")
	}
	sp, c := satelliteScene(t, "Sagittarius dwarf")
	c.Position, c.Velocity = [3]float64{}, [3]float64{}
	s := sim.NewState(3000)
	addGalaxy(s, c, 3000, sp.Params, 0)
	rh0, _ := selfBound(s, 0, s.Len(), sp.Params.Softening)
	m := sim.New(s, &bh.Solver{}, sp.Params)
	for m.Time < 40 { // ~3 crossing times at the half-mass radius, 530 Myr
		m.Step()
	}
	rh, frac := selfBound(m.State, 0, m.State.Len(), sp.Params.Softening)
	t.Logf("half-mass radius %.3f → %.3f, bound stars %.3f", rh0, rh, frac)
	if math.Abs(rh/rh0-1) > 0.15 {
		t.Errorf("stellar half-mass radius changed from %.3f to %.3f, want within 15 %%", rh0, rh)
	}
	if frac < 0.98 {
		t.Errorf("only %.3f of the stars stay bound", frac)
	}
}

// Sagittarius loses stars to a tidal stream at its first pericenter (~16 kpc, after ~600 Myr), and
// the lost stars spread along its orbit. The host is the scene's Milky Way without gas.
func TestSagittariusFeedsATidalStream(t *testing.T) {
	if raceEnabled {
		t.Skip("long physics run, too slow under the race detector")
	}
	sp, mw := satelliteScene(t, "Milky Way")
	_, sgr := satelliteScene(t, "Sagittarius dwarf")
	mw.GasFraction, mw.InfallMsunPerYear, mw.Weight, sgr.Weight = 0, 0, 0.6, 0.4
	spec := *sp
	spec.Components = []Component{mw, sgr}
	spec.Params.Dt = 0.008
	const n = 6000
	s, p := spec.Build(n)
	cut := s.ID[s.Len()-spec.counts(n)[1]-1] // last Milky Way body
	m := sim.New(s, &bh.Solver{}, p)
	for m.Time < 64 {
		m.Step()
	}
	s = m.State
	lo := slices.IndexFunc(s.ID, func(id uint64) bool { return id > cut })
	_, frac := selfBound(s, lo, s.Len(), p.Softening)
	// Spread of the stars along the orbit: distance from the median star position.
	var xs, ys, zs []float64
	for i := lo; i < s.Len(); i++ {
		if s.Kind[i] == sim.Star {
			xs, ys, zs = append(xs, s.PosX[i]), append(ys, s.PosY[i]), append(zs, s.PosZ[i])
		}
	}
	med := func(a []float64) float64 { b := slices.Clone(a); slices.Sort(b); return b[len(b)/2] }
	cx, cy, cz := med(xs), med(ys), med(zs)
	far := 0
	for k := range xs {
		if math.Sqrt((xs[k]-cx)*(xs[k]-cx)+(ys[k]-cy)*(ys[k]-cy)+(zs[k]-cz)*(zs[k]-cz)) > 4 { // 10 kpc
			far++
		}
	}
	t.Logf("bound stars %.3f, stars beyond 10 kpc from the core %.3f", frac, float64(far)/float64(len(xs)))
	if frac > 0.9 {
		t.Errorf("Sagittarius kept %.3f of its stars after its pericenter, want a stream", frac)
	}
	if float64(far) < 0.05*float64(len(xs)) {
		t.Errorf("only %d of %d stars beyond 10 kpc from the core: no stream along the orbit", far, len(xs))
	}
}
