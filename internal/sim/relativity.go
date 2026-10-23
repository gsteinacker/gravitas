package sim

import (
	"math"

	"galaxy/internal/vec"
)

// Relativity model (docs/physics.md, ADR-0003): with Params.C > 0 the stored velocity is the
// proper velocity u = γv. The integrator kicks u by the Newtonian acceleration and drifts with
// v = u/γ, γ = √(1 + u²/c²). That is a symplectic leapfrog for H = Σ m c² (γ − 1) + U, so
// |v| < c by construction and energy errors stay bounded. Momentum is p = m·u.

// gamma returns the Lorentz factor for proper velocity u; c <= 0 means Newtonian (γ = 1).
func gamma(ux, uy, uz, c float64) float64 {
	if c <= 0 {
		return 1
	}
	return math.Sqrt(1 + (ux*ux+uy*uy+uz*uz)/(c*c))
}

// kinetic is m c² (γ − 1), written as m u² / (γ + 1) so it is exact for u ≪ c and equals
// the Newtonian m u²/2 for c <= 0.
func kinetic(m, ux, uy, uz, c float64) float64 {
	return m * (ux*ux + uy*uy + uz*uz) / (gamma(ux, uy, uz, c) + 1)
}

// CoordinateVelocity converts a stored (proper) velocity u into the velocity v = u/γ.
func CoordinateVelocity(u vec.Vec3, c float64) vec.Vec3 {
	return u.Scale(1 / gamma(u.X, u.Y, u.Z, c))
}

// maxSpeedFraction caps velocities handed to ProperVelocity, which diverges at v → c.
const maxSpeedFraction = 0.99

// ProperVelocity converts a coordinate velocity v into the stored proper velocity u = γv.
// Speeds at or above c are capped to 0.99c.
func ProperVelocity(v vec.Vec3, c float64) vec.Vec3 {
	if c <= 0 {
		return v
	}
	if l := v.Len(); l > maxSpeedFraction*c {
		v = v.Scale(maxSpeedFraction * c / l)
	}
	return v.Scale(1 / math.Sqrt(1-v.Len2()/(c*c)))
}

// SchwarzschildRadius returns r_s = 2GM/c² (G = 1), or 0 for c <= 0.
func SchwarzschildRadius(m, c float64) float64 {
	if c <= 0 {
		return 0
	}
	return 2 * m / (c * c)
}
