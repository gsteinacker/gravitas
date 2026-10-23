package bh

import (
	"math"
	"runtime"

	"galaxy/internal/sim"
)

// build sorts the stars by Morton key and rebuilds the tree, reusing all buffers.
func (b *Solver) build(s *sim.State, p *sim.Params, theta float64) {
	minX, minY, minZ, maxX, maxY, maxZ := b.collect(s)
	b.nodes = b.nodes[:0]
	n := len(b.order)
	if n == 0 {
		return
	}

	side := math.Max(maxX-minX, math.Max(maxY-minY, maxZ-minZ))
	side = math.Max(side*(1+1e-9), 1e-12)
	scale := float64(1<<mortonBits) / side
	b.keys, b.tmpK, b.tmpI = resize(b.keys, n), resize(b.tmpK, n), resize(b.tmpI, n)
	cell := func(v, lo float64) uint64 {
		return uint64(min(float64(1<<mortonBits-1), math.Max(0, (v-lo)*scale)))
	}
	staticChunks(n, b.Workers, func(_, lo, hi int) {
		for k := lo; k < hi; k++ {
			i := b.order[k]
			b.keys[k] = mortonKey(cell(s.PosX[i], minX), cell(s.PosY[i], minY), cell(s.PosZ[i], minZ))
		}
	})
	radixSortParallel(b.keys, b.order, b.tmpK, b.tmpI, b.Workers, &b.hist)

	b.sx, b.sy, b.sz, b.sm = resize(b.sx, n), resize(b.sy, n), resize(b.sz, n), resize(b.sm, n)
	staticChunks(n, b.Workers, func(_, lo, hi int) {
		for k := lo; k < hi; k++ {
			i := b.order[k]
			// Source masses: switched-off dark matter is in the tree as a target but pulls nothing.
			b.sx[k], b.sy[k], b.sz[k], b.sm[k] = s.PosX[i], s.PosY[i], s.PosZ[i], p.SourceMass(s.Kind[i], s.Mass[i])
		}
	})

	half := side / 2
	if n >= parallelMin && b.workers() > 1 {
		b.buildParallel(int32(n), minX+half, minY+half, minZ+half, half, theta)
		return
	}
	t := treeBuilder{b: b, nodes: append(b.nodes[:0], node{}), theta: theta}
	t.buildNode(0, 0, int32(n), 0, minX+half, minY+half, minZ+half, half, -1)
	b.nodes = t.nodes
}

// part is one chunk's result in collect.
type part struct {
	count                              int
	minX, minY, minZ, maxX, maxY, maxZ float64
}

// collect fills b.order (alive stars in state order) and b.holes, and returns the stars'
// bounding box. Counting and bounds run in parallel chunks; the order is assembled from per-chunk
// offsets, so it stays in state order.
func (b *Solver) collect(s *sim.State) (minX, minY, minZ, maxX, maxY, maxZ float64) {
	n := s.Len()
	b.holes = b.holes[:0]
	for i := range n {
		if s.Alive[i] && s.Kind[i] == sim.BlackHole {
			b.holes = append(b.holes, int32(i))
		}
	}
	parts := b.parts[:0]
	for range max(1, b.workers()) {
		parts = append(parts, part{})
	}
	isStar := func(i int) bool { return s.Alive[i] && s.Kind[i] != sim.BlackHole } // stars and dark matter
	workers := staticChunks(n, len(parts), func(w, lo, hi int) {
		p := part{minX: math.Inf(1), minY: math.Inf(1), minZ: math.Inf(1), maxX: math.Inf(-1), maxY: math.Inf(-1), maxZ: math.Inf(-1)}
		for i := lo; i < hi; i++ {
			if !isStar(i) {
				continue
			}
			p.count++
			x, y, z := s.PosX[i], s.PosY[i], s.PosZ[i]
			p.minX, p.maxX = math.Min(p.minX, x), math.Max(p.maxX, x)
			p.minY, p.maxY = math.Min(p.minY, y), math.Max(p.maxY, y)
			p.minZ, p.maxZ = math.Min(p.minZ, z), math.Max(p.maxZ, z)
		}
		parts[w] = p
	})
	parts = parts[:workers]
	minX, minY, minZ = math.Inf(1), math.Inf(1), math.Inf(1)
	maxX, maxY, maxZ = math.Inf(-1), math.Inf(-1), math.Inf(-1)
	total := 0
	offsets := b.offsets[:0]
	for _, p := range parts {
		offsets = append(offsets, total)
		total += p.count
		minX, maxX = math.Min(minX, p.minX), math.Max(maxX, p.maxX)
		minY, maxY = math.Min(minY, p.minY), math.Max(maxY, p.maxY)
		minZ, maxZ = math.Min(minZ, p.minZ), math.Max(maxZ, p.maxZ)
	}
	b.order = resize(b.order, total)
	staticChunks(n, workers, func(w, lo, hi int) {
		k := offsets[w]
		for i := lo; i < hi; i++ {
			if isStar(i) {
				b.order[k] = int32(i)
				k++
			}
		}
	})
	b.parts, b.offsets = parts, offsets
	return
}

