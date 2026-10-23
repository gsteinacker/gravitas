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
// (parallel O(N²), ~70 ms at the limit). Tree estimates are biased by ~10⁻⁴…10⁻³ and only track
// the drift to within about ±30 % (measured, see T-007), so exact is preferred where affordable.
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
	k := KineticEnergy(m.State)
	if m.State.Len() <= exactEnergyLimit {
		return k + PotentialEnergyParallel(m.State, m.Params.Softening), EnergyExact
	}
	if ps, ok := m.Solver.(PotentialSolver); ok {
		return k + ps.PotentialEnergy(m.State, m.Params), EnergyEstimated
	}
	return 0, EnergyUnknown
}

// PotentialEnergyParallel is the exact softened potential energy, computed on all cores.
func PotentialEnergyParallel(s *State, softening float64) float64 {
	eps2 := softening * softening
	n := s.Len()
	var mu sync.Mutex
	var sum float64
	ParallelFor(n, 64, 0, func(lo, hi int) {
		var part float64
		for i := lo; i < hi; i++ {
			xi, yi, zi := s.PosX[i], s.PosY[i], s.PosZ[i]
			var psi float64
			for j := i + 1; j < n; j++ {
				dx, dy, dz := s.PosX[j]-xi, s.PosY[j]-yi, s.PosZ[j]-zi
				psi += s.Mass[j] / math.Sqrt(dx*dx+dy*dy+dz*dz+eps2)
			}
			part += s.Mass[i] * psi
		}
		mu.Lock()
		sum += part
		mu.Unlock()
	})
	return -sum
}

// KineticEnergy returns the total kinetic energy.
func KineticEnergy(s *State) float64 {
	var k float64
	for i := range s.Len() {
		v2 := s.VelX[i]*s.VelX[i] + s.VelY[i]*s.VelY[i] + s.VelZ[i]*s.VelZ[i]
		k += 0.5 * s.Mass[i] * v2
	}
	return k
}

// Energy returns the kinetic and (softened) potential energy. The potential is O(N²).
func Energy(s *State, softening float64) (kinetic, potential float64) {
	eps2 := softening * softening
	n := s.Len()
	kinetic = KineticEnergy(s)
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
