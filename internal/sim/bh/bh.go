// Package bh implements a Barnes-Hut octree solver.
//
// Stars are sorted by Morton key, the tree is built top-down from the sorted ranges, and
// every node stores a skip pointer, so the force walk needs neither recursion nor a stack.
// Black holes are kept out of the tree; forces involving them are summed by the integrator.
package bh

import (
	"math"

	"galaxy/internal/sim"
)

const (
	// DefaultTheta is the opening angle used when Solver.Theta is 0.
	DefaultTheta = 0.9
	// leafSize is the maximum number of stars in a leaf. Measured on an evolved 100k galaxy:
	// 12 → 40.5 ms, 24 → 36.0 ms, 32 → 35.7 ms per force evaluation.
	leafSize = 24
	// diagTheta caps the opening angle for energy diagnostics: the potential's systematic error
	// must stay well below the integration drift it is meant to reveal.
	diagTheta = 0.6
	// grain is the minimum number of target bodies per parallel chunk.
	grain = 128
)

// Solver is a Barnes-Hut solver. All buffers are reused between calls.
type Solver struct {
	// Theta is the opening angle; smaller is more accurate. Must be in [0, 1].
	// Exactly 0 is treated as DefaultTheta; use a tiny value for "open everything".
	Theta   float64
	Workers int // 0 means GOMAXPROCS
	// MonopoleOnly drops the quadrupole term of accepted nodes (for comparisons).
	MonopoleOnly bool

	nodes      []node
	keys       []uint64
	order      []int32 // sorted position -> state index
	tmpK       []uint64
	tmpI       []int32
	sx, sy, sz []float64 // star positions and masses in sorted order
	sm         []float64
	holes      []int32 // state indices of black holes
}

type node struct {
	x, y, z, m float64 // center of mass and mass
	// Traceless quadrupole tensor about the center of mass: Q_ij = Σ m (3 d_i d_j − d² δ_ij).
	qxx, qyy, qzz, qxy, qxz, qyz float64
	crit2                        float64 // open the node if the squared distance to its center of mass is below this
	first                        int32   // first child, -1 for leaves
	next                         int32   // node to continue with after this subtree, -1 at the end
	lo, hi                       int32   // star range in sorted order
}

func (b *Solver) theta() float64 {
	if b.Theta == 0 {
		return DefaultTheta
	}
	return b.Theta
}

func (b *Solver) Accelerations(s *sim.State, p sim.Params) {
	b.build(s, b.theta())
	eps2 := p.Softening * p.Softening
	sim.ParallelFor(len(b.order), grain, b.Workers, func(lo, hi int) {
		for k := lo; k < hi; k++ {
			i := b.order[k]
			s.AccX[i], s.AccY[i], s.AccZ[i] = b.walk(b.sx[k], b.sy[k], b.sz[k], int32(k), eps2)
		}
	})
	// Forces involving holes are summed by the integrator (sim.Solver).
	for _, i := range b.holes {
		s.AccX[i], s.AccY[i], s.AccZ[i] = 0, 0, 0
	}
}

// walk returns the acceleration at (x, y, z) from all stars in the tree, skipping the star at
// sorted position self.
func (b *Solver) walk(x, y, z float64, self int32, eps2 float64) (ax, ay, az float64) {
	if len(b.nodes) == 0 {
		return 0, 0, 0
	}
	nodes, sx, sy, sz, sm := b.nodes, b.sx, b.sy, b.sz, b.sm
	quad := !b.MonopoleOnly
	for i := int32(0); i >= 0; {
		nd := &nodes[i]
		dx, dy, dz := nd.x-x, nd.y-y, nd.z-z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 >= nd.crit2 {
			inv := 1 / (d2 + eps2)
			inv3 := inv * math.Sqrt(inv)
			f := nd.m * inv3
			if quad {
				// a += −Q·d / r⁵ + (5/2) (d·Q·d) d / r⁷, d pointing from the star to the node.
				qx := nd.qxx*dx + nd.qxy*dy + nd.qxz*dz
				qy := nd.qxy*dx + nd.qyy*dy + nd.qyz*dz
				qz := nd.qxz*dx + nd.qyz*dy + nd.qzz*dz
				inv5 := inv3 * inv
				f += 2.5 * (dx*qx + dy*qy + dz*qz) * inv5 * inv
				ax -= qx * inv5
				ay -= qy * inv5
				az -= qz * inv5
			}
			ax += f * dx
			ay += f * dy
			az += f * dz
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
			r2 := dx*dx + dy*dy + dz*dz + eps2
			f := sm[j] / (r2 * math.Sqrt(r2))
			ax += f * dx
			ay += f * dy
			az += f * dz
		}
		i = nd.next
	}
	return ax, ay, az
}
