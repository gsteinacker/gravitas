package sim

import (
	"fmt"
	"math"
)

// Finite propagation speed of the gravity of black holes (T-020, spike; ADR-0005).
//
// With a retarded mode a body feels a hole where the hole was at the retarded time t_r, given by
// the light cone |x − x_h(t_r)| = c (t − t_r). Naive retardation creates aberration: a body sees
// its partner lagging behind, which exerts a torque and destabilizes orbits (Laplace's argument).
// In general relativity velocity-dependent terms almost cancel this; for uniformly moving sources
// the field points at the position extrapolated from the retarded one (Carlip 2000). Mode
// GravityRetardedExtrapolated does exactly that. What remains in both modes is the propagation
// itself: the gravity of a newly placed hole spreads out as a spherical front at speed c.
//
// Only forces from holes are retarded; forces of stars on holes stay instantaneous. Momentum and
// energy are therefore not conserved in retarded modes (the field would carry them).

// GravityMode selects how the gravity of black holes propagates.
type GravityMode int

const (
	GravityInstant              GravityMode = iota // Newtonian, infinite speed
	GravityRetarded                                // retarded position (naive, unstable orbits)
	GravityRetardedExtrapolated                    // retarded position + v·Δt (aberration cancels)
)

// ParseGravityMode accepts the names printed by String.
func ParseGravityMode(s string) (GravityMode, error) {
	for _, g := range []GravityMode{GravityInstant, GravityRetarded, GravityRetardedExtrapolated} {
		if g.String() == s {
			return g, nil
		}
	}
	return GravityInstant, fmt.Errorf("unknown gravity mode %q (want instant, retarded, retarded+v)", s)
}

func (g GravityMode) String() string {
	switch g {
	case GravityRetarded:
		return "retarded"
	case GravityRetardedExtrapolated:
		return "retarded+v"
	default:
		return "instant"
	}
}

// historyLen is the number of past outer steps kept per hole (8192 · dt 0.004 ≈ 33 time units,
// i.e. light travel over 330 length units at c = 10). Older times use the oldest entry.
const historyLen = 8192

// The light-cone fixed point contracts with v/c per iteration, so near c it needs many; iterate
// to retardTolerance (in time units), at most retardMaxIterations.
const (
	retardTolerance     = 1e-7
	retardMaxIterations = 64
)

// holeHistory is a ring buffer of a hole's positions and coordinate velocities at outer steps.
type holeHistory struct {
	t0       float64 // time of the oldest entry still in the buffer
	appeared float64 // time the hole was placed; before that it did not exist (−Inf for scene holes)
	dt       float64
	n, head  int // entries in use, index of the newest entry
	x, y, z  [historyLen]float64
	vx, vy   [historyLen]float64
	vz       [historyLen]float64
}

func (h *holeHistory) push(t, x, y, z, vx, vy, vz float64) {
	h.head = (h.head + 1) % historyLen
	h.x[h.head], h.y[h.head], h.z[h.head] = x, y, z
	h.vx[h.head], h.vy[h.head], h.vz[h.head] = vx, vy, vz
	if h.n < historyLen {
		h.n++
	}
	h.t0 = t - float64(h.n-1)*h.dt
}

// at returns the interpolated state at time t. Before the oldest entry the motion is extrapolated
// linearly from it; ok is false only if the hole had not appeared yet at t.
func (h *holeHistory) at(t float64) (x, y, z, vx, vy, vz float64, ok bool) {
	if t < h.appeared {
		return 0, 0, 0, 0, 0, 0, false
	}
	oldest := (h.head - h.n + 1 + historyLen) % historyLen
	if t <= h.t0 || h.n == 1 {
		d := t - h.t0
		return h.x[oldest] + d*h.vx[oldest], h.y[oldest] + d*h.vy[oldest], h.z[oldest] + d*h.vz[oldest],
			h.vx[oldest], h.vy[oldest], h.vz[oldest], true
	}
	f := (t - h.t0) / h.dt
	k := min(int(f), h.n-2)
	w := f - float64(k)
	a, b := (oldest+k)%historyLen, (oldest+k+1)%historyLen
	lerp := func(p, q float64) float64 { return p + w*(q-p) }
	return lerp(h.x[a], h.x[b]), lerp(h.y[a], h.y[b]), lerp(h.z[a], h.z[b]),
		lerp(h.vx[a], h.vx[b]), lerp(h.vy[a], h.vy[b]), lerp(h.vz[a], h.vz[b]), true
}

