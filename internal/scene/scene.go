// Package scene builds initial conditions (units: G = 1).
package scene

import (
	"math"
	"math/rand/v2"
	"slices"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// DiskConfig describes a rotating exponential disk galaxy around a central black hole.
type DiskConfig struct {
	N           int
	StarMass    float64 // total stellar mass
	BHMass      float64 // central black hole; 0 for none
	ScaleRadius float64
	MinRadius   float64
	MaxRadius   float64
	Thickness   float64 // vertical scale height
	Softening   float64 // used for circular velocities
	Dispersion  float64 // random velocity as a fraction of the circular velocity
	HaloMass    float64 // mass of a Plummer dark matter halo around the disk (added separately)
	HaloScale   float64 // its Plummer radius
	Center      vec.Vec3
	Velocity    vec.Vec3
	Inclination float64 // rotation around x, radians
	Orientation float64 // rotation around z after inclination, radians
	Seed        uint64
}

// AddDisk adds a disk galaxy to s.
func AddDisk(s *sim.State, c DiskConfig) {
	rng := rand.New(rand.NewPCG(c.Seed, 0x9a1a))
	masses := stellarMasses(rng, c.N, c.StarMass)

	radii := make([]float64, c.N)
	for i := range radii {
		for {
			// Σ(r) ∝ exp(-r/Rd) ⇒ r follows Gamma(2, Rd).
			r := -c.ScaleRadius * math.Log(rng.Float64()*rng.Float64())
			if r >= c.MinRadius && r <= c.MaxRadius {
				radii[i] = r
				break
			}
		}
	}
	slices.Sort(radii)

	place := func(p, v vec.Vec3) (vec.Vec3, vec.Vec3) {
		p = p.RotateX(c.Inclination).RotateZ(c.Orientation)
		v = v.RotateX(c.Inclination).RotateZ(c.Orientation)
		return p.Add(c.Center), v.Add(c.Velocity)
	}
	if c.BHMass > 0 {
		p, v := place(vec.Vec3{}, vec.Vec3{})
		s.Add(sim.Body{Pos: p, Vel: v, Mass: c.BHMass, Kind: sim.BlackHole})
	}

	eps2 := c.Softening * c.Softening
	enclosed := c.BHMass
	for i, r := range radii {
		enclosed += masses[i]
		phi := rng.Float64() * 2 * math.Pi
		z := rng.NormFloat64() * c.Thickness
		r2 := r*r + z*z + eps2
		m := enclosed
		if c.HaloMass > 0 {
			// Halo mass inside the star's (spherical) radius, analytic for a Plummer sphere.
			rr := r*r + z*z
			m += c.HaloMass * rr * math.Sqrt(rr) / math.Pow(rr+c.HaloScale*c.HaloScale, 1.5)
		}
		vc := math.Sqrt(m * r * r / (r2 * math.Sqrt(r2)))
		sinp, cosp := math.Sincos(phi)
		pos := vec.Vec3{X: r * cosp, Y: r * sinp, Z: z}
		disp := c.Dispersion * vc
		vel := vec.Vec3{
			X: -vc*sinp + rng.NormFloat64()*disp,
			Y: vc*cosp + rng.NormFloat64()*disp,
			Z: rng.NormFloat64() * disp,
		}
		p, v := place(pos, vel)
		s.Add(sim.Body{Pos: p, Vel: v, Mass: masses[i]})
	}
}

// AddPlummer adds a Plummer sphere of stars in virial equilibrium (Aarseth, Hénon & Wielen 1974).
func AddPlummer(s *sim.State, n int, totalMass, a float64, center, velocity vec.Vec3, seed uint64) {
	addPlummer(s, n, totalMass, a, center, velocity, seed, sim.Star, 0.999)
}

// AddHalo adds an invisible dark matter halo as a Plummer sphere of equal-mass particles, cut off
// at the mass fraction maxFrac (so halos do not reach absurd radii).
func AddHalo(s *sim.State, n int, totalMass, a float64, center, velocity vec.Vec3, seed uint64, maxFrac float64) {
	addPlummer(s, n, totalMass, a, center, velocity, seed, sim.DarkMatter, maxFrac)
}

func addPlummer(s *sim.State, n int, totalMass, a float64, center, velocity vec.Vec3, seed uint64, kind sim.Kind, maxFrac float64) {
	rng := rand.New(rand.NewPCG(seed, 0x91))
	var masses []float64
	if kind == sim.Star {
		masses = stellarMasses(rng, n, totalMass)
	} else {
		masses = make([]float64, n)
		for i := range masses {
			masses[i] = totalMass / float64(n)
		}
	}
	for i := range n {
		r := a / math.Sqrt(math.Pow(rng.Float64()*maxFrac+1e-6, -2.0/3.0)-1)
		var q float64
		for {
			q = rng.Float64()
			if 0.1*rng.Float64() < q*q*math.Pow(1-q*q, 3.5) {
				break
			}
		}
		ve := math.Sqrt(2*totalMass) * math.Pow(r*r+a*a, -0.25)
		s.Add(sim.Body{
			Pos:  randomDirection(rng).Scale(r).Add(center),
			Vel:  randomDirection(rng).Scale(q * ve).Add(velocity),
			Mass: masses[i],
			Kind: kind,
		})
	}
}

// stellarMasses draws n masses from a Salpeter-like power law and normalizes them to total.
func stellarMasses(rng *rand.Rand, n int, total float64) []float64 {
	const lo, hi, alpha = 0.1, 10.0, 2.35
	k := 1 - alpha
	a, b := math.Pow(lo, k), math.Pow(hi, k)
	masses := make([]float64, n)
	var sum float64
	for i := range masses {
		masses[i] = math.Pow(a+rng.Float64()*(b-a), 1/k)
		sum += masses[i]
	}
	for i := range masses {
		masses[i] *= total / sum
	}
	// Shuffle so heavy stars are not correlated with radius after sorting radii.
	rng.Shuffle(n, func(i, j int) { masses[i], masses[j] = masses[j], masses[i] })
	return masses
}

func randomDirection(rng *rand.Rand) vec.Vec3 {
	z := 2*rng.Float64() - 1
	phi := 2 * math.Pi * rng.Float64()
	r := math.Sqrt(1 - z*z)
	return vec.Vec3{X: r * math.Cos(phi), Y: r * math.Sin(phi), Z: z}
}

// RemoveBulkMotion shifts s so that its center of mass is at rest in the origin.
func RemoveBulkMotion(s *sim.State) {
	c, m := sim.CenterOfMass(s)
	if m == 0 {
		return
	}
	v := sim.Momentum(s).Scale(1 / m)
	for i := range s.Len() {
		s.PosX[i] -= c.X
		s.PosY[i] -= c.Y
		s.PosZ[i] -= c.Z
		s.VelX[i] -= v.X
		s.VelY[i] -= v.Y
		s.VelZ[i] -= v.Z
	}
}
