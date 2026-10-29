package sim

import "math"

// Comoving integration of an expanding ΛCDM universe (T-074, ADR-0014). Positions x are comoving
// in a periodic cube of side Box centered on the origin, the stored velocity is the canonical
// momentum per mass u = a² dx/dt, and the forces come from the peculiar potential of the comoving
// density contrast, ∇²φ = 4πG (ρ − ρ̄) (periodic, Comoving solver). Then
//
//	dx/dt = u / a²,   du/dt = −∇φ / a,
//
// integrated as a kick-drift-kick leapfrog in steps of Δ ln a = Params.Dt, with the exact factors
// ∫dt/a and ∫dt/a² over each part (Quinn et al. 1997). Holes, gas and the speed of light are not
// used in this mode.

// Cosmology configures the comoving mode; nil in ordinary scenes.
type Cosmology struct {
	OmegaM float64 `json:"omegaM"` // matter density today (flat ΛCDM: Ω_Λ = 1 − Ω_m)
	H0     float64 `json:"h0"`     // Hubble constant in simulation units (1/time)
	Box    float64 `json:"box"`    // comoving side of the periodic cube
	AStart float64 `json:"aStart"` // scale factor at the start
	AEnd   float64 `json:"aEnd"`   // the run stops here (0 means 1, today)
}

// Hubble returns H(a) for flat ΛCDM.
func (c *Cosmology) Hubble(a float64) float64 {
	return c.H0 * math.Sqrt(c.OmegaM/(a*a*a)+1-c.OmegaM)
}

// factor returns ∫ dt / a^k from a0 to a1 (k = 1 for kicks, 2 for drifts), with dt = da/(a H).
func (c *Cosmology) factor(a0, a1 float64, k int) float64 {
	const n = 16 // Simpson in ln a, the integrand is smooth
	l0, l1 := math.Log(a0), math.Log(a1)
	h := (l1 - l0) / n
	f := func(l float64) float64 {
		a := math.Exp(l)
		return 1 / (c.Hubble(a) * math.Pow(a, float64(k)))
	}
	sum := f(l0) + f(l1)
	for i := 1; i < n; i++ {
		w := 2.0
		if i%2 == 1 {
			w = 4
		}
		sum += w * f(l0+float64(i)*h)
	}
	return sum * h / 3
}

// ComovingSolver builds the periodic force solver for the comoving mode for n bodies; set by the
// package that provides it (internal/cosmo), since sim must not depend on it.
var ComovingSolver func(p Params, n int) Solver

// Finished reports whether a comoving run has reached its end (today): further steps do nothing.
func (m *Simulation) Finished() bool {
	cp := m.Params.Cosmology
	if cp == nil {
		return false
	}
	end := cp.AEnd
	if end == 0 {
		end = 1
	}
	return m.ScaleFactor >= end
}

// stepComoving advances the scale factor by Δ ln a = Params.Dt (stopping at AEnd).
func (m *Simulation) stepComoving() {
	cp, s := m.Params.Cosmology, m.State
	end := cp.AEnd
	if end == 0 {
		end = 1
	}
	if m.ScaleFactor == 0 {
		m.ScaleFactor = cp.AStart
	}
	a0 := m.ScaleFactor
	if a0 >= end {
		return
	}
	a1 := math.Min(end, a0*math.Exp(m.Params.Dt))
	am := math.Sqrt(a0 * a1)
	if m.comoving == nil {
		if ComovingSolver == nil {
			panic("sim: comoving mode without a solver (import internal/cosmo)")
		}
		m.comoving = ComovingSolver(m.Params, s.Len())
	}
	if !m.accValid {
		m.comoving.Accelerations(s, m.Params)
		m.accValid = true
	}
	kick(s, cp.factor(a0, am, 1))
	drift(s, cp.factor(a0, a1, 2), 0, false)
	m.wrap()
	m.comoving.Accelerations(s, m.Params)
	kick(s, cp.factor(am, a1, 1))
	m.Time += cp.factor(a0, a1, 0)
	m.ScaleFactor = a1
	m.Substeps = 1
}

// wrap puts every body back into the periodic cube [−Box/2, Box/2)³.
func (m *Simulation) wrap() {
	s, l := m.State, m.Params.Cosmology.Box
	w := func(x float64) float64 { return x - l*math.Floor(x/l+0.5) }
	for i := range s.Len() {
		s.PosX[i], s.PosY[i], s.PosZ[i] = w(s.PosX[i]), w(s.PosY[i]), w(s.PosZ[i])
	}
}
