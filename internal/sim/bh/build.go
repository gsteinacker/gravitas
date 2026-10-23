package bh

import (
	"math"
	"runtime"

	"galaxy/internal/sim"
)

// build sorts the stars by Morton key and rebuilds the tree, reusing all buffers.
func (b *Solver) build(s *sim.State, theta float64) {
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
			b.sx[k], b.sy[k], b.sz[k], b.sm[k] = s.PosX[i], s.PosY[i], s.PosZ[i], s.Mass[i]
		}
	})

	half := side / 2
	b.nodes = append(b.nodes, node{})
	b.buildNode(0, 0, int32(n), 0, minX+half, minY+half, minZ+half, half, -1, theta)
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
	isStar := func(i int) bool { return s.Alive[i] && s.Kind[i] == sim.Star }
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

// buildNode fills node ni for the sorted star range [lo, hi). The node is a cube with the given
// center and half side length at the given tree level.
func (b *Solver) buildNode(ni, lo, hi int32, level int, cx, cy, cz, half float64, next int32, theta float64) {
	var x, y, z, m float64
	first, children := int32(-1), int32(0)
	if hi-lo <= leafSize || level >= mortonBits {
		for j := lo; j < hi; j++ {
			mj := b.sm[j]
			x, y, z, m = x+mj*b.sx[j], y+mj*b.sy[j], z+mj*b.sz[j], m+mj
		}
	} else {
		shift := uint(3 * (mortonBits - 1 - level))
		var bounds [9]int32
		bounds[0], bounds[8] = lo, hi
		for oct := uint64(1); oct < 8; oct++ {
			bounds[oct] = b.lowerBound(bounds[oct-1], hi, shift, oct)
		}
		for oct := range 8 {
			if bounds[oct+1] > bounds[oct] {
				children++
			}
		}
		first = int32(len(b.nodes))
		for range children {
			b.nodes = append(b.nodes, node{})
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
			b.buildNode(c, clo, chi, level+1,
				cx+offset(oct, 2)*q, cy+offset(oct, 1)*q, cz+offset(oct, 0)*q, q, childNext, theta)
			ch := &b.nodes[c]
			x, y, z, m = x+ch.m*ch.x, y+ch.m*ch.y, z+ch.m*ch.z, m+ch.m
			c++
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
			ch := &b.nodes[c]
			q.qxx, q.qyy, q.qzz = q.qxx+ch.qxx, q.qyy+ch.qyy, q.qzz+ch.qzz
			q.qxy, q.qxz, q.qyz = q.qxy+ch.qxy, q.qxz+ch.qxz, q.qyz+ch.qyz
			q.add(ch.m, ch.x-x, ch.y-y, ch.z-z)
		}
	}
	// Opening criterion with Barnes' (1994) offset correction: a node whose center of mass
	// lies off-center must be opened earlier. For θ ≤ 1 a node containing the target is always opened.
	delta := math.Sqrt((x-cx)*(x-cx) + (y-cy)*(y-cy) + (z-cz)*(z-cz))
	crit := 2*half/theta + delta
	b.nodes[ni] = node{
		x: x, y: y, z: z, m: m,
		qxx: q.qxx, qyy: q.qyy, qzz: q.qzz, qxy: q.qxy, qxz: q.qxz, qyz: q.qyz,
		crit2: crit * crit, first: first, next: next, lo: lo, hi: hi,
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
