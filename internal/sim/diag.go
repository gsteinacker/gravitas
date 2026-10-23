package sim

import (
	"math"
	"sync"

	"galaxy/internal/vec"
)

// PotentialSolver is implemented by solvers that can estimate the potential energy much faster
// than the exact O(N²) sum.
type PotentialSolver interface {
	PotentialEnergy(s *State, p Params) float64
}

// exactEnergyLimit is the body count up to which the running app measures the potential exactly
// (parallel O(N²), ~70 ms at the limit). Above it the tree estimate is used; with quadrupole
// moments its error is ~10⁻⁶ (T-028), far below typical drifts.
const exactEnergyLimit = 30000

// EnergyQuality says how a total energy was obtained.
type EnergyQuality int

const (
	EnergyUnknown   EnergyQuality = iota // too expensive to measure
	EnergyEstimated                      // tree estimate (PotentialSolver)
	EnergyExact                          // exact softened O(N²) sum
)

// TotalEnergy returns K + U for periodic diagnostics.
func TotalEnergy(m *Simulation) (float64, EnergyQuality) {
	k := KineticEnergy(m.State, m.Params.C)
	if m.State.Len() <= exactEnergyLimit {
		return k + PotentialEnergyParallel(m.State, m.Params), EnergyExact
	}
	if ps, ok := m.Solver.(PotentialSolver); ok {
		return k + ps.PotentialEnergy(m.State, m.Params), EnergyEstimated
	}
	return 0, EnergyUnknown
}

// PotentialEnergyParallel is the exact potential energy of the pair model (pair.go), computed on
// all cores.
func PotentialEnergyParallel(s *State, p Params) float64 {
	eps2 := p.Softening * p.Softening
	n := s.Len()
	var mu sync.Mutex
	var sum float64
	ParallelFor(n, 64, 0, func(lo, hi int) {
		var part float64
		for i := lo; i < hi; i++ {
			xi, yi, zi := s.PosX[i], s.PosY[i], s.PosZ[i]
			for j := i + 1; j < n; j++ {
				dx, dy, dz := s.PosX[j]-xi, s.PosY[j]-yi, s.PosZ[j]-zi
				rs := PairRS(s.Kind[i], s.Mass[i], s.Kind[j], s.Mass[j], p.C)
				part += PairPotential(s.Mass[i], s.Mass[j], dx*dx+dy*dy+dz*dz, eps2, rs)
			}
		}
		mu.Lock()
		sum += part
		mu.Unlock()
	})
	return sum
}

// KineticEnergy returns the total kinetic energy, relativistic for c > 0 (see kinetic).
func KineticEnergy(s *State, c float64) float64 {
	var k float64
	for i := range s.Len() {
		k += kinetic(s.Mass[i], s.VelX[i], s.VelY[i], s.VelZ[i], c)
	}
	return k
}

// Energy returns the Newtonian kinetic and softened Newtonian potential energy, ignoring the
// Paczyński-Wiita pair model; for the model's energy use TotalEnergy. The potential is O(N²).
func Energy(s *State, softening float64) (kinetic, potential float64) {
	eps2 := softening * softening
	n := s.Len()
	kinetic = KineticEnergy(s, 0)
	for i := range n {
		for j := i + 1; j < n; j++ {
			dx, dy, dz := s.PosX[j]-s.PosX[i], s.PosY[j]-s.PosY[i], s.PosZ[j]-s.PosZ[i]
			potential -= s.Mass[i] * s.Mass[j] / math.Sqrt(dx*dx+dy*dy+dz*dz+eps2)
		}
	}
	return kinetic, potential
}

// Momentum returns the total linear momentum.
func Momentum(s *State) vec.Vec3 {
	var p vec.Vec3
	for i := range s.Len() {
		b := s.Body(i)
		p = p.Add(b.Vel.Scale(b.Mass))
	}
	return p
}

// AngularMomentum returns the total angular momentum around the origin.
func AngularMomentum(s *State) vec.Vec3 {
	var l vec.Vec3
	for i := range s.Len() {
		b := s.Body(i)
		l = l.Add(b.Pos.Cross(b.Vel.Scale(b.Mass)))
	}
	return l
}

// CenterOfMass returns the center of mass and the total mass.
func CenterOfMass(s *State) (vec.Vec3, float64) {
	var c vec.Vec3
	var m float64
	for i := range s.Len() {
		b := s.Body(i)
		c = c.Add(b.Pos.Scale(b.Mass))
		m += b.Mass
	}
	if m == 0 {
		return vec.Vec3{}, 0
	}
	return c.Scale(1 / m), m
}
