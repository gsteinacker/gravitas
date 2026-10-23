package sph

import (
	"math"
	"slices"
)

// The neighbor search uses a bounding volume hierarchy over the gas bodies in Morton order: an
// implicit binary tree over sorted index ranges, each node with its bounding box and the largest
// smoothing length inside. A query for body i visits a node if its box is within
// 2·max(h_i, hmax_node), so pairs with very different smoothing lengths still find each other.

const leafSize = 16

type node struct {
	minX, minY, minZ, maxX, maxY, maxZ float64
	hmax                               float64
	lo, hi                             int32 // range in order
	left, right                        int32 // children, -1 for leaves
}

type tree struct {
	order []int32 // gas body indices in Morton order
	keys  []uint64
	nodes []node
}

func spread(v uint64) uint64 {
	v &= 1<<21 - 1
	v = (v | v<<32) & 0x1f00000000ffff
	v = (v | v<<16) & 0x1f0000ff0000ff
	v = (v | v<<8) & 0x100f00f00f00f00f
	v = (v | v<<4) & 0x10c30c30c30c30c3
	v = (v | v<<2) & 0x1249249249249249
	return v
}

// build sorts the given bodies along a Morton curve and builds the hierarchy.
func (t *tree) build(gas []int32, x, y, z, h []float64) {
	minX, minY, minZ := math.Inf(1), math.Inf(1), math.Inf(1)
	maxX, maxY, maxZ := math.Inf(-1), math.Inf(-1), math.Inf(-1)
	for _, i := range gas {
		minX, maxX = math.Min(minX, x[i]), math.Max(maxX, x[i])
		minY, maxY = math.Min(minY, y[i]), math.Max(maxY, y[i])
		minZ, maxZ = math.Min(minZ, z[i]), math.Max(maxZ, z[i])
	}
	side := math.Max(math.Max(maxX-minX, maxY-minY), math.Max(maxZ-minZ, 1e-12)) * (1 + 1e-9)
	scale := float64(1<<21-1) / side
	t.order = append(t.order[:0], gas...)
	t.keys = slices.Grow(t.keys[:0], len(gas))[:len(gas)]
	for k, i := range t.order {
		t.keys[k] = spread(uint64((x[i]-minX)*scale))<<2 | spread(uint64((y[i]-minY)*scale))<<1 | spread(uint64((z[i]-minZ)*scale))
	}
	idx := make([]int, len(gas))
	for k := range idx {
		idx[k] = k
	}
	slices.SortFunc(idx, func(a, b int) int {
		switch {
		case t.keys[a] < t.keys[b]:
			return -1
		case t.keys[a] > t.keys[b]:
			return 1
		}
		return a - b
	})
	sorted := make([]int32, len(gas))
	for k, j := range idx {
		sorted[k] = t.order[j]
	}
	t.order = sorted
	t.nodes = t.nodes[:0]
	if len(gas) > 0 {
		t.buildNode(0, int32(len(gas)), x, y, z, h)
	}
}

func (t *tree) buildNode(lo, hi int32, x, y, z, h []float64) int32 {
	ni := int32(len(t.nodes))
	t.nodes = append(t.nodes, node{lo: lo, hi: hi, left: -1, right: -1})
	if hi-lo <= leafSize {
		t.refit(ni, x, y, z, h)
		return ni
	}
	mid := (lo + hi) / 2
	l := t.buildNode(lo, mid, x, y, z, h)
	r := t.buildNode(mid, hi, x, y, z, h)
	t.nodes[ni].left, t.nodes[ni].right = l, r
	t.refit(ni, x, y, z, h)
	return ni
}

// refit recomputes the box and hmax of node ni from its bodies or children.
func (t *tree) refit(ni int32, x, y, z, h []float64) {
	nd := &t.nodes[ni]
	nd.minX, nd.minY, nd.minZ = math.Inf(1), math.Inf(1), math.Inf(1)
	nd.maxX, nd.maxY, nd.maxZ = math.Inf(-1), math.Inf(-1), math.Inf(-1)
	nd.hmax = 0
	if nd.left < 0 {
		for _, i := range t.order[nd.lo:nd.hi] {
			nd.minX, nd.maxX = math.Min(nd.minX, x[i]), math.Max(nd.maxX, x[i])
			nd.minY, nd.maxY = math.Min(nd.minY, y[i]), math.Max(nd.maxY, y[i])
			nd.minZ, nd.maxZ = math.Min(nd.minZ, z[i]), math.Max(nd.maxZ, z[i])
			nd.hmax = math.Max(nd.hmax, h[i])
		}
		return
	}
	for _, c := range []*node{&t.nodes[nd.left], &t.nodes[nd.right]} {
		nd.minX, nd.maxX = math.Min(nd.minX, c.minX), math.Max(nd.maxX, c.maxX)
		nd.minY, nd.maxY = math.Min(nd.minY, c.minY), math.Max(nd.maxY, c.maxY)
		nd.minZ, nd.maxZ = math.Min(nd.minZ, c.minZ), math.Max(nd.maxZ, c.maxZ)
		nd.hmax = math.Max(nd.hmax, c.hmax)
	}
}

// refitH updates hmax after the smoothing lengths changed (children are built after parents, so a
// reverse sweep visits children first).
func (t *tree) refitH(h []float64) {
	for ni := len(t.nodes) - 1; ni >= 0; ni-- {
		nd := &t.nodes[ni]
		if nd.left < 0 {
			nd.hmax = 0
			for _, i := range t.order[nd.lo:nd.hi] {
				nd.hmax = math.Max(nd.hmax, h[i])
			}
		} else {
			nd.hmax = math.Max(t.nodes[nd.left].hmax, t.nodes[nd.right].hmax)
		}
	}
}

// each calls fn for every gas body j within radius of (px, py, pz). With symmetric set, the radius
// grows to 2·hmax of the visited nodes, i.e. j is reported if r < 2·max(hi, hj) (hi = radius/2).
func (t *tree) each(px, py, pz, radius float64, symmetric bool, x, y, z, h []float64, stack []int32, fn func(j int32, r2 float64)) []int32 {
	if len(t.nodes) == 0 {
		return stack
	}
	stack = append(stack[:0], 0)
	for len(stack) > 0 {
		nd := &t.nodes[stack[len(stack)-1]]
		stack = stack[:len(stack)-1]
		reach := radius
		if symmetric {
			reach = math.Max(reach, 2*nd.hmax)
		}
		dx := math.Max(0, math.Max(nd.minX-px, px-nd.maxX))
		dy := math.Max(0, math.Max(nd.minY-py, py-nd.maxY))
		dz := math.Max(0, math.Max(nd.minZ-pz, pz-nd.maxZ))
		if dx*dx+dy*dy+dz*dz >= reach*reach {
			continue
		}
		if nd.left >= 0 {
			stack = append(stack, nd.left, nd.right)
			continue
		}
		for _, j := range t.order[nd.lo:nd.hi] {
			ex, ey, ez := x[j]-px, y[j]-py, z[j]-pz
			r2 := ex*ex + ey*ey + ez*ez
			lim := radius
			if symmetric {
				lim = math.Max(lim, 2*h[j])
			}
			if r2 < lim*lim {
				fn(j, r2)
			}
		}
	}
	return stack
}
