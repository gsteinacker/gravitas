// Package sim holds the simulation state, the integrator and diagnostics.
// It must not depend on any rendering or UI package.
package sim

import (
	"math"

	"galaxy/internal/vec"
)

type Kind uint8

const (
	Star Kind = iota
	BlackHole
	// DarkMatter behaves like a star in every respect (gravity, accretion) but is not drawn.
	DarkMatter
	// Gas feels gravity like a star plus pressure and viscosity (internal/sph).
	Gas
)

// Body is a convenience view of a single body, used for setup and tests.
type Body struct {
	Pos, Vel vec.Vec3
	Mass     float64
	Kind     Kind
	U        float64  // specific internal energy (gas)
	Lum      float64  // ionizing luminosity (relative); > 0 makes the body a source
	Spin     vec.Vec3 // angular momentum of a black hole (kerr.go), zero for other bodies
}

// State stores all bodies as struct-of-arrays. Slices always have equal length.
type State struct {
	PosX, PosY, PosZ []float64
	VelX, VelY, VelZ []float64
	AccX, AccY, AccZ []float64
	Mass             []float64
	Kind             []Kind
	Alive            []bool
	ID               []uint64 // stable identity across Compact; assigned by Add
	// Gas (unused for other kinds): specific internal energy, smoothing length, density.
	U, H, Rho []float64
	// Born is the simulation time a star formed from gas; -Inf for stars of the initial conditions.
	Born []float64
	// Lum > 0 marks a source of ionizing radiation (relative strength); Ion is the ionized
	// fraction of gas (ionize.go).
	Lum, Ion []float64
	// SpinX/Y/Z is the angular momentum J of a black hole (kerr.go), zero for other bodies.
	SpinX, SpinY, SpinZ []float64

	nextID uint64
}

// NewState returns an empty state with room for n bodies.
func NewState(n int) *State {
	f := func() []float64 { return make([]float64, 0, n) }
	return &State{
		PosX: f(), PosY: f(), PosZ: f(),
		VelX: f(), VelY: f(), VelZ: f(),
		AccX: f(), AccY: f(), AccZ: f(),
		Mass: f(), Kind: make([]Kind, 0, n), Alive: make([]bool, 0, n), ID: make([]uint64, 0, n),
		U: f(), H: f(), Rho: f(), Born: f(), Lum: f(), Ion: f(),
		SpinX: f(), SpinY: f(), SpinZ: f(),
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
	s.nextID++
	s.ID = append(s.ID, s.nextID)
	s.U, s.H, s.Rho = append(s.U, b.U), append(s.H, 0), append(s.Rho, 0)
	s.Born = append(s.Born, math.Inf(-1))
	s.Lum, s.Ion = append(s.Lum, b.Lum), append(s.Ion, 0)
	s.SpinX, s.SpinY, s.SpinZ = append(s.SpinX, b.Spin.X), append(s.SpinY, b.Spin.Y), append(s.SpinZ, b.Spin.Z)
	return s.Len() - 1
}

// Body returns a copy of body i.
func (s *State) Body(i int) Body {
	return Body{
		Pos:  vec.Vec3{X: s.PosX[i], Y: s.PosY[i], Z: s.PosZ[i]},
		Vel:  vec.Vec3{X: s.VelX[i], Y: s.VelY[i], Z: s.VelZ[i]},
		Mass: s.Mass[i],
		Kind: s.Kind[i],
		U:    s.U[i],
		Lum:  s.Lum[i],
		Spin: s.Spin(i),
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
		s.Mass[j], s.Kind[j], s.Alive[j], s.ID[j] = s.Mass[i], s.Kind[i], true, s.ID[i]
		s.U[j], s.H[j], s.Rho[j], s.Born[j] = s.U[i], s.H[i], s.Rho[i], s.Born[i]
		s.Lum[j], s.Ion[j] = s.Lum[i], s.Ion[i]
		s.SpinX[j], s.SpinY[j], s.SpinZ[j] = s.SpinX[i], s.SpinY[i], s.SpinZ[i]
		j++
	}
	s.PosX, s.PosY, s.PosZ = s.PosX[:j], s.PosY[:j], s.PosZ[:j]
	s.VelX, s.VelY, s.VelZ = s.VelX[:j], s.VelY[:j], s.VelZ[:j]
	s.AccX, s.AccY, s.AccZ = s.AccX[:j], s.AccY[:j], s.AccZ[:j]
	s.Mass, s.Kind, s.Alive, s.ID = s.Mass[:j], s.Kind[:j], s.Alive[:j], s.ID[:j]
	s.U, s.H, s.Rho, s.Born = s.U[:j], s.H[:j], s.Rho[:j], s.Born[:j]
	s.Lum, s.Ion = s.Lum[:j], s.Ion[:j]
	s.SpinX, s.SpinY, s.SpinZ = s.SpinX[:j], s.SpinY[:j], s.SpinZ[:j]
}
