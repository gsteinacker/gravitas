package cosmo

import "math"

// Cosmology are the parameters of a flat ΛCDM universe.
type Cosmology struct {
	H      float64 // h = H₀ / (100 km/s/Mpc)
	OmegaM float64 // matter
	OmegaB float64 // baryons (part of OmegaM)
	Sigma8 float64 // rms of the linear density in spheres of 8 Mpc/h today
	NS     float64 // spectral index
}

// Planck2018 are the Planck 2018 parameters (TT,TE,EE+lowE+lensing; Planck Collaboration 2020).
var Planck2018 = Cosmology{H: 0.677, OmegaM: 0.31, OmegaB: 0.049, Sigma8: 0.81, NS: 0.965}

// E returns H(a)/H₀ for flat ΛCDM (radiation neglected).
func (c Cosmology) E(a float64) float64 { return math.Sqrt(c.OmegaM/(a*a*a) + 1 - c.OmegaM) }

// Growth returns the linear growth factor D(a), normalized to D(1) = 1:
// D ∝ E(a) ∫₀ᵃ da' / (a' E(a'))³ (Heath 1977; exact for ΛCDM without radiation).
func (c Cosmology) Growth(a float64) float64 { return c.growth(a) / c.growth(1) }

func (c Cosmology) growth(a float64) float64 {
	const n = 2000
	var sum float64
	for i := range n { // midpoint rule in a; the integrand is smooth (∝ a^{3/2} near 0)
		x := a * (float64(i) + 0.5) / n
		ex := x * c.E(x)
		sum += 1 / (ex * ex * ex)
	}
	return 2.5 * c.OmegaM * c.E(a) * sum * a / n
}

// GrowthRate returns f = d ln D / d ln a ≈ Ω_m(a)^0.55 (Linder 2005, accurate to ~0.1 % in ΛCDM).
func (c Cosmology) GrowthRate(a float64) float64 {
	om := c.OmegaM / (a * a * a) / (c.E(a) * c.E(a))
	return math.Pow(om, 0.55)
}

// transfer is the no-wiggle transfer function of Eisenstein & Hu (1998, eq. 26–31); k in h/Mpc.
func (c Cosmology) transfer(k float64) float64 {
	h := c.H
	om, ob := c.OmegaM*h*h, c.OmegaB*h*h
	fb := c.OmegaB / c.OmegaM
	const theta = 2.7255 / 2.7
	s := 44.5 * math.Log(9.83/om) / math.Sqrt(1+10*math.Pow(ob, 0.75)) // sound horizon in Mpc
	alpha := 1 - 0.328*math.Log(431*om)*fb + 0.38*math.Log(22.3*om)*fb*fb
	kMpc := k * h
	gamma := c.OmegaM * h * (alpha + (1-alpha)/(1+math.Pow(0.43*kMpc*s, 4)))
	q := k * theta * theta / gamma
	l0 := math.Log(2*math.E + 1.8*q)
	c0 := 14.2 + 731/(1+62.5*q)
	return l0 / (l0 + c0*q*q)
}

// Power is the linear matter power spectrum today, P(k) in (Mpc/h)³ with k in h/Mpc, normalized
// to Sigma8.
type Power struct {
	c   Cosmology
	amp float64
}

// NewPower normalizes the spectrum to σ₈.
func NewPower(c Cosmology) Power {
	p := Power{c: c, amp: 1}
	s := p.Sigma(8)
	p.amp = c.Sigma8 * c.Sigma8 / (s * s)
	return p
}

// At returns P(k).
func (p Power) At(k float64) float64 {
	if k <= 0 {
		return 0
	}
	t := p.c.transfer(k)
	return p.amp * math.Pow(k, p.c.NS) * t * t
}

// Sigma returns the rms of the linear density today in top-hat spheres of radius r (Mpc/h):
// σ² = 1/(2π²) ∫ P(k) W²(kr) k² dk.
func (p Power) Sigma(r float64) float64 {
	const n = 4000
	lo, hi := math.Log(1e-4), math.Log(1e2)
	var sum float64
	for i := range n {
		k := math.Exp(lo + (hi-lo)*(float64(i)+0.5)/n)
		x := k * r
		w := 3 * (math.Sin(x) - x*math.Cos(x)) / (x * x * x)
		sum += p.At(k) * w * w * k * k * k // dk = k d ln k
	}
	return math.Sqrt(sum * (hi - lo) / n / (2 * math.Pi * math.Pi))
}
