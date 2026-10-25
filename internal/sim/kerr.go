package sim

import (
	"math"

	"galaxy/internal/vec"
)

// Rotating black holes (T-071, docs/analysis/03-kerr.md, model A): every hole carries its angular
// momentum J (SpinX/Y/Z in State). It grows with what the hole swallows and with mergers and is
// capped at the Thorne limit. The dynamics stays Schwarzschild (Paczyński-Wiita); the spin sets the
// drawn horizon, ISCO and photon ring, and the inner edge of accretion disks.

// MaxSpin is the Thorne limit a* = 0.998: radiation swallowed by the hole stops the spin-up.
const MaxSpin = 0.998

// SpinParameter returns a* = |J| c / (G M²) (G = 1), 0 without a speed of light.
func SpinParameter(j vec.Vec3, m, c float64) float64 {
	if c <= 0 || m <= 0 {
		return 0
	}
	return math.Min(1, j.Len()*c/(m*m))
}

// maxAngularMomentum is |J| at the Thorne limit.
func maxAngularMomentum(m, c float64) float64 { return MaxSpin * m * m / c }

// KerrHorizon returns the outer horizon r₊ = r_g (1 + √(1 − a²)) with r_g = GM/c².
func KerrHorizon(m, a, c float64) float64 {
	if c <= 0 {
		return 0
	}
	return m / (c * c) * (1 + math.Sqrt(math.Max(0, 1-a*a)))
}

// KerrISCO returns the radius of the innermost stable circular orbit (Bardeen, Press & Teukolsky
// 1972) for orbits with (prograde) or against the spin: 6 r_g at a = 0, 1 and 9 r_g at a → 1.
func KerrISCO(m, a, c float64, prograde bool) float64 {
	if c <= 0 {
		return 0
	}
	z1 := 1 + math.Cbrt(1-a*a)*(math.Cbrt(1+a)+math.Cbrt(1-a))
	z2 := math.Sqrt(3*a*a + z1*z1)
	s := math.Sqrt((3 - z1) * (3 + z1 + 2*z2))
	if prograde {
		s = -s
	}
	return m / (c * c) * (3 + z2 + s)
}

// KerrPhotonOrbit returns the radius of the circular photon orbit in the equatorial plane:
// 2 r_g (1 + cos(⅔ arccos(∓a))), 3 r_g at a = 0, 1 and 4 r_g at a → 1.
func KerrPhotonOrbit(m, a, c float64, prograde bool) float64 {
	if c <= 0 {
		return 0
	}
	if prograde {
		a = -a
	}
	return 2 * m / (c * c) * (1 + math.Cos(2.0/3*math.Acos(a)))
}

// Spin returns the angular momentum of body i.
func (s *State) Spin(i int) vec.Vec3 { return vec.Vec3{X: s.SpinX[i], Y: s.SpinY[i], Z: s.SpinZ[i]} }

// capSpin limits the spin of hole h to the Thorne limit; the excess would be radiated away.
func (s *State) capSpin(h int, c float64) {
	if c <= 0 || s.Kind[h] != BlackHole {
		return
	}
	j := s.Spin(h)
	if l, lim := j.Len(), maxAngularMomentum(s.Mass[h], c); l > lim {
		j = j.Scale(lim / l)
		s.SpinX[h], s.SpinY[h], s.SpinZ[h] = j.X, j.Y, j.Z
	}
}
