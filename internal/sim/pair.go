package sim

import "math"

// Pair interaction model (docs/physics.md): star–star and hole–hole pairs are softened Newtonian;
// star–hole pairs use the Paczyński-Wiita potential Φ = −M/(r − r_s) of the hole, with r replaced
// by the softened distance r_ε = √(r² + ε²). PW is a symmetric pair potential, so momentum is
// conserved. Inside 1.5 r_s (photon sphere) the force is capped at its value there to avoid the
// singularity at r_s; the ISCO at 3 r_s is unaffected. With rs = 0 everything is Newtonian.

// PairRS returns the Schwarzschild radius governing the pair (i, j): that of the hole if exactly
// one of them is a hole, 0 otherwise (dark matter counts like a star).
func PairRS(ki Kind, mi float64, kj Kind, mj, c float64) float64 {
	switch {
	case c <= 0 || (ki == BlackHole) == (kj == BlackHole):
		return 0
	case ki == BlackHole:
		return SchwarzschildRadius(mi, c)
	default:
		return SchwarzschildRadius(mj, c)
	}
}

// PairForce returns f such that the acceleration on a body from a source of mass m at separation
// d (|d|² = r2) is f·d.
func PairForce(m, r2, eps2, rs float64) float64 {
	re2 := r2 + eps2
	re := math.Sqrt(re2)
	if rs == 0 {
		return m / (re2 * re)
	}
	den := math.Max(re-rs, rs/2)
	return m / (den * den * re)
}

// PairPotential returns the potential energy of masses mi and mj at squared separation r2.
func PairPotential(mi, mj, r2, eps2, rs float64) float64 {
	re := math.Sqrt(r2 + eps2)
	if rs == 0 {
		return -mi * mj / re
	}
	if re >= 1.5*rs {
		return -mi * mj / (re - rs)
	}
	// Linear continuation matching the capped force 4m/rs² below the photon sphere.
	return mi * mj * (-2/rs + 4/(rs*rs)*(re-1.5*rs))
}
