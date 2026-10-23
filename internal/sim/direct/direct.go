// Package direct implements the O(N²) reference solver, parallelized over all cores.
package direct

import (
	"math"
	"runtime"
	"sync"

	"galaxy/internal/sim"
)

// serialBelow is the body count under which goroutine overhead outweighs parallelism.
const serialBelow = 256

// chunksPerWorker > 1 balances load between performance and efficiency cores.
const chunksPerWorker = 4

// Solver sums all pair forces exactly (up to softening).
type Solver struct {
	Workers int // 0 means GOMAXPROCS
}

func (d *Solver) Accelerations(s *sim.State, p sim.Params) {
	n := s.Len()
	eps2 := p.Softening * p.Softening
	if n < serialBelow {
		accRange(s, eps2, 0, n)
		return
	}
	workers := d.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	chunk := max(1, (n+workers*chunksPerWorker-1)/(workers*chunksPerWorker))
	var wg sync.WaitGroup
	for lo := 0; lo < n; lo += chunk {
		hi := min(lo+chunk, n)
		wg.Go(func() { accRange(s, eps2, lo, hi) })
	}
	wg.Wait()
}

func accRange(s *sim.State, eps2 float64, lo, hi int) {
	n := s.Len()
	px, py, pz, m := s.PosX[:n], s.PosY[:n], s.PosZ[:n], s.Mass[:n]
	for i := lo; i < hi; i++ {
		xi, yi, zi := px[i], py[i], pz[i]
		var ax, ay, az float64
		for j := range px {
			if j == i {
				continue
			}
			dx, dy, dz := px[j]-xi, py[j]-yi, pz[j]-zi
			r2 := dx*dx + dy*dy + dz*dz + eps2
			f := m[j] / (r2 * math.Sqrt(r2))
			ax += f * dx
			ay += f * dy
			az += f * dz
		}
		s.AccX[i], s.AccY[i], s.AccZ[i] = ax, ay, az
	}
}
