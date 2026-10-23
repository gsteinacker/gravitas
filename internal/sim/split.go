package sim

import (
	"math"
	"sync"
)

// Multiple time stepping near black holes (T-025, an impulse/RESPA scheme).
//
// Forces are split by kind: star–star forces come from the Solver ("slow", once per step);
// every force involving a hole is summed here directly (O(N·N_holes)). Stars whose timescale at a
// hole, τ = √(r/a), is below nearSteps·dt are "near": they and all holes receive hole forces in n
// substeps of dt/n, with n chosen per step from the shortest τ. Far stars receive hole forces at
// the outer kicks. Both partners of every pair are always kicked at the same times with the same
// force, so momentum is conserved exactly, and each level is a time-reversible KDK leapfrog.

const (
	// nearSteps: a star is near a hole if its timescale there is shorter than this many outer steps.
	nearSteps = 20.0
	// substepsPerTau: substeps resolve the shortest timescale with at least this many steps.
	substepsPerTau = 20.0
	// maxSubsteps caps the work per outer step; beyond it accuracy degrades gracefully.
	maxSubsteps = 256
)

// holeTimescale returns √(r_ε / a) for body i in the field of hole h.
func (m *Simulation) holeTimescale(i, h int, eps2 float64) float64 {
	s := m.State
	r2 := dist2(s, i, h)
	rs := PairRS(s.Kind[i], s.Mass[i], BlackHole, s.Mass[h], m.Params.C)
	a := PairForce(s.Mass[h], r2, eps2, rs) * math.Sqrt(r2)
	if a == 0 {
		return math.Inf(1)
	}
	return math.Sqrt(math.Sqrt(r2+eps2) / a)
}

// classify marks near stars, collects them in m.near and returns the number of substeps.
func (m *Simulation) classify(eps2 float64) int {
	s, n := m.State, m.State.Len()
	m.isNear = resize(m.isNear, n)
	limit := nearSteps * m.Params.Dt
	tauMin := math.Inf(1)
	var mu sync.Mutex
	ParallelFor(n, 2048, 0, func(lo, hi int) {
		local := math.Inf(1)
		for i := lo; i < hi; i++ {
			near := false
			if s.Kind[i] == Star {
				for _, h := range m.holes {
					if tau := m.holeTimescale(i, h, eps2); tau < limit {
						near, local = true, math.Min(local, tau)
					}
				}
			}
			m.isNear[i] = near
		}
		mu.Lock()
		tauMin = math.Min(tauMin, local)
		mu.Unlock()
	})
	for a, h := range m.holes {
		for _, g := range m.holes[a+1:] {
			if tau := m.holeTimescale(h, g, eps2); tau < limit {
				tauMin = math.Min(tauMin, tau)
			}
		}
	}
	m.near = m.near[:0]
	for i, near := range m.isNear {
		if near {
			m.near = append(m.near, i)
		}
	}
	if math.IsInf(tauMin, 1) {
		return 1
	}
	k := 1
	for float64(k)*tauMin < substepsPerTau*m.Params.Dt && k < maxSubsteps {
		k *= 2
	}
	return k
}

// outerHoleForces fills m.far*: for far stars the pull of all holes, for holes the pull of all
// far stars.
func (m *Simulation) outerHoleForces(eps2 float64) {
	s, n, c := m.State, m.State.Len(), m.Params.C
	m.farX, m.farY, m.farZ = zeroed(m.farX, n), zeroed(m.farY, n), zeroed(m.farZ, n)
	ParallelFor(n, 2048, 0, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			if s.Kind[i] != Star || m.isNear[i] {
				continue
			}
			m.farX[i], m.farY[i], m.farZ[i] = m.holePull(i, eps2)
		}
	})
	for _, h := range m.holes {
		x, y, z := s.PosX[h], s.PosY[h], s.PosZ[h]
		rs := SchwarzschildRadius(s.Mass[h], c)
		var mu sync.Mutex
		ParallelFor(n, 2048, 0, func(lo, hi int) {
			var ax, ay, az float64
			for i := lo; i < hi; i++ {
				if s.Kind[i] != Star || m.isNear[i] {
					continue
				}
				dx, dy, dz := s.PosX[i]-x, s.PosY[i]-y, s.PosZ[i]-z
				f := PairForce(s.Mass[i], dx*dx+dy*dy+dz*dz, eps2, rs)
				ax, ay, az = ax+f*dx, ay+f*dy, az+f*dz
			}
			mu.Lock()
			m.farX[h], m.farY[h], m.farZ[h] = m.farX[h]+ax, m.farY[h]+ay, m.farZ[h]+az
			mu.Unlock()
		})
	}
}

