// Package scene builds initial conditions (units: G = 1).
package scene

import (
	"math"
	"math/rand/v2"
	"slices"
	"sort"

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
	Dispersion  float64 // random velocity as a fraction of the circular velocity (if Q == 0)
	// Q > 0 sets the radial dispersion from the Toomre criterion instead (see addToomreDisk).
	Q float64
	// ColdFraction of the stars form a young, cold population (dispersion ×coldScale, thinner,
	// more massive and therefore drawn bluer and brighter); it traces spiral arms. Needs Q > 0.
	ColdFraction float64
	// GasFraction of the disk particles (and mass) are gas with specific internal energy GasU.
	// Needs Q > 0.
	GasFraction float64
	GasU        float64
	HaloMass    float64 // mass of a dark matter halo around the disk (added separately)
	HaloScale   float64 // its scale radius
	// HaloHernquist selects a Hernquist profile; HaloMass is then the mass inside HaloMax.
	HaloHernquist bool
	HaloMax       float64
	Center        vec.Vec3
	Velocity      vec.Vec3
	Inclination   float64 // rotation around x, radians
	Orientation   float64 // rotation around z after inclination, radians
	Seed          uint64
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

	if c.Q > 0 {
		addToomreDisk(s, c, rng, masses, radii, place)
		return
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
			// Halo mass inside the star's (spherical) radius, analytic for both profiles.
			rr := r*r + z*z
			if c.HaloHernquist {
				m += hernquistEnclosed(c.HaloMass, c.HaloScale, c.HaloMax, math.Sqrt(rr))
			} else {
				m += c.HaloMass * rr * math.Sqrt(rr) / math.Pow(rr+c.HaloScale*c.HaloScale, 1.5)
			}
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

// coldScale is the dispersion (and thickness) factor of the cold population, coldMass the mass
// of its stars relative to the mean.
const (
	coldScale = 0.3
	coldMass  = 3.0
)

// addToomreDisk places the stars with dispersions from the Toomre criterion, following the usual
// construction of disk initial conditions (Hernquist 1993): σ_R = Q · 3.36 G Σ / κ, σ_φ = σ_R κ / 2Ω
// (epicycle approximation), σ_z² = π G Σ z0 (isothermal sheet, ρ ∝ sech²(z/z0)), and the mean
// rotation lowered by the asymmetric drift v_φ² = v_c² + σ_R² (1 − κ²/4Ω² − 2R/R_d).
// The rotation curve uses the spherically approximated enclosed mass of hole, disk and halo.
func addToomreDisk(s *sim.State, c DiskConfig, rng *rand.Rand, masses, radii []float64, place func(p, v vec.Vec3) (vec.Vec3, vec.Vec3)) {
	n := len(radii)
	cold, gas := make([]bool, n), make([]bool, n)
	var total float64
	for i := range cold {
		switch r := rng.Float64(); {
		case r < c.GasFraction:
			gas[i] = true
			masses[i] = c.StarMass / float64(n) // equal-mass gas particles
		case r < c.GasFraction+c.ColdFraction:
			cold[i] = true
			masses[i] = coldMass * c.StarMass / float64(n)
		}
		total += masses[i]
	}
	for i := range masses {
		masses[i] *= c.StarMass / total
	}
	cum := make([]float64, n) // disk mass inside radii[i], with the radii sorted
	var acc float64
	for i := range radii {
		acc += masses[i]
		cum[i] = acc
	}
	eps2 := c.Softening * c.Softening
	enclosed := func(r float64) float64 {
		m := c.BHMass
		if k := sort.SearchFloat64s(radii, r); k > 0 {
			m += cum[k-1]
		}
		if c.HaloMass > 0 {
			if c.HaloHernquist {
				m += hernquistEnclosed(c.HaloMass, c.HaloScale, c.HaloMax, r)
			} else {
				m += c.HaloMass * r * r * r / math.Pow(r*r+c.HaloScale*c.HaloScale, 1.5)
			}
		}
		return m
	}
	omega2 := func(r float64) float64 {
		r2 := r*r + eps2
		return enclosed(r) / (r2 * math.Sqrt(r2))
	}
	rd := c.ScaleRadius
	gammaCDF := func(r float64) float64 { return 1 - math.Exp(-r/rd)*(1+r/rd) }
	norm := gammaCDF(c.MaxRadius) - gammaCDF(c.MinRadius)
	for i, r := range radii {
		w2 := omega2(r)
		h := 0.01 * r
		kappa2 := math.Max(r*(omega2(r+h)-omega2(r-h))/(2*h)+4*w2, 1e-12)
		kappa, omega := math.Sqrt(kappa2), math.Sqrt(w2)
		vc := omega * r
		sigma := c.StarMass * math.Exp(-r/rd) / (2 * math.Pi * rd * rd * norm)
		sigR := math.Min(c.Q*3.36*sigma/kappa, 0.6*vc)
		z0 := c.Thickness
		if cold[i] {
			sigR, z0 = sigR*coldScale, z0*coldScale
		}
		sigPhi := sigR * kappa / (2 * omega)
		sigZ := math.Sqrt(math.Pi * sigma * z0)
		if cold[i] {
			sigZ *= coldScale
		}
		vphi := math.Sqrt(math.Max(0, vc*vc+sigR*sigR*(1-kappa2/(4*w2)-2*r/rd)))
		if gas[i] {
			// Gas is supported by pressure, not by random motions: small turbulence, circular
			// rotation, a thin layer.
			cs := math.Sqrt(10.0 / 9 * c.GasU)
			sigR, sigPhi, sigZ, z0, vphi = 0.3*cs, 0.3*cs, 0.3*cs, 0.5*c.Thickness, vc
		}

		phi := rng.Float64() * 2 * math.Pi
		z := z0 * math.Atanh(2*rng.Float64()-1)
		sinp, cosp := math.Sincos(phi)
		vr, vt := rng.NormFloat64()*sigR, vphi+rng.NormFloat64()*sigPhi
		pos := vec.Vec3{X: r * cosp, Y: r * sinp, Z: z}
		vel := vec.Vec3{X: vr*cosp - vt*sinp, Y: vr*sinp + vt*cosp, Z: rng.NormFloat64() * sigZ}
		p, v := place(pos, vel)
		b := sim.Body{Pos: p, Vel: v, Mass: masses[i]}
		if gas[i] {
			b.Kind, b.U = sim.Gas, c.GasU
		}
		s.Add(b)
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

// Hernquist (1990) halo: ρ ∝ 1/(r (r+a)³), M(<r) = M r²/(r+a)². Unlike Plummer it has a cusp and
// an extended envelope, which gives flat rotation curves, e.g. ~220 km/s from 8 to 50 kpc for the
// Milky Way. The profile is truncated at rmax; mass is the mass inside rmax.

// hernquistTotal is the untruncated profile mass whose truncation at rmax holds mass.
func hernquistTotal(mass, a, rmax float64) float64 {
	u := rmax / (rmax + a)
	return mass / (u * u)
}

func hernquistEnclosed(mass, a, rmax, r float64) float64 {
	r = math.Min(r, rmax)
	return hernquistTotal(mass, a, rmax) * r * r / ((r + a) * (r + a))
}

// hernquistSigma2 is the isotropic one-dimensional velocity dispersion of an untruncated
// Hernquist sphere of total mass m (Hernquist 1990, eq. 10).
func hernquistSigma2(m, a, r float64) float64 {
	x := r / a
	return m / (12 * a) * (12*x*math.Pow(1+x, 3)*math.Log((1+x)/x) - x/(1+x)*(25+52*x+42*x*x+12*x*x*x))
}

// AddHernquistHalo adds an invisible dark matter halo with a Hernquist profile truncated at rmax.
// Velocities are drawn from a Gaussian with the local isotropic Jeans dispersion and capped below the
// local escape speed; that is close to, though not exactly, equilibrium.
func AddHernquistHalo(s *sim.State, n int, mass, a, rmax float64, center, velocity vec.Vec3, seed uint64) {
	rng := rand.New(rand.NewPCG(seed, 0x4e57))
	mTot := hernquistTotal(mass, a, rmax)
	smax := math.Sqrt(mass / mTot) // s = r/(r+a) at rmax; the enclosed fraction is s², uniform in mass
	for range n {
		su := smax * math.Sqrt(rng.Float64())
		r := a * su / (1 - su)
		sigma := math.Sqrt(math.Max(0, hernquistSigma2(mTot, a, r)))
		vesc := math.Sqrt(2 * mTot / (r + a))
		var v vec.Vec3
		for {
			v = vec.Vec3{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()}.Scale(sigma)
			if v.Len() < 0.95*vesc {
				break
			}
		}
		s.Add(sim.Body{
			Pos:  randomDirection(rng).Scale(r).Add(center),
			Vel:  v.Add(velocity),
			Mass: mass / float64(n),
			Kind: sim.DarkMatter,
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
