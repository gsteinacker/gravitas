package sim

import (
	"math"

	"galaxy/internal/vec"
)

// Energy returns the kinetic and (softened) potential energy. The potential is O(N²).
func Energy(s *State, softening float64) (kinetic, potential float64) {
	eps2 := softening * softening
	n := s.Len()
	for i := range n {
		v2 := s.VelX[i]*s.VelX[i] + s.VelY[i]*s.VelY[i] + s.VelZ[i]*s.VelZ[i]
		kinetic += 0.5 * s.Mass[i] * v2
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
