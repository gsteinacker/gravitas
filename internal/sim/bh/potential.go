package bh

import (
	"math"
	"sync"

	"galaxy/internal/sim"
)

// PotentialEnergy estimates the total softened potential energy with a tree walk, using an
// opening angle of at most diagTheta. Cost is a few force evaluations.
func (b *Solver) PotentialEnergy(s *sim.State, p sim.Params) float64 {
	b.build(s, min(b.theta(), diagTheta))
	eps2 := p.Softening * p.Softening
	var mu sync.Mutex
	var sum float64 // Σ m_i ψ_i with ψ_i = Σ_j m_j / r_ij, so U = -sum / 2
	sim.ParallelFor(len(b.order), grain, b.Workers, func(lo, hi int) {
		var part float64
		for k := lo; k < hi; k++ {
			psi := b.walkPotential(b.sx[k], b.sy[k], b.sz[k], int32(k), eps2) + b.holePotential(s, b.order[k], eps2)
			part += b.sm[k] * psi
		}
		mu.Lock()
		sum += part
		mu.Unlock()
	})
	for _, i := range b.holes {
		psi := b.walkPotential(s.PosX[i], s.PosY[i], s.PosZ[i], -1, eps2) + b.holePotential(s, i, eps2)
		sum += s.Mass[i] * psi
	}
	return -sum / 2
}

// walkPotential mirrors walk, accumulating Σ m / r instead of the force.
func (b *Solver) walkPotential(x, y, z float64, self int32, eps2 float64) float64 {
	if len(b.nodes) == 0 {
		return 0
	}
	nodes, sx, sy, sz, sm := b.nodes, b.sx, b.sy, b.sz, b.sm
	var psi float64
	for i := int32(0); i >= 0; {
		nd := &nodes[i]
		dx, dy, dz := nd.x-x, nd.y-y, nd.z-z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 >= nd.crit2 {
			psi += nd.m / math.Sqrt(d2+eps2)
			i = nd.next
			continue
		}
		if nd.first >= 0 {
			i = nd.first
			continue
		}
		for j := nd.lo; j < nd.hi; j++ {
			if j == self {
				continue
			}
			dx, dy, dz := sx[j]-x, sy[j]-y, sz[j]-z
			psi += sm[j] / math.Sqrt(dx*dx+dy*dy+dz*dz+eps2)
		}
		i = nd.next
	}
	return psi
}

func (b *Solver) holePotential(s *sim.State, i int32, eps2 float64) float64 {
	x, y, z := s.PosX[i], s.PosY[i], s.PosZ[i]
	var psi float64
	for _, h := range b.holes {
		if h == i {
			continue
		}
		dx, dy, dz := s.PosX[h]-x, s.PosY[h]-y, s.PosZ[h]-z
		psi += s.Mass[h] / math.Sqrt(dx*dx+dy*dy+dz*dz+eps2)
	}
	return psi
}
