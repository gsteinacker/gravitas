// Package direct implements the O(N²) reference solver, parallelized over all cores.
package direct

import (
	"math"

	"galaxy/internal/sim"
)

// grain is the minimum number of bodies per parallel chunk.
const grain = 64

// Solver sums all star–star pair forces exactly (up to softening).
type Solver struct {
	Workers int // 0 means GOMAXPROCS
}

func (d *Solver) Accelerations(s *sim.State, p sim.Params) {
	eps2 := p.Softening * p.Softening
	sim.ParallelFor(s.Len(), grain, d.Workers, func(lo, hi int) { accRange(s, eps2, lo, hi) })
}

// accRange sums all star–star forces on bodies [lo, hi). Pairs with a hole are left to the
// integrator (sim.Solver), so holes get zero.
func accRange(s *sim.State, eps2 float64, lo, hi int) {
	n := s.Len()
	px, py, pz, m, kind := s.PosX[:n], s.PosY[:n], s.PosZ[:n], s.Mass[:n], s.Kind[:n]
	for i := lo; i < hi; i++ {
		xi, yi, zi := px[i], py[i], pz[i]
		var ax, ay, az float64
		if kind[i] == sim.BlackHole {
			s.AccX[i], s.AccY[i], s.AccZ[i] = 0, 0, 0
			continue
		}
		for j := range px {
			if j == i || kind[j] == sim.BlackHole {
				continue
			}
			dx, dy, dz := px[j]-xi, py[j]-yi, pz[j]-zi
			re2 := dx*dx + dy*dy + dz*dz + eps2
			f := m[j] / (re2 * math.Sqrt(re2))
			ax += f * dx
			ay += f * dy
			az += f * dz
		}
		s.AccX[i], s.AccY[i], s.AccZ[i] = ax, ay, az
	}
}
