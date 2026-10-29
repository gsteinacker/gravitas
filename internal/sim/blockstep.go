package sim

import (
	"math"
	"sync"
	"sync/atomic"
)

// Individual time steps for gas (T-085, ADR-0018).
//
// Gravity acts at the ends of the global step Dt (the outer kicks), as for stars. Within the step,
// every gas body moves with its own step Dt/2^r (its rung r), chosen from the Courant condition
// and limited by its neighbors, and receives pressure and viscosity forces in a kick-drift-kick
// leapfrog of its own (block time steps, as in GADGET). All gas drifts together from event to
// event; only the bodies whose step ends are evaluated. Inactive neighbors enter with density and
// smoothing length predicted from their velocity divergence. A body whose neighbor gets a much
// shorter step is woken up: its step ends early (Saitoh & Makino 2009), otherwise a shock would
// run into it unnoticed. At the end of the global step all
// gas is synchronized and evaluated together (concurrently with gravity), so star formation,
// supernovae, infall and saving see a consistent state.

// maxRung: the shortest gas step is Dt/2^maxRung.
const maxRung = 10

// fullTicks is Dt in units of the shortest gas step.
const fullTicks = 1 << maxRung

// gasSteps is the state of the individual gas steps within a global step, per body.
type gasSteps struct {
	own        bool    // gas moves in its own steps (hydro present, gas evaluated)
	idx        []int32 // gas bodies
	start, end []int32 // start and end of the current step in ticks
	step       []float64
	active     []bool
}

// rungFor returns the smallest rung r whose step T/2^r is at most dt (capped at maxRung).
func rungFor(dt, T float64) int {
	r, x := 0, T
	for x > dt && r < maxRung {
		x *= 0.5
		r++
	}
	return r
}

// collectGas lists the gas bodies and reports whether they move in their own steps.
func (m *Simulation) collectGas() bool {
	s, g, n := m.State, &m.gas, m.State.Len()
	g.idx = g.idx[:0]
	if m.Hydro != nil && len(m.hydro.Dt) == n {
		for i := range n {
			if s.Kind[i] == Gas && s.Alive[i] {
				g.idx = append(g.idx, int32(i))
			}
		}
	}
	g.own = len(g.idx) > 0
	if g.own {
		g.start, g.end, g.active = resize(g.start, n), resize(g.end, n), resize(g.active, n)
		old := len(g.step)
		g.step = resize(g.step, n)
		if n > old {
			clear(g.step[old:])
		}
	}
	return g.own
}

// kickGas applies the hydro forces to gas body i for h: velocity and internal energy.
func (m *Simulation) kickGas(i int32, h, uMin float64) {
	s, o := m.State, &m.hydro
	s.VelX[i] += h * o.AccX[i]
	s.VelY[i] += h * o.AccY[i]
	s.VelZ[i] += h * o.AccZ[i]
	s.U[i] = math.Max(s.U[i]+h*o.DuDt[i], uMin)
}

// startStep gives gas body i a new step at tick now (aligned to the block structure: a step of
// T/2^r can only start at a multiple of its length) and applies its opening kick.
func (m *Simulation) startStep(i int32, now int32, T, uMin float64) {
	g := &m.gas
	r := rungFor(m.hydro.Dt[i], T)
	for now%(fullTicks>>r) != 0 {
		r++
	}
	g.start[i], g.end[i] = now, now+fullTicks>>r
	g.step[i] = T / float64(int(1)<<r)
	m.kickGas(i, 0.5*g.step[i], uMin)
}

// openGas starts the gas steps of a global step of length T: rungs from the last evaluation and
// the opening kicks. Call it after the outer gravity kick, before anything drifts.
func (m *Simulation) openGas(T float64) {
	m.Hydro.Bound(m.State, &m.Params, m.hydro.Dt)
	g, uMin := &m.gas, 1e-3*m.Params.Gas.UFloor
	ParallelFor(len(g.idx), 4096, 0, func(lo, hi int) {
		for _, i := range g.idx[lo:hi] {
			m.startStep(i, 0, T, uMin)
		}
	})
}

// wake ends the step of inactive gas body i early, because a neighbor got a much shorter step:
// at the first tick after now that suits the step the limiter allows. The opening kick for the
// old step length is corrected to the shortened one.
func (m *Simulation) wake(i int32, now int32, T, uMin float64) {
	g := &m.gas
	limit := LimiterRatio * math.Float64frombits(m.hydro.Wake[i])
	if g.step[i] <= limit {
		return
	}
	length := int32(fullTicks >> rungFor(limit, T))
	end := (now/length + 1) * length
	if end >= g.end[i] {
		return
	}
	unit := T / fullTicks
	m.kickGas(i, 0.5*float64(end-g.end[i])*unit, uMin)
	g.end[i] = end
	g.step[i] = float64(end-g.start[i]) * unit
}

// advanceGas moves the gas through the global step of length T (after openGas): from event to
// event, every gas body drifts (except those in skip, which move elsewhere) and those whose step
// ends are evaluated, kicked and given a new step. The gravity solver runs concurrently with the
// last evaluation, when all bodies have reached the end of the step. Without own gas steps it
// only runs the gravity solver.
func (m *Simulation) advanceGas(T float64, skip []bool) {
	g := &m.gas
	if !g.own {
		m.Solver.Accelerations(m.State, m.Params)
		m.HydroSubsteps, m.HydroLoad = 0, 0
		return
	}
	s, c, uMin := m.State, m.Params.C, 1e-3*m.Params.Gas.UFloor
	unit := T / fullTicks
	events, load := 0, 0.0
	for now := int32(0); now < fullTicks; {
		next := int32(fullTicks)
		for _, i := range g.idx {
			next = min(next, g.end[i])
		}
		dt := float64(next-now) * unit
		now = next
		var active atomic.Int64
		ParallelFor(len(g.idx), 2048, 0, func(lo, hi int) {
			var count int64
			for _, i := range g.idx[lo:hi] {
				if skip == nil || !skip[i] {
					driftOne(s, int(i), dt, c)
				}
				// Predict density and smoothing length for the bodies that are only neighbors:
				// dρ/dt = −ρ ∇·v.
				if d := m.hydro.DivV[i] * dt; d != 0 {
					s.Rho[i] *= math.Exp(-d)
					s.H[i] *= math.Exp(d / 3)
				}
				m.hydro.Wake[i] = noWake
				g.active[i] = g.end[i] == now
				if g.active[i] {
					count++
				}
			}
			active.Add(count)
		})
		if now == fullTicks {
			var wg sync.WaitGroup
			wg.Go(func() { m.Solver.Accelerations(s, m.Params) })
			m.Hydro.Forces(s, &m.Params, nil, g.step, &m.hydro)
			wg.Wait()
		} else {
			m.Hydro.Forces(s, &m.Params, g.active, g.step, &m.hydro)
		}
		events++
		load += float64(active.Load()) / float64(len(g.idx))
		ParallelFor(len(g.idx), 4096, 0, func(lo, hi int) {
			for _, i := range g.idx[lo:hi] {
				switch {
				case g.active[i]:
					m.kickGas(i, 0.5*g.step[i], uMin)
					if now < fullTicks {
						m.startStep(i, now, T, uMin)
					}
				case m.hydro.Wake[i] != noWake && now < fullTicks:
					m.wake(i, now, T, uMin)
				}
			}
		})
	}
	m.HydroSubsteps, m.HydroLoad = events, load
}