func (m *Simulation) retarded() bool { return m.Params.Gravity != GravityInstant && m.Params.C > 0 }

// bindHistories looks up the history of every hole in m.holes once per step (holeHists[k] belongs
// to m.holes[k]), so the hot loops avoid map lookups.
func (m *Simulation) bindHistories() {
	m.holeHists = m.holeHists[:0]
	if !m.retarded() {
		return
	}
	for _, h := range m.holes {
		m.holeHists = append(m.holeHists, m.histories[m.State.ID[h]])
	}
}

// sourcePos returns where a body at (x, y, z) sees hole m.holes[k] now, and whether its gravity
// has arrived there yet.
func (m *Simulation) sourcePos(k int, x, y, z float64) (float64, float64, float64, bool) {
	s := m.State
	h := m.holes[k]
	hx, hy, hz := s.PosX[h], s.PosY[h], s.PosZ[h]
	if !m.retarded() {
		return hx, hy, hz, true
	}
	var hist *holeHistory
	if k < len(m.holeHists) {
		hist = m.holeHists[k]
	}
	c := m.Params.C
	delay := math.Sqrt(sq(x-hx, y-hy, z-hz)) / c
	if hist == nil || delay < m.Params.Dt {
		return hx, hy, hz, true // within one step the difference is below the step resolution
	}
	var vx, vy, vz float64
	for range retardMaxIterations {
		var ok bool
		hx, hy, hz, vx, vy, vz, ok = hist.at(m.Time - delay)
		if !ok {
			return 0, 0, 0, false
		}
		next := math.Sqrt(sq(x-hx, y-hy, z-hz)) / c
		converged := math.Abs(next-delay) < retardTolerance
		delay = next
		if converged {
			break
		}
	}
	hx, hy, hz, vx, vy, vz, _ = hist.at(m.Time - delay)
	if m.Params.Gravity == GravityRetardedExtrapolated {
		hx, hy, hz = hx+delay*vx, hy+delay*vy, hz+delay*vz
	}
	return hx, hy, hz, true
}

// holeAppeared starts the history of a hole placed during the run: its gravity starts spreading now.
func (m *Simulation) holeAppeared(i int) {
	h, _ := m.history(i)
	h.appeared = m.Time
}

// history returns the history of hole i, creating it with the current state as first entry.
func (m *Simulation) history(i int) (h *holeHistory, created bool) {
	if m.histories == nil {
		m.histories = map[uint64]*holeHistory{}
	}
	id := m.State.ID[i]
	h = m.histories[id]
	if h == nil {
		s := m.State
		h = &holeHistory{appeared: math.Inf(-1), dt: m.Params.Dt}
		v := CoordinateVelocity(s.Body(i).Vel, m.Params.C)
		h.push(m.Time, s.PosX[i], s.PosY[i], s.PosZ[i], v.X, v.Y, v.Z)
		m.histories[id] = h
		created = true
	}
	return h, created
}

// recordHoles appends the current state of every hole to its history (only in retarded modes).
func (m *Simulation) recordHoles() {
	if !m.retarded() {
		m.histories = nil
		return
	}
	s := m.State
	m.collectHoles()
	seen := len(m.histories) != len(m.holes)
	for _, i := range m.holes {
		// A new history already holds the current state; pushing it twice would shift the time axis.
		if h, created := m.history(i); !created {
			v := CoordinateVelocity(s.Body(i).Vel, m.Params.C)
			h.push(m.Time, s.PosX[i], s.PosY[i], s.PosZ[i], v.X, v.Y, v.Z)
		}
	}
	if seen { // drop histories of holes that merged away
		alive := map[uint64]bool{}
		for _, i := range m.holes {
			alive[s.ID[i]] = true
		}
		for id := range m.histories {
			if !alive[id] {
				delete(m.histories, id)
			}
		}
	}
}
