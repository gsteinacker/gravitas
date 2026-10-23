package bh

// mortonBits is the resolution per axis; 3 × 21 = 63 bits fit into a uint64 key.
const mortonBits = 21

// spread inserts two zero bits between each of the lower 21 bits of v.
func spread(v uint64) uint64 {
	v &= 1<<mortonBits - 1
	v = (v | v<<32) & 0x1f00000000ffff
	v = (v | v<<16) & 0x1f0000ff0000ff
	v = (v | v<<8) & 0x100f00f00f00f00f
	v = (v | v<<4) & 0x10c30c30c30c30c3
	v = (v | v<<2) & 0x1249249249249249
	return v
}

// mortonKey interleaves the bits as ...x1y1z1x0y0z0, so each 3-bit digit is an octant
// (bit 2 = upper x half, bit 1 = upper y half, bit 0 = upper z half).
func mortonKey(x, y, z uint64) uint64 {
	return spread(x)<<2 | spread(y)<<1 | spread(z)
}

// radixSort sorts keys ascending and applies the same permutation to idx.
// tmpK and tmpI must have the same length as keys.
func radixSort(keys []uint64, idx []int32, tmpK []uint64, tmpI []int32) {
	var count [256]int
	for shift := uint(0); shift < 64; shift += 8 {
		count = [256]int{}
		for _, k := range keys {
			count[(k>>shift)&0xff]++
		}
		if count[(keys[0]>>shift)&0xff] == len(keys) {
			continue // all keys share this digit
		}
		sum := 0
		for i, c := range count {
			count[i] = sum
			sum += c
		}
		for i, k := range keys {
			d := (k >> shift) & 0xff
			tmpK[count[d]] = k
			tmpI[count[d]] = idx[i]
			count[d]++
		}
		copy(keys, tmpK)
		copy(idx, tmpI)
	}
}
