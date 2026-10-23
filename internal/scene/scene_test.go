package scene

import (
	"math"
	"testing"

	"galaxy/internal/sim"
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
