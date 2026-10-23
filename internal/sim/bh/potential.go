package bh

import (
	"math"
	"sync"

	"galaxy/internal/sim"
)

// PotentialEnergy estimates the total softened potential energy with a tree walk, using an
// opening angle of at most diagTheta. Cost is a few force evaluations.
func (b *Solver) PotentialEnergy(s *sim.State, p sim.Params) float64 {
	return b.potentialWith(s, p, min(b.theta(), diagTheta))
}

func (b *Solver) potentialWith(s *sim.State, p sim.Params, theta float64) float64 {
	b.build(s, &p, theta)
	eps2 := p.Softening * p.Softening
	var mu sync.Mutex
	var sum float64 // −Σ_i Σ_j U_ij over ordered pairs, so U = −sum / 2
	sim.ParallelFor(len(b.order), grain, b.Workers, func(lo, hi int) {
		var part float64
		for k := lo; k < hi; k++ {
			i := b.order[k]
			part += b.sm[k]*b.walkPotential(b.sx[k], b.sy[k], b.sz[k], int32(k), eps2) - b.holePotential(s, &p, i, p.C, eps2)
		}
		mu.Lock()
		sum += part
		mu.Unlock()
	})
	// Each star–hole pair is counted once from the star's and once from the hole's side.
	for _, i := range b.holes {
		x, y, z := s.PosX[i], s.PosY[i], s.PosZ[i]
		rs := sim.SchwarzschildRadius(s.Mass[i], p.C)
		for k := range b.order {
			dx, dy, dz := b.sx[k]-x, b.sy[k]-y, b.sz[k]-z
			sum -= sim.PairPotential(s.Mass[i], b.sm[k], dx*dx+dy*dy+dz*dz, eps2, rs)
		}
		sum -= b.holePotential(s, &p, i, p.C, eps2)
	}
	return -sum / 2
}

// walkPotential mirrors walk, accumulating Σ m / r instead of the force.
func (b *Solver) walkPotential(x, y, z float64, self int32, eps2 float64) float64 {
	if len(b.nodes) == 0 {
		return 0
	}
	nodes, sx, sy, sz, sm := b.nodes, b.sx, b.sy, b.sz, b.sm
	quad := !b.MonopoleOnly
	var psi float64
	for i := int32(0); i >= 0; {
		nd := &nodes[i]
		dx, dy, dz := nd.x-x, nd.y-y, nd.z-z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 >= nd.crit2 {
			inv := 1 / (d2 + eps2)
			psi += nd.m * math.Sqrt(inv)
			if quad {
				// Φ = −M/r − (d·Q·d) / (2 r⁵); psi collects −Φ.
				dqd := nd.qxx*dx*dx + nd.qyy*dy*dy + nd.qzz*dz*dz + 2*(nd.qxy*dx*dy+nd.qxz*dx*dz+nd.qyz*dy*dz)
				psi += 0.5 * dqd * inv * inv * math.Sqrt(inv)
			}
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

// holePotential returns Σ_h U(i, h) over all black holes h ≠ i, using the pair model.
func (b *Solver) holePotential(s *sim.State, p *sim.Params, i int32, c, eps2 float64) float64 {
	x, y, z := s.PosX[i], s.PosY[i], s.PosZ[i]
	var u float64
	for _, h := range b.holes {
		if h == i {
			continue
		}
		dx, dy, dz := s.PosX[h]-x, s.PosY[h]-y, s.PosZ[h]-z
		rs := sim.PairRS(s.Kind[i], s.Mass[i], sim.BlackHole, s.Mass[h], c)
		u += sim.PairPotential(p.SourceMass(s.Kind[i], s.Mass[i]), s.Mass[h], dx*dx+dy*dy+dz*dz, eps2, rs)
	}
	return u
}
