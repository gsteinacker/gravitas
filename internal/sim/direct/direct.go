// Package direct implements the O(N²) reference solver, parallelized over all cores.
package direct

import (
	"math"

	"galaxy/internal/sim"
)

// grain is the minimum number of bodies per parallel chunk.
const grain = 64

// Solver sums all pair forces exactly (up to softening).
type Solver struct {
	Workers int // 0 means GOMAXPROCS
}

func (d *Solver) Accelerations(s *sim.State, p sim.Params) {
	eps2 := p.Softening * p.Softening
	sim.ParallelFor(s.Len(), grain, d.Workers, func(lo, hi int) { accRange(s, p.C, eps2, lo, hi) })
}

// accRange sums all pair forces on bodies [lo, hi); star–hole pairs use sim.PairForce.
func accRange(s *sim.State, c, eps2 float64, lo, hi int) {
	n := s.Len()
	px, py, pz, m, kind := s.PosX[:n], s.PosY[:n], s.PosZ[:n], s.Mass[:n], s.Kind[:n]
	for i := lo; i < hi; i++ {
		xi, yi, zi, ki := px[i], py[i], pz[i], kind[i]
		var ax, ay, az float64
		for j := range px {
			if j == i {
				continue
			}
			dx, dy, dz := px[j]-xi, py[j]-yi, pz[j]-zi
			r2 := dx*dx + dy*dy + dz*dz
			var f float64
			if kind[j] == ki || c <= 0 {
				re2 := r2 + eps2
				f = m[j] / (re2 * math.Sqrt(re2))
			} else {
				f = sim.PairForce(m[j], r2, eps2, sim.PairRS(ki, m[i], kind[j], m[j], c))
			}
			ax += f * dx
			ay += f * dy
			az += f * dz
		}
		s.AccX[i], s.AccY[i], s.AccZ[i] = ax, ay, az
	}
}