func (b *Solver) workers() int {
	if b.Workers > 0 {
		return b.Workers
	}
	return runtime.GOMAXPROCS(0)
}

// subtreeNext marks, inside a subtree built in isolation, the link to whatever follows the subtree.
const subtreeNext = -2

// treeBuilder builds (part of) the tree into nodes. The sorted star arrays are shared.
type treeBuilder struct {
	b     *Solver
	nodes []node
	theta float64
}

// buildNode fills node ni for the sorted star range [lo, hi). The node is a cube with the given
// center and half side length at the given tree level.
func (t *treeBuilder) buildNode(ni, lo, hi int32, level int, cx, cy, cz, half float64, next int32) {
	first, children := int32(-1), int32(0)
	if hi-lo > leafSize && level < mortonBits {
		bounds, n := t.b.split(lo, hi, level)
		first, children = int32(len(t.nodes)), n
		for range children {
			t.nodes = append(t.nodes, node{})
		}
		c, q := first, half/2
		for oct := range 8 {
			clo, chi := bounds[oct], bounds[oct+1]
			if chi == clo {
				continue
			}
			childNext := next
			if c < first+children-1 {
				childNext = c + 1
			}
			t.buildNode(c, clo, chi, level+1, cx+offset(oct, 2)*q, cy+offset(oct, 1)*q, cz+offset(oct, 0)*q, q, childNext)
			c++
		}
	}
	t.nodes[ni] = t.b.finish(t.nodes, lo, hi, first, children, cx, cy, cz, half, next, t.theta)
}

// split returns the octant boundaries of [lo, hi) at level and the number of non-empty octants.
func (b *Solver) split(lo, hi int32, level int) (bounds [9]int32, children int32) {
	shift := uint(3 * (mortonBits - 1 - level))
	bounds[0], bounds[8] = lo, hi
	for oct := uint64(1); oct < 8; oct++ {
		bounds[oct] = b.lowerBound(bounds[oct-1], hi, shift, oct)
	}
	for oct := range 8 {
		if bounds[oct+1] > bounds[oct] {
			children++
		}
	}
	return bounds, children
}

// finish computes a node's moments, from its stars (first < 0) or from its children
// nodes[first : first+children].
func (b *Solver) finish(nodes []node, lo, hi, first, children int32, cx, cy, cz, half float64, next int32, theta float64) node {
	var x, y, z, m float64
	if first < 0 {
		for j := lo; j < hi; j++ {
			mj := b.sm[j]
			x, y, z, m = x+mj*b.sx[j], y+mj*b.sy[j], z+mj*b.sz[j], m+mj
		}
	} else {
		for c := first; c < first+children; c++ {
			ch := &nodes[c]
			x, y, z, m = x+ch.m*ch.x, y+ch.m*ch.y, z+ch.m*ch.z, m+ch.m
		}
	}
	if m > 0 {
		x, y, z = x/m, y/m, z/m
	} else {
		x, y, z = cx, cy, cz
	}
	var q quadrupole
	if first < 0 {
		for j := lo; j < hi; j++ {
			q.add(b.sm[j], b.sx[j]-x, b.sy[j]-y, b.sz[j]-z)
		}
	} else {
		// Parallel axis theorem: shift each child's tensor to this node's center of mass.
		for c := first; c < first+children; c++ {
			ch := &nodes[c]
			q.qxx, q.qyy, q.qzz = q.qxx+ch.qxx, q.qyy+ch.qyy, q.qzz+ch.qzz
			q.qxy, q.qxz, q.qyz = q.qxy+ch.qxy, q.qxz+ch.qxz, q.qyz+ch.qyz
			q.add(ch.m, ch.x-x, ch.y-y, ch.z-z)
		}
	}
	// Opening criterion with Barnes' (1994) offset correction: a node whose center of mass
	// lies off-center must be opened earlier. For θ ≤ 1 a node containing the target is always opened.
	delta := math.Sqrt((x-cx)*(x-cx) + (y-cy)*(y-cy) + (z-cz)*(z-cz))
	crit := 2*half/theta + delta
	return node{
		x: x, y: y, z: z, m: m,
		qxx: q.qxx, qyy: q.qyy, qzz: q.qzz, qxy: q.qxy, qxz: q.qxz, qyz: q.qyz,
		crit2: crit * crit, first: first, next: next, lo: lo, hi: hi,
	}
}

// Parallel tree build (T-029): the top of the tree is split serially into subtree tasks, the
// subtrees are built concurrently into their own node slices, merged with shifted indices, and the
// top nodes are finished bottom-up. The walk order is defined by the links only, so forces are
// bit-identical to the serial build.

type subtreeTask struct {
	ni, lo, hi       int32
	level            int
	cx, cy, cz, half float64
	next             int32
}

type topNode struct {
	ni, lo, hi, first, children int32
	cx, cy, cz, half            float64
	next                        int32
}

