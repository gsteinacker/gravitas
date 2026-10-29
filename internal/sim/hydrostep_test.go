package sim

import (
	"math"
	"testing"

	"galaxy/internal/vec"
)

// fixedHydro is a hydro without forces that demands a given time step.
type fixedHydro struct{ dt float64 }

func (f fixedHydro) Forces(s *State, p *Params, dudt, divv []float64) {}
func (f fixedHydro) Timestep() float64                                { return f.dt }
func (f fixedHydro) AddAccelerations(*State)                          {}

// A step longer than the Courant step of the gas is split into equal parts that add up to Dt.
func TestStepSplitsForCourantCondition(t *testing.T) {
	s := NewState(2)
	s.Add(Body{Pos: vec.Vec3{X: 1}, Mass: 1, Kind: Gas})
	s.Add(Body{Pos: vec.Vec3{X: -1}, Mass: 1, Kind: Gas})
	m := New(s, nopSolver{}, Params{Dt: 0.01})
	m.Hydro = fixedHydro{dt: 0.003}
	m.Step() // the first step learns the Courant step
	m.Step()
	if m.HydroSubsteps != 4 {
		t.Errorf("%d substeps for dt 0.01 and a Courant step of 0.003, want 4", m.HydroSubsteps)
	}
	if math.Abs(m.Time-0.02) > 1e-12 || m.Params.Dt != 0.01 {
		t.Errorf("time %v, dt %v after two steps of 0.01", m.Time, m.Params.Dt)
	}
	m.Hydro = fixedHydro{dt: 1e-9}
	m.Step()
	m.Step()
	if m.HydroSubsteps != maxHydroSubsteps {
		t.Errorf("%d substeps, want the cap %d", m.HydroSubsteps, maxHydroSubsteps)
	}
}
