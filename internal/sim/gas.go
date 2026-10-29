package sim

import (
	"math"
	"sync"
)

// Gas model glue (ADR-0007): the Hydro adds pressure and viscosity to the slow forces; after every
// step the internal energy is advanced, relaxed towards the cooling floor, and dense, cold,
// converging gas turns into stars.

// initForces evaluates gravity and the hydrodynamics of all gas, at the start of a run and after
// the bodies changed.
func (m *Simulation) initForces() {
	if m.Hydro == nil || !m.hasGas() {
		m.Solver.Accelerations(m.State, m.Params)
		return
	}
	m.hydro.Resize(m.State.Len())
	// Gravity and hydro write disjoint fields, so the hydro runs while the gravity solver works
	// (on the GPU the CPU would otherwise wait).
	var wg sync.WaitGroup
	wg.Go(func() { m.Solver.Accelerations(m.State, m.Params) })
	m.Hydro.Forces(m.State, &m.Params, nil, nil, &m.hydro)
	wg.Wait()
}

func (m *Simulation) hasGas() bool {
	for _, k := range m.State.Kind {
		if k == Gas {
			return true
		}
	}
	return false
}

const (
	defaultSFEfficiency = 0.02
	defaultYoungAge     = 5.0
)

// updateGas relaxes internal energies (implicit cooling, which is stable for any time step; the
// hydro heating is integrated in blockstep.go) and forms stars.
func (m *Simulation) updateGas(dt float64) {
	if m.Hydro == nil || len(m.hydro.DivV) != m.State.Len() {
		return
	}
	s, g := m.State, &m.Params.Gas
	eff := g.SFEfficiency
	if eff == 0 {
		eff = defaultSFEfficiency
	}
	decay := 0.0
	if g.CoolTime > 0 {
		decay = math.Exp(-dt / g.CoolTime)
	}
	if g.UIon > 0 {
		every := g.IonEvery
		if every <= 0 {
			every = 4
		}
		if m.Steps%every == 0 {
			m.ionize()
		}
	}
	unresolved := m.unresolvedDensity()
	now, step := m.Time, uint64(m.Steps)
	ParallelFor(s.Len(), 4096, 0, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			if s.Kind[i] != Gas {
				continue
			}
			u := s.U[i]
			// Relax towards the equilibrium temperature: photoheated where ionized, cooled elsewhere.
			target := g.UFloor
			if g.UIon > 0 {
				target = g.UFloor + s.Ion[i]*(g.UIon-g.UFloor)
			}
			if g.CoolTime > 0 && (u > target || g.UIon > 0) {
				u = target + (u-target)*decay
			}
			s.U[i] = u
			if g.SFDensity > 0 && s.Rho[i] > g.SFDensity && m.hydro.DivV[i] < 0 && s.Ion[i] < 0.5 &&
				(u < 3*g.UFloor || s.Rho[i] > unresolved*s.Mass[i]) {
				tff := math.Sqrt(3 * math.Pi / (32 * s.Rho[i]))
				// Converging gas denser than the SPH resolves (ADR-0017) collapses below the
				// resolution anyway: it becomes a star at once, whatever its temperature.
				if s.Rho[i] > unresolved*s.Mass[i] || unitRandom(s.ID[i], step) < 1-math.Exp(-eff*dt/tff) {
					s.Kind[i], s.Born[i], s.U[i] = Star, now, 0
				}
			}
		}
	})
	// Count the new stars in index order, so the sums do not depend on the scheduling.
	for i := range s.Len() {
		if s.Kind[i] == Star && s.Born[i] == now {
			m.StarsFormed++
			m.FormedMass += s.Mass[i]
		}
	}
}

// unresolvedDensity is the density per unit body mass above which gas is not resolved: its SPH
// smoothing length h = ½ (3 N m / 4π ρ)^⅓ would fall below the floor ε/4 (internal/sph), i.e.
// below the gravitational resolution. +Inf without softening.
func (m *Simulation) unresolvedDensity() float64 {
	eps := m.Params.Softening
	if eps <= 0 {
		return math.Inf(1)
	}
	ngb := m.Params.Gas.Neighbors
	if ngb == 0 {
		ngb = 40 // sph default
	}
	return 3 * ngb / (4 * math.Pi * math.Pow(eps/2, 3))
}

// unitRandom is a deterministic uniform number in [0, 1) for (id, step): star formation stays
// reproducible, also across save and load.
func unitRandom(id, step uint64) float64 {
	x := id*0x9e3779b97f4a7c15 ^ step*0xbf58476d1ce4e5b9
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return float64(x>>11) / (1 << 53)
}

// thermalEnergy is Σ m u over gas bodies.
func thermalEnergy(s *State) float64 {
	var e float64
	for i := range s.Len() {
		if s.Kind[i] == Gas {
			e += s.Mass[i] * s.U[i]
		}
	}
	return e
}