// buildParallel builds the tree for n sorted stars into b.nodes.
func (b *Solver) buildParallel(n int32, cx, cy, cz, half, theta float64) {
	taskSize := max(8*leafSize, n/int32(8*b.workers()))
	b.tasks, b.tops = b.tasks[:0], b.tops[:0]
	b.nodes = append(b.nodes[:0], node{})
	b.splitTop(0, 0, n, 0, cx, cy, cz, half, -1, taskSize)

	for len(b.locals) < len(b.tasks) {
		b.locals = append(b.locals, nil)
	}
	sim.ParallelFor(len(b.tasks), 1, b.Workers, func(lo, hi int) {
		for ti := lo; ti < hi; ti++ {
			task := &b.tasks[ti]
			t := treeBuilder{b: b, nodes: append(b.locals[ti][:0], node{}), theta: theta}
			t.buildNode(0, task.lo, task.hi, task.level, task.cx, task.cy, task.cz, task.half, subtreeNext)
			b.locals[ti] = t.nodes
		}
	})

	// Prefix sums give every subtree its place, so merging can run in parallel as well.
	b.bases = b.bases[:0]
	total := int32(len(b.nodes))
	for ti := range b.tasks {
		b.bases = append(b.bases, total)
		total += int32(len(b.locals[ti])) - 1 // the local root goes to the task's placeholder
	}
	b.nodes = resize(b.nodes, int(total))
	sim.ParallelFor(len(b.tasks), 1, b.Workers, func(lo, hi int) {
		for ti := lo; ti < hi; ti++ {
			task, base, local := &b.tasks[ti], b.bases[ti], b.locals[ti]
			remap := func(j int32) int32 {
				switch {
				case j == subtreeNext:
					return task.next
				case j < 0:
					return j
				case j == 0:
					return task.ni
				default:
					return base + j - 1
				}
			}
			for j := range local {
				nd := local[j]
				nd.first, nd.next = remap(nd.first), remap(nd.next)
				if j == 0 {
					b.nodes[task.ni] = nd
				} else {
					b.nodes[base+int32(j)-1] = nd
				}
			}
		}
	})

	// Top nodes were created parents first, so finishing them in reverse sees finished children.
	for i := len(b.tops) - 1; i >= 0; i-- {
		tn := &b.tops[i]
		b.nodes[tn.ni] = b.finish(b.nodes, tn.lo, tn.hi, tn.first, tn.children, tn.cx, tn.cy, tn.cz, tn.half, tn.next, theta)
	}
}

// splitTop creates the top of the tree down to subtrees of at most taskSize stars.
func (b *Solver) splitTop(ni, lo, hi int32, level int, cx, cy, cz, half float64, next, taskSize int32) {
	if hi-lo <= taskSize || hi-lo <= leafSize || level >= mortonBits {
		b.tasks = append(b.tasks, subtreeTask{ni: ni, lo: lo, hi: hi, level: level, cx: cx, cy: cy, cz: cz, half: half, next: next})
		return
	}
	bounds, children := b.split(lo, hi, level)
	first := int32(len(b.nodes))
	for range children {
		b.nodes = append(b.nodes, node{})
	}
	b.tops = append(b.tops, topNode{ni: ni, lo: lo, hi: hi, first: first, children: children, cx: cx, cy: cy, cz: cz, half: half, next: next})
	c, q := first, half/2
	for oct := range 8 {
		clo, chi := bounds[oct], bounds[oct+1]
		if chi == clo {
			continue
		}
		childNext := next
		if c < first+children-1 {
			childNext = c + 1
		}
		b.splitTop(c, clo, chi, level+1, cx+offset(oct, 2)*q, cy+offset(oct, 1)*q, cz+offset(oct, 0)*q, q, childNext, taskSize)
		c++
	}
}

// quadrupole accumulates a traceless quadrupole tensor.
type quadrupole struct{ qxx, qyy, qzz, qxy, qxz, qyz float64 }

// add adds point mass m at offset (dx, dy, dz) from the center of mass.
func (q *quadrupole) add(m, dx, dy, dz float64) {
	r2 := dx*dx + dy*dy + dz*dz
	q.qxx += m * (3*dx*dx - r2)
	q.qyy += m * (3*dy*dy - r2)
	q.qzz += m * (3*dz*dz - r2)
	q.qxy += m * 3 * dx * dy
	q.qxz += m * 3 * dx * dz
	q.qyz += m * 3 * dy * dz
}

// lowerBound returns the first index in [lo, hi) whose octant digit at shift is >= oct.
func (b *Solver) lowerBound(lo, hi int32, shift uint, oct uint64) int32 {
	for lo < hi {
		mid := int32(uint32(lo+hi) >> 1)
		if (b.keys[mid]>>shift)&7 < oct {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// offset returns -1 or +1 depending on the given bit of the octant digit.
func offset(oct, bit int) float64 {
	if oct>>bit&1 == 1 {
		return 1
	}
	return -1
}

func resize[T any](s []T, n int) []T {
	if cap(s) < n {
		return make([]T, n)
	}
	return s[:n]
}
