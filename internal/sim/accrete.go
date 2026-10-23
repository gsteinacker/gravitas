package sim

import (
	"math"
	"sync"
)

// accrete lets black holes swallow every star inside their Schwarzschild radius and merges
// black holes whose horizons touch. Rest mass and momentum m·u are conserved; the position is
// the center of mass. Each event's exact energy change is added to ExternalEnergy.
//
// Accelerations are kept after star absorption (the field changes only by the moved mass of one
// star) but recomputed after a black hole merger.
func (m *Simulation) accrete() {
	s, c := m.State, m.Params.C
	m.holes = m.holes[:0]
	for i := range s.Len() {
		if s.Alive[i] && s.Kind[i] == BlackHole {
			m.holes = append(m.holes, i)
		}
	}
	if len(m.holes) == 0 {
		return
	}
	changed := false
	for a, h := range m.holes {
		for _, g := range m.holes[a+1:] {
			if !s.Alive[h] || !s.Alive[g] {
				continue
			}
			reach := SchwarzschildRadius(s.Mass[h], c) + SchwarzschildRadius(s.Mass[g], c)
			if dist2(s, h, g) < reach*reach {
				survivor, victim := h, g
				if s.Mass[g] > s.Mass[h] {
					survivor, victim = g, h
				}
				m.merge(survivor, victim)
				m.accValid, changed = false, true
			}
		}
	}
	for _, h := range m.holes {
		if !s.Alive[h] {
			continue
		}
		for i := range s.Len() {
			if s.Kind[i] != Star || !s.Alive[i] {
				continue
			}
			rs := SchwarzschildRadius(s.Mass[h], c) // grows with every absorbed star
			if dist2(s, h, i) < rs*rs {
				m.merge(h, i)
				changed = true
			}
		}
	}
	if changed {
		s.Compact()
	}
}

func dist2(s *State, i, j int) float64 {
	dx, dy, dz := s.PosX[i]-s.PosX[j], s.PosY[i]-s.PosY[j], s.PosZ[i]-s.PosZ[j]
	return dx*dx + dy*dy + dz*dz
}

// merge lets body h absorb body i and books the exact energy change.
func (m *Simulation) merge(h, i int) {
	s, c := m.State, m.Params.C
	eps2 := m.Params.Softening * m.Params.Softening
	mh, mi := s.Mass[h], s.Mass[i]
	mn := mh + mi
	xh, yh, zh := s.PosX[h], s.PosY[h], s.PosZ[h]
	xi, yi, zi := s.PosX[i], s.PosY[i], s.PosZ[i]
	xn, yn, zn := (mh*xh+mi*xi)/mn, (mh*yh+mi*yi)/mn, (mh*zh+mi*zi)/mn
	ux, uy, uz := (mh*s.VelX[h]+mi*s.VelX[i])/mn, (mh*s.VelY[h]+mi*s.VelY[i])/mn, (mh*s.VelZ[h]+mi*s.VelZ[i])/mn

	dK := kinetic(mn, ux, uy, uz, c) - kinetic(mh, s.VelX[h], s.VelY[h], s.VelZ[h], c) -
		kinetic(mi, s.VelX[i], s.VelY[i], s.VelZ[i], c)
	// Potential: the pair term (h, i) disappears, and every other body j now sees mass mn at
	// the center of mass instead of mh and mi at their old positions.
	dU := mh * mi / math.Sqrt(dist2(s, h, i)+eps2)
	var mu sync.Mutex
	ParallelFor(s.Len(), 4096, 0, func(lo, hi int) {
		var part float64
		for j := lo; j < hi; j++ {
			if j == h || j == i || !s.Alive[j] {
				continue
			}
			x, y, z := s.PosX[j], s.PosY[j], s.PosZ[j]
			part += s.Mass[j] * (mh/dist(x-xh, y-yh, z-zh, eps2) + mi/dist(x-xi, y-yi, z-zi, eps2) -
				mn/dist(x-xn, y-yn, z-zn, eps2))
		}
		mu.Lock()
		dU += part
		mu.Unlock()
	})
	m.ExternalEnergy += dK + dU

	s.PosX[h], s.PosY[h], s.PosZ[h] = xn, yn, zn
	s.VelX[h], s.VelY[h], s.VelZ[h] = ux, uy, uz
	s.Mass[h] = mn
	s.Alive[i], s.Mass[i] = false, 0
}

func dist(dx, dy, dz, eps2 float64) float64 { return math.Sqrt(dx*dx + dy*dy + dz*dz + eps2) }
