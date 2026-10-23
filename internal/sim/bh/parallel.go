package bh

import (
	"runtime"
	"sync"
)

// parallelMin is the item count below which the build phases run serially.
const parallelMin = 1 << 15

// staticChunks splits [0, n) into one fixed chunk per worker and runs fn(w, lo, hi) for each
// concurrently. Unlike sim.ParallelFor the mapping chunk → worker index is fixed, which the
// stable parallel radix sort needs.
func staticChunks(n, workers int, fn func(w, lo, hi int)) int {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if n < parallelMin {
		workers = 1
	}
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := range workers {
		lo, hi := min(w*chunk, n), min((w+1)*chunk, n)
		wg.Go(func() { fn(w, lo, hi) })
	}
	wg.Wait()
	return workers
}

// radixSortParallel is a stable LSD radix sort of keys (applying the same permutation to idx)
// with per-worker histograms. hist is a reusable buffer.
func radixSortParallel(keys []uint64, idx []int32, tmpK []uint64, tmpI []int32, workers int, hist *[][256]int) {
	n := len(keys)
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if n < parallelMin || workers == 1 {
		radixSort(keys, idx, tmpK, tmpI)
		return
	}
	if len(*hist) < workers {
		*hist = make([][256]int, workers)
	}
	h := (*hist)[:workers]
	src, dst, srcI, dstI := keys, tmpK, idx, tmpI
	for shift := uint(0); shift < 64; shift += 8 {
		staticChunks(n, workers, func(w, lo, hi int) {
			h[w] = [256]int{}
			for _, k := range src[lo:hi] {
				h[w][(k>>shift)&0xff]++
			}
		})
		var total [256]int
		for w := range h {
			for d, c := range h[w] {
				total[d] += c
			}
		}
		if total[(src[0]>>shift)&0xff] == n {
			continue // all keys share this digit
		}
		// Offsets: digit-major, then worker order, so the scatter is stable.
		sum := 0
		for d := range 256 {
			for w := range h {
				c := h[w][d]
				h[w][d] = sum
				sum += c
			}
		}
		staticChunks(n, workers, func(w, lo, hi int) {
			off := &h[w]
			for i := lo; i < hi; i++ {
				k := src[i]
				d := (k >> shift) & 0xff
				dst[off[d]] = k
				dstI[off[d]] = srcI[i]
				off[d]++
			}
		})
		src, dst, srcI, dstI = dst, src, dstI, srcI
	}
	if &src[0] != &keys[0] {
		copy(keys, src)
		copy(idx, srcI)
	}
}
