package sph

import (
	"math"
	"math/bits"
	"slices"
	"sort"

	"galaxy/internal/sim"
)

// The neighbor search uses a bounding volume hierarchy over the gas bodies in Morton order: an
// implicit binary tree over sorted index ranges, each node with its bounding box and the largest
// smoothing length inside. A query for body i visits a node if its box is within
// 2·max(h_i, hmax_node), so pairs with very different smoothing lengths still find each other.

const leafSize = 8

type node struct {
	minX, minY, minZ, maxX, maxY, maxZ float64
	hmax                               float64
	lo, hi                             int32 // range in order
	left, right                        int32 // children, -1 for leaves
}

type tree struct {
	leaves []int32 // leaf nodes in tree order (group walks, T-086)
	order  []int32 // gas body indices in Morton order
	keys   []uint64
	nodes  []node
	// Copies of position and smoothing length in Morton order, for cache-friendly leaf scans.
	sx, sy, sz, sh []float64
	// Scratch of build, kept to avoid allocations per force evaluation.
	items, spare []keyed
}

// keyed is a body with its Morton key.
type keyed struct {
	key uint64
	i   int32
}

// radixBits is the digit width of radixSort: 6 passes cover the 63-bit Morton keys.
const radixBits = 11

// radixSort sorts items by key (stable, so equal keys keep the order of gas) with an LSD radix
// sort, using spare as buffer; it returns the sorted slice and the other buffer. Digits shared by
// all items (high bits of a compact region) are skipped. O(n) instead of the O(n log n) of a
// comparison sort, which made the tree build the serial bottleneck of the hydro (T-086).
func radixSort(items, spare []keyed) (sorted, buf []keyed) {
	if len(items) == 0 {
		return items, spare
	}
	var count [1 << radixBits]int
	for shift := uint(0); shift < 64; shift += radixBits {
		clear(count[:])
		for _, it := range items {
			count[(it.key>>shift)&(1<<radixBits-1)]++
		}
		if count[(items[0].key>>shift)&(1<<radixBits-1)] == len(items) {
			continue
		}
		sum := 0
		for d, c := range count {
			count[d] = sum
			sum += c
		}
		for _, it := range items {
			d := (it.key >> shift) & (1<<radixBits - 1)
			spare[count[d]] = it
			count[d]++
		}
		items, spare = spare, items
	}
	return items, spare
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
	n := len(gas)
	t.items = slices.Grow(t.items[:0], n)[:n]
	sim.ParallelFor(n, 4096, 0, func(lo, hi int) {
		for k := lo; k < hi; k++ {
			i := gas[k]
			key := spread(uint64((x[i]-minX)*scale))<<2 | spread(uint64((y[i]-minY)*scale))<<1 | spread(uint64((z[i]-minZ)*scale))
			t.items[k] = keyed{key, i}
		}
	})
	t.items, t.spare = radixSort(t.items, slices.Grow(t.spare[:0], n)[:n])
	t.order = slices.Grow(t.order[:0], n)[:n]
	t.keys = slices.Grow(t.keys[:0], n)[:n]
	for k, it := range t.items {
		t.order[k], t.keys[k] = it.i, it.key
	}
	t.sx, t.sy, t.sz, t.sh = resize(t.sx, len(gas)), resize(t.sy, len(gas)), resize(t.sz, len(gas)), resize(t.sh, len(gas))
	sim.ParallelFor(n, 4096, 0, func(lo, hi int) {
		for k := lo; k < hi; k++ {
			i := t.order[k]
			t.sx[k], t.sy[k], t.sz[k], t.sh[k] = x[i], y[i], z[i], h[i]
		}
	})
	t.nodes, t.leaves = t.nodes[:0], t.leaves[:0]
	if len(gas) > 0 {
		t.buildNode(0, int32(len(gas)), x, y, z, h)
	}
}

func (t *tree) buildNode(lo, hi int32, x, y, z, h []float64) int32 {
	ni := int32(len(t.nodes))
	t.nodes = append(t.nodes, node{lo: lo, hi: hi, left: -1, right: -1})
	if hi-lo <= leafSize {
		t.refit(ni, x, y, z, h)
		t.leaves = append(t.leaves, ni)
		return ni
	}
	mid := t.split(lo, hi)
	l := t.buildNode(lo, mid, x, y, z, h)
	r := t.buildNode(mid, hi, x, y, z, h)
	t.nodes[ni].left, t.nodes[ni].right = l, r
	t.refit(ni, x, y, z, h)
	return ni
}

// split returns where to divide the sorted range [lo, hi): at the highest Morton bit in which its
// keys differ, so that children are compact octree cells rather than arbitrary curve segments.
func (t *tree) split(lo, hi int32) int32 {
	a, b := t.keys[lo], t.keys[hi-1]
	if a == b {
		return (lo + hi) / 2
	}
	bit := uint64(1) << (63 - bits.LeadingZeros64(a^b))
	mid := lo + int32(sort.Search(int(hi-lo), func(k int) bool { return t.keys[lo+int32(k)]&bit != 0 }))
	return mid
}

