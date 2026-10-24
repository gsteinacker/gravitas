package scene

import (
	"math"
	"slices"
	"testing"

	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
	"galaxy/internal/sim/direct"
	"galaxy/internal/vec"
)

func TestPlummerIsNearVirialEquilibrium(t *testing.T) {
	s := sim.NewState(2000)
	AddPlummer(s, 2000, 1, 1, vec.Vec3{}, vec.Vec3{}, 1)
	RemoveBulkMotion(s)
	k, u := sim.Energy(s, 0)
	if ratio := 2 * k / -u; math.Abs(ratio-1) > 0.15 {
		t.Errorf("virial ratio 2K/|U| = %.3f, want ≈ 1", ratio)
	}
}

func TestPresetsBuildAndStayBound(t *testing.T) {
	for _, p := range Presets {
		t.Run(p.Name, func(t *testing.T) {
			s, params := p.Build(500)
			if s.Len() < 500 {
				t.Fatalf("only %d bodies", s.Len())
			}
			c, m := sim.CenterOfMass(s)
			if c.Len() > 1e-9*m && p.Spec.BulkRest {
				t.Errorf("center of mass not at origin: %v", c)
			}
			k, u := sim.Energy(s, params.Softening)
			if k+u >= 0 {
				t.Errorf("system unbound: E = %.3f", k+u)
			}
			m2 := sim.New(s, &direct.Solver{}, params)
			for range 20 {
				m2.Step()
			}
			for i := range s.Len() {
				if math.IsNaN(s.PosX[i]) {
					t.Fatalf("NaN position for body %d", i)
				}
			}
		})
	}
}

func TestHernquistHaloIsNearEquilibriumAndFlat(t *testing.T) {
	// Untruncated limit (large rmax): virial ratio ≈ 1.
	s := sim.NewState(20000)
	AddHernquistHalo(s, 20000, 1, 0.3, 60, vec.Vec3{}, vec.Vec3{}, 7)
	RemoveBulkMotion(s)
	k := sim.KineticEnergy(s, 0)
	u := sim.PotentialEnergyParallel(s, sim.Params{Softening: 0.01})
	if ratio := 2 * k / -u; math.Abs(ratio-1) > 0.1 {
		t.Errorf("virial ratio 2K/|U| = %.3f, want ≈ 1", ratio)
	}
	// Milky Way-like halo (1e12 Msun inside 300 kpc, a = 30 kpc) plus a 6e10 disk: flat rotation curve.
	for _, r := range []float64{0.08, 0.2, 0.5} {
		m := hernquistEnclosed(0.94, 0.3, 3, r) + 0.06*(1-math.Exp(-r/0.03)*(1+r/0.03))
		if v := math.Sqrt(m/r) * 207; v < 180 || v > 240 {
			t.Errorf("v_c(%.0f kpc) = %.0f km/s, want ~200-230", r*100, v)
		}
	}
}

// An isolated Hernquist bulge with Jeans velocities is near virial equilibrium, and its mass
// profile stays put over a few dynamical times.
func TestBulgeIsNearEquilibrium(t *testing.T) {
	s := sim.NewState(8000)
	c := DiskConfig{BulgeMass: 1, BulgeScale: 0.2, Softening: 0.01, Seed: 5}
	AddBulge(s, 8000, c)
	RemoveBulkMotion(s)
	k := sim.KineticEnergy(s, 0)
	u := sim.PotentialEnergyParallel(s, sim.Params{Softening: 0.01})
	if ratio := 2 * k / -u; math.Abs(ratio-1) > 0.1 {
		t.Errorf("virial ratio 2K/|U| = %.3f, want ≈ 1", ratio)
	}
	half := func() float64 {
		r := make([]float64, s.Len())
		for i := range r {
			r[i] = math.Sqrt(s.PosX[i]*s.PosX[i] + s.PosY[i]*s.PosY[i] + s.PosZ[i]*s.PosZ[i])
		}
		slices.Sort(r)
		return r[len(r)/2]
	}
	r0 := half()
	// Dynamical time at the half-mass radius: √(r³/GM) ≈ 0.2; run ~5 of them.
	m := sim.New(s, &bh.Solver{}, sim.Params{Dt: 0.002, Softening: 0.01})
	for range 500 {
		m.Step()
	}
	if r1 := half(); math.Abs(r1/r0-1) > 0.1 {
		t.Errorf("half-mass radius %.3f -> %.3f", r0, r1)
	}
}

// The bulge deepens the potential: the inner rotation curve of a disk rises with it.
func TestBulgeRaisesInnerRotation(t *testing.T) {
	c := DiskConfig{StarMass: 1, ScaleRadius: 1, MinRadius: 0.05, MaxRadius: 5, Softening: 0.01}
	without := c.enclosed(0.5)
	c.BulgeMass, c.BulgeScale = 0.3, 0.1
	if with := c.enclosed(0.5); with-without < 0.6*0.3 {
		t.Errorf("enclosed mass at 0.5: %.3f without, %.3f with bulge", without, with)
	}
}

// The dust opacity follows from the scene units: a column of 1 Msun/pc² is A_V ≈ 0.05 mag
// (Milky Way dust-to-gas ratio).
func TestDustOpacityFromUnits(t *testing.T) {
	// 1 unit = 1 pc, 0.656 km/s: mass unit = v² L / G ≈ 100 Msun.
	sp := &Spec{KmPerS: 0.656, YearsPer: 1.49e6}
	perMsunPc2 := sp.DustOpacity() / 100
	if av := 1.086 * perMsunPc2; math.Abs(av-0.047) > 0.005 {
		t.Errorf("A_V of 1 Msun/pc² = %.4f, want ≈ 0.047", av)
	}
	if (&Spec{}).DustOpacity() != 0 {
		t.Error("without units the opacity must be 0 (automatic)")
	}
}
