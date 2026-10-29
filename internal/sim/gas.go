package sim

import "math"

// Gas model glue (ADR-0007): the Hydro adds pressure and viscosity to the slow forces; after every
// step the internal energy is advanced, relaxed towards the cooling floor, and dense, cold,
// converging gas turns into stars.

// computeSlow computes the forces that are evaluated once per outer step: star–star gravity from
// the Solver plus hydrodynamics of gas.
func (m *Simulation) computeSlow() {
	m.Solver.Accelerations(m.State, m.Params)
	if m.Hydro == nil || !m.hasGas() {
		return
	}
	n := m.State.Len()
	m.dudt, m.divv = resize(m.dudt, n), resize(m.divv, n)
	m.Hydro.Forces(m.State, &m.Params, m.dudt, m.divv)
	m.hydroDt = m.Hydro.Timestep()
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

// updateGas advances internal energies (explicit hydro heating plus implicit cooling, which is
// stable for any time step) and forms stars.
func (m *Simulation) updateGas(dt float64) {
	if m.Hydro == nil || len(m.dudt) != m.State.Len() {
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
	for i := range s.Len() {
		if s.Kind[i] != Gas {
			continue
		}
		u := math.Max(s.U[i]+dt*m.dudt[i], 1e-3*g.UFloor)
		// Relax towards the equilibrium temperature: photoheated where ionized, cooled elsewhere.
		target := g.UFloor
		if g.UIon > 0 {
			target = g.UFloor + s.Ion[i]*(g.UIon-g.UFloor)
		}
		if g.CoolTime > 0 && (u > target || g.UIon > 0) {
			u = target + (u-target)*decay
		}
		s.U[i] = u
		if g.SFDensity > 0 && s.Rho[i] > g.SFDensity && m.divv[i] < 0 && u < 3*g.UFloor && s.Ion[i] < 0.5 {
			tff := math.Sqrt(3 * math.Pi / (32 * s.Rho[i]))
			if unitRandom(s.ID[i], uint64(m.Steps)) < 1-math.Exp(-eff*dt/tff) {
				s.Kind[i], s.Born[i], s.U[i] = Star, m.Time, 0
				m.StarsFormed++
				m.FormedMass += s.Mass[i]
			}
		}
	}
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
