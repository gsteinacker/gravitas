// Package sim holds the simulation state, the integrator and diagnostics.
// It must not depend on any rendering or UI package.
package sim

import "galaxy/internal/vec"

type Kind uint8

const (
	Star Kind = iota
	BlackHole
)

// Body is a convenience view of a single body, used for setup and tests.
type Body struct {
	Pos, Vel vec.Vec3
	Mass     float64
	Kind     Kind
}

// State stores all bodies as struct-of-arrays. Slices always have equal length.
type State struct {
	PosX, PosY, PosZ []float64
	VelX, VelY, VelZ []float64
	AccX, AccY, AccZ []float64
	Mass             []float64
	Kind             []Kind
	Alive            []bool
}

// NewState returns an empty state with room for n bodies.
func NewState(n int) *State {
	f := func() []float64 { return make([]float64, 0, n) }
	return &State{
		PosX: f(), PosY: f(), PosZ: f(),
		VelX: f(), VelY: f(), VelZ: f(),
		AccX: f(), AccY: f(), AccZ: f(),
		Mass: f(), Kind: make([]Kind, 0, n), Alive: make([]bool, 0, n),
	}
}

func (s *State) Len() int { return len(s.Mass) }

// Add appends a body and returns its index.
func (s *State) Add(b Body) int {
	s.PosX, s.PosY, s.PosZ = append(s.PosX, b.Pos.X), append(s.PosY, b.Pos.Y), append(s.PosZ, b.Pos.Z)
	s.VelX, s.VelY, s.VelZ = append(s.VelX, b.Vel.X), append(s.VelY, b.Vel.Y), append(s.VelZ, b.Vel.Z)
	s.AccX, s.AccY, s.AccZ = append(s.AccX, 0), append(s.AccY, 0), append(s.AccZ, 0)
	s.Mass = append(s.Mass, b.Mass)
	s.Kind = append(s.Kind, b.Kind)
	s.Alive = append(s.Alive, true)
	return s.Len() - 1
}

// Body returns a copy of body i.
func (s *State) Body(i int) Body {
	return Body{
		Pos:  vec.Vec3{X: s.PosX[i], Y: s.PosY[i], Z: s.PosZ[i]},
		Vel:  vec.Vec3{X: s.VelX[i], Y: s.VelY[i], Z: s.VelZ[i]},
		Mass: s.Mass[i],
		Kind: s.Kind[i],
	}
}

// Compact removes all bodies that are no longer alive, preserving order.
func (s *State) Compact() {
	j := 0
	for i := range s.Alive {
		if !s.Alive[i] {
			continue
		}
		s.PosX[j], s.PosY[j], s.PosZ[j] = s.PosX[i], s.PosY[i], s.PosZ[i]
		s.VelX[j], s.VelY[j], s.VelZ[j] = s.VelX[i], s.VelY[i], s.VelZ[i]
		s.AccX[j], s.AccY[j], s.AccZ[j] = s.AccX[i], s.AccY[i], s.AccZ[i]
		s.Mass[j], s.Kind[j], s.Alive[j] = s.Mass[i], s.Kind[i], true
		j++
	}
	s.PosX, s.PosY, s.PosZ = s.PosX[:j], s.PosY[:j], s.PosZ[:j]
	s.VelX, s.VelY, s.VelZ = s.VelX[:j], s.VelY[:j], s.VelZ[:j]
	s.AccX, s.AccY, s.AccZ = s.AccX[:j], s.AccY[:j], s.AccZ[:j]
	s.Mass, s.Kind, s.Alive = s.Mass[:j], s.Kind[:j], s.Alive[:j]
}
