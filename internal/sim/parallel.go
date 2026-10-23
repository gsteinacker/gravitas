package sim

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// chunksPerWorker > 1 lets fast cores pick up more work than slow ones (P- vs E-cores).
const chunksPerWorker = 8

// ParallelFor processes [0, n) in chunks of at least grain items on up to workers goroutines
// (0 = GOMAXPROCS). Chunks are handed out dynamically, so uneven per-item cost is balanced.
// Allocations are O(workers), independent of n.
func ParallelFor(n, grain, workers int, fn func(lo, hi int)) {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	if n <= grain || workers == 1 {
		fn(0, n)
		return
	}
	chunk := max(grain, n/(workers*chunksPerWorker))
	chunks := (n + chunk - 1) / chunk
	var next atomic.Int64
	var wg sync.WaitGroup
	for range min(workers, chunks) {
		wg.Go(func() {
			for {
				c := int(next.Add(1) - 1)
				if c >= chunks {
					return
				}
				lo := c * chunk
				fn(lo, min(lo+chunk, n))
			}
		})
	}
	wg.Wait()
}
