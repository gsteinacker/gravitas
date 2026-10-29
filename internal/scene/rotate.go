package scene

import (
	"math"
	"math/rand/v2"
	"slices"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// Rotation of placed clusters and clouds (T-082). Molecular clouds rotate with β = E_rot/|W| of
// about 10⁻⁴–0.07, typically 0.02 (Goodman et al. 1993); globular clusters with v_rot/σ ≈ 0.1–0.5.

// SpinUpCloud adds ordered rotation about axis to the bodies from index first on: each moves an
// extra f·v_c(r)·sin θ in the azimuthal direction, with v_c(r) = √(M(<r)/r) from all bodies of s
// (so a halo added before counts). For a spherical system Σ m v_c² = |W|, so the rotation carries
// β = E_rot/|W| = f²/3 of the binding energy (CloudBeta).
func SpinUpCloud(s *sim.State, first int, center, axis vec.Vec3, f float64) {
	if f == 0 || first >= s.Len() {
		return
	}
	axis = unitOr(axis, vec.Vec3{Z: 1})
	radii, cum := enclosedMass(s, center)
	for i := first; i < s.Len(); i++ {
		d := vec.Vec3{X: s.PosX[i] - center.X, Y: s.PosY[i] - center.Y, Z: s.PosZ[i] - center.Z}
		r := d.Len()
		if r == 0 {
			continue
		}
		k, _ := slices.BinarySearch(radii, math.Nextafter(r, math.Inf(1)))
		m := 0.0
		if k > 0 {
			m = cum[k-1]
		}
		// axis × d has length r sin θ and points along φ̂.
		v := axis.Cross(d).Scale(f * math.Sqrt(m/r) / r)
		s.VelX[i], s.VelY[i], s.VelZ[i] = s.VelX[i]+v.X, s.VelY[i]+v.Y, s.VelZ[i]+v.Z
	}
}

// CloudBeta is the ratio of rotational to gravitational energy of a cloud spun up by f.
func CloudBeta(f float64) float64 { return f * f / 3 }

// SpinUpCluster makes a share f of the stars from index first on orbit about axis in the same
// sense (Lynden-Bell 1960): a star whose azimuthal velocity (relative to vel) has the wrong sign
// gets it reversed with probability f. Speeds, radii and energy stay the same, so an equilibrium
// cluster stays in equilibrium; f = 1 is the maximum rotation for its velocity dispersion.
func SpinUpCluster(s *sim.State, first int, center, vel, axis vec.Vec3, f float64, seed uint64) {
	if f <= 0 {
		return
	}
	axis = unitOr(axis, vec.Vec3{Z: 1})
	rng := rand.New(rand.NewPCG(seed, 0x5917))
	for i := first; i < s.Len(); i++ {
		d := vec.Vec3{X: s.PosX[i] - center.X, Y: s.PosY[i] - center.Y, Z: s.PosZ[i] - center.Z}
		phi := axis.Cross(d)
		if l := phi.Len(); l > 0 {
			phi = phi.Scale(1 / l)
		} else {
			continue
		}
		v := vec.Vec3{X: s.VelX[i], Y: s.VelY[i], Z: s.VelZ[i]}.Sub(vel)
		if vp := v.Dot(phi); vp < 0 && rng.Float64() < f {
			v = v.Sub(phi.Scale(2 * vp)).Add(vel)
			s.VelX[i], s.VelY[i], s.VelZ[i] = v.X, v.Y, v.Z
		}
	}
}

// enclosedMass returns the sorted distances of all bodies from center and the cumulative mass up to
// each of them.
func enclosedMass(s *sim.State, center vec.Vec3) (radii, cum []float64) {
	type body struct{ r, m float64 }
	bs := make([]body, s.Len())
	for i := range bs {
		bs[i] = body{math.Sqrt(sq(s.PosX[i]-center.X) + sq(s.PosY[i]-center.Y) + sq(s.PosZ[i]-center.Z)), s.Mass[i]}
	}
	slices.SortFunc(bs, func(a, b body) int { return cmpFloat(a.r, b.r) })
	radii, cum = make([]float64, len(bs)), make([]float64, len(bs))
	var m float64
	for k, b := range bs {
		m += b.m
		radii[k], cum[k] = b.r, m
	}
	return radii, cum
}

func sq(x float64) float64 { return x * x }

func cmpFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func unitOr(v, fallback vec.Vec3) vec.Vec3 {
	if l := v.Len(); l > 0 {
		return v.Scale(1 / l)
	}
	return fallback
}