// holePull returns the acceleration of body i from all holes except itself.
func (m *Simulation) holePull(i int, eps2 float64) (ax, ay, az float64) {
	s := m.State
	for k, h := range m.holes {
		if h == i {
			continue
		}
		x, y, z := s.PosX[i], s.PosY[i], s.PosZ[i]
		hx, hy, hz, visible := m.sourcePos(k, x, y, z)
		if !visible {
			continue // the hole's gravity has not reached this body yet
		}
		dx, dy, dz := hx-x, hy-y, hz-z
		rs := PairRS(s.Kind[i], s.Mass[i], BlackHole, s.Mass[h], m.Params.C)
		f := PairForce(s.Mass[h], dx*dx+dy*dy+dz*dz, eps2, rs)
		ax, ay, az = ax+f*dx, ay+f*dy, az+f*dz
	}
	return ax, ay, az
}

// fastForces fills m.fast* for near stars (pull of all holes) and holes (pull of near stars and
// of the other holes).
func (m *Simulation) fastForces(eps2 float64) {
	s, c := m.State, m.Params.C
	m.fastX, m.fastY, m.fastZ = resize(m.fastX, s.Len()), resize(m.fastY, s.Len()), resize(m.fastZ, s.Len())
	ParallelFor(len(m.near), 512, 0, func(lo, hi int) {
		for _, i := range m.near[lo:hi] {
			m.fastX[i], m.fastY[i], m.fastZ[i] = m.holePull(i, eps2)
		}
	})
	for _, h := range m.holes {
		x, y, z := s.PosX[h], s.PosY[h], s.PosZ[h]
		rs := SchwarzschildRadius(s.Mass[h], c)
		ax, ay, az := m.holePull(h, eps2)
		for _, i := range m.near {
			dx, dy, dz := s.PosX[i]-x, s.PosY[i]-y, s.PosZ[i]-z
			f := PairForce(s.Mass[i], dx*dx+dy*dy+dz*dz, eps2, rs)
			ax, ay, az = ax+f*dx, ay+f*dy, az+f*dz
		}
		m.fastX[h], m.fastY[h], m.fastZ[h] = ax, ay, az
	}
}

// stepSplit is Step with holes present: outer KDK with slow and far forces, inner KDK substeps
// for near stars and holes.
func (m *Simulation) stepSplit() {
	s, dt, c := m.State, m.Params.Dt, m.Params.C
	eps2 := m.Params.Softening * m.Params.Softening
	if !m.accValid {
		m.Solver.Accelerations(s, m.Params)
		m.accValid = true
	}
	m.bindHistories()
	n := m.classify(eps2)
	m.Substeps = n
	m.outerHoleForces(eps2)
	m.kickOuter(0.5 * dt)

	h := dt / float64(n)
	m.fastForces(eps2)
	for range n {
		m.kickFast(0.5 * h)
		m.driftIdx(m.near, h, c)
		m.driftIdx(m.holes, h, c)
		m.fastForces(eps2)
		m.kickFast(0.5 * h)
	}
	m.driftFar(dt, c)

	m.Solver.Accelerations(s, m.Params)
	m.outerHoleForces(eps2)
	m.kickOuter(0.5 * dt)
}

func (m *Simulation) kickOuter(h float64) {
	s := m.State
	for i := range s.Len() {
		s.VelX[i] += h * (s.AccX[i] + m.farX[i])
		s.VelY[i] += h * (s.AccY[i] + m.farY[i])
		s.VelZ[i] += h * (s.AccZ[i] + m.farZ[i])
	}
}

func (m *Simulation) kickFast(h float64) {
	s := m.State
	for _, idx := range [][]int{m.near, m.holes} {
		for _, i := range idx {
			s.VelX[i] += h * m.fastX[i]
			s.VelY[i] += h * m.fastY[i]
			s.VelZ[i] += h * m.fastZ[i]
		}
	}
}

func (m *Simulation) driftIdx(idx []int, dt, c float64) {
	s := m.State
	for _, i := range idx {
		f := dt / gamma(s.VelX[i], s.VelY[i], s.VelZ[i], c)
		s.PosX[i] += f * s.VelX[i]
		s.PosY[i] += f * s.VelY[i]
		s.PosZ[i] += f * s.VelZ[i]
	}
}

func (m *Simulation) driftFar(dt, c float64) {
	s := m.State
	ParallelFor(s.Len(), 4096, 0, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			if s.Kind[i] != Star || m.isNear[i] {
				continue
			}
			f := dt / gamma(s.VelX[i], s.VelY[i], s.VelZ[i], c)
			s.PosX[i] += f * s.VelX[i]
			s.PosY[i] += f * s.VelY[i]
			s.PosZ[i] += f * s.VelZ[i]
		}
	})
}

func resize[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}

func zeroed(s []float64, n int) []float64 {
	s = resize(s, n)
	clear(s)
	return s
}