// refit recomputes the box and hmax of node ni from its bodies or children.
func (t *tree) refit(ni int32, x, y, z, h []float64) {
	nd := &t.nodes[ni]
	nd.minX, nd.minY, nd.minZ = math.Inf(1), math.Inf(1), math.Inf(1)
	nd.maxX, nd.maxY, nd.maxZ = math.Inf(-1), math.Inf(-1), math.Inf(-1)
	nd.hmax = 0
	if nd.left < 0 { // from the sorted copies (build fills them first)
		for k := nd.lo; k < nd.hi; k++ {
			nd.minX, nd.maxX = min(nd.minX, t.sx[k]), max(nd.maxX, t.sx[k])
			nd.minY, nd.maxY = min(nd.minY, t.sy[k]), max(nd.maxY, t.sy[k])
			nd.minZ, nd.maxZ = min(nd.minZ, t.sz[k]), max(nd.maxZ, t.sz[k])
			nd.hmax = max(nd.hmax, t.sh[k])
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
	for k, i := range t.order {
		t.sh[k] = h[i]
	}
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
func (t *tree) each(px, py, pz, radius float64, symmetric bool, stack []int32, fn func(j int32, r2 float64)) []int32 {
	if len(t.nodes) == 0 {
		return stack
	}
	stack = append(stack[:0], 0)
	for len(stack) > 0 {
		nd := &t.nodes[stack[len(stack)-1]]
		stack = stack[:len(stack)-1]
		reach := radius
		if symmetric && 2*nd.hmax > reach {
			reach = 2 * nd.hmax
		}
		if boxDist2(nd, px, py, pz) >= reach*reach {
			continue
		}
		if nd.left >= 0 {
			stack = append(stack, nd.left, nd.right)
			continue
		}
		for k := nd.lo; k < nd.hi; k++ {
			ex, ey, ez := t.sx[k]-px, t.sy[k]-py, t.sz[k]-pz
			r2 := ex*ex + ey*ey + ez*ez
			lim := radius
			if symmetric && 2*t.sh[k] > lim {
				lim = 2 * t.sh[k]
			}
			if r2 < lim*lim {
				fn(t.order[k], r2)
			}
		}
	}
	return stack
}

// candidates collects the body ranges (positions in order) of all leaves that can hold neighbors of
// any body in leaf: within 2·hmax of the leaf (gather), or with symmetric set, within
// 2·max(hmax of the leaf, hmax of the visited node). One walk serves all bodies of the leaf, which
// then scan the ranges (group walk).
func (t *tree) candidates(leaf int32, symmetric bool, stack []int32, ranges [][2]int32) ([]int32, [][2]int32) {
	lf := &t.nodes[leaf]
	ranges = ranges[:0]
	stack = append(stack[:0], 0)
	for len(stack) > 0 {
		nd := &t.nodes[stack[len(stack)-1]]
		stack = stack[:len(stack)-1]
		reach := 2 * lf.hmax
		if symmetric && 2*nd.hmax > reach {
			reach = 2 * nd.hmax
		}
		if boxBoxDist2(nd, lf) >= reach*reach {
			continue
		}
		if nd.left >= 0 {
			stack = append(stack, nd.left, nd.right)
			continue
		}
		if n := len(ranges); n > 0 && ranges[n-1][1] == nd.lo { // adjacent leaves: one range
			ranges[n-1][1] = nd.hi
		} else {
			ranges = append(ranges, [2]int32{nd.lo, nd.hi})
		}
	}
	return stack, ranges
}

// scan calls fn for every body in ranges within radius of (px, py, pz), with symmetric set within
// 2·max(radius/2, h_j), like each.
func (t *tree) scan(ranges [][2]int32, px, py, pz, radius float64, symmetric bool, fn func(j int32, r2 float64)) {
	r2max := radius * radius
	for _, rg := range ranges {
		for k := rg[0]; k < rg[1]; k++ {
			ex, ey, ez := t.sx[k]-px, t.sy[k]-py, t.sz[k]-pz
			r2 := ex*ex + ey*ey + ez*ez
			lim2 := r2max
			if symmetric {
				if l := 2 * t.sh[k]; l*l > lim2 {
					lim2 = l * l
				}
			}
			if r2 < lim2 {
				fn(t.order[k], r2)
			}
		}
	}
}

// boxBoxDist2 is the squared distance between the boxes of a and b (0 if they overlap).
func boxBoxDist2(a, b *node) float64 {
	var d2 float64
	for _, d := range [3]float64{
		max(a.minX-b.maxX, b.minX-a.maxX, 0),
		max(a.minY-b.maxY, b.minY-a.maxY, 0),
		max(a.minZ-b.maxZ, b.minZ-a.maxZ, 0),
	} {
		d2 += d * d
	}
	return d2
}

// boxDist2 is the squared distance from (px, py, pz) to the box of nd (0 inside).
func boxDist2(nd *node, px, py, pz float64) float64 {
	var d2 float64
	if d := nd.minX - px; d > 0 {
		d2 += d * d
	} else if d := px - nd.maxX; d > 0 {
		d2 += d * d
	}
	if d := nd.minY - py; d > 0 {
		d2 += d * d
	} else if d := py - nd.maxY; d > 0 {
		d2 += d * d
	}
	if d := nd.minZ - pz; d > 0 {
		d2 += d * d
	} else if d := pz - nd.maxZ; d > 0 {
		d2 += d * d
	}
	return d2
}

func resize(s []float64, n int) []float64 {
	if cap(s) < n {
		return make([]float64, n)
	}
	return s[:n]
}
