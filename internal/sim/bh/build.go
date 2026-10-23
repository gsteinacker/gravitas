package bh

import (
	"math"

	"galaxy/internal/sim"
)

// build sorts the stars by Morton key and rebuilds the tree, reusing all buffers.
func (b *Solver) build(s *sim.State) {
	b.order, b.holes = b.order[:0], b.holes[:0]
	minX, minY, minZ := math.Inf(1), math.Inf(1), math.Inf(1)
	maxX, maxY, maxZ := math.Inf(-1), math.Inf(-1), math.Inf(-1)
	for i := range s.Len() {
		if !s.Alive[i] {
			continue
		}
		if s.Kind[i] == sim.BlackHole {
			b.holes = append(b.holes, int32(i))
			continue
		}
		b.order = append(b.order, int32(i))
		x, y, z := s.PosX[i], s.PosY[i], s.PosZ[i]
		minX, maxX = math.Min(minX, x), math.Max(maxX, x)
		minY, maxY = math.Min(minY, y), math.Max(maxY, y)
		minZ, maxZ = math.Min(minZ, z), math.Max(maxZ, z)
	}
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
	for k, i := range b.order {
		b.keys[k] = mortonKey(cell(s.PosX[i], minX), cell(s.PosY[i], minY), cell(s.PosZ[i], minZ))
	}
	radixSort(b.keys, b.order, b.tmpK, b.tmpI)

	b.sx, b.sy, b.sz, b.sm = resize(b.sx, n), resize(b.sy, n), resize(b.sz, n), resize(b.sm, n)
	for k, i := range b.order {
		b.sx[k], b.sy[k], b.sz[k], b.sm[k] = s.PosX[i], s.PosY[i], s.PosZ[i], s.Mass[i]
	}

	half := side / 2
	b.nodes = append(b.nodes, node{})
	b.buildNode(0, 0, int32(n), 0, minX+half, minY+half, minZ+half, half, -1, b.theta())
}

// buildNode fills node ni for the sorted star range [lo, hi). The node is a cube with the given
// center and half side length at the given tree level.
func (b *Solver) buildNode(ni, lo, hi int32, level int, cx, cy, cz, half float64, next int32, theta float64) {
	var x, y, z, m float64
	first := int32(-1)
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
		children := int32(0)
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
	// Opening criterion with Barnes' (1994) offset correction: a node whose center of mass
	// lies off-center must be opened earlier. For θ ≤ 1 a node containing the target is always opened.
	delta := math.Sqrt((x-cx)*(x-cx) + (y-cy)*(y-cy) + (z-cz)*(z-cz))
	crit := 2*half/theta + delta
	b.nodes[ni] = node{x: x, y: y, z: z, m: m, crit2: crit * crit, first: first, next: next, lo: lo, hi: hi}
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
