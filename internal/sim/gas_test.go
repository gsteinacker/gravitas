package sim

import (
	"math"
	"testing"

	"galaxy/internal/vec"
)

// densityHydro sets a given density and a converging flow for every gas body, without forces.
type densityHydro struct{ rho float64 }

func (h densityHydro) Forces(s *State, _ *Params, _, divv []float64) {
	for i := range s.Len() {
		if s.Kind[i] == Gas {
			s.Rho[i], divv[i] = h.rho, -1
		}
	}
}

func (densityHydro) Timestep() float64       { return math.Inf(1) }
func (densityHydro) AddAccelerations(*State) {}

func TestUnresolvedGasFormsStarsAtOnce(t *testing.T) {
	p := Params{Dt: 1e-3, Softening: 0.1, Gas: GasParams{UFloor: 0.01, SFDensity: 1e-6, SFEfficiency: 1e-9}}
	run := func(rhoFactor float64) int {
		s := NewState(10)
		for k := range 10 {
			s.Add(Body{Pos: vec.Vec3{X: float64(k)}, Mass: 0.01, Kind: Gas, U: 1}) // hot: 100 u_floor
		}
		m := New(s, pairSolver{}, p)
		m.Hydro = densityHydro{rho: rhoFactor * m.unresolvedDensity() * 0.01}
		m.Step()
		m.Step()
		return m.StarsFormed
	}
	if n := run(1.1); n != 10 {
		t.Errorf("%d of 10 unresolved hot bodies became stars, want all", n)
	}
	if n := run(0.9); n != 0 {
		t.Errorf("%d resolved hot bodies became stars, want none", n)
	}
}
