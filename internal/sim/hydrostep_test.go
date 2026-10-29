package sim

import (
	"math"
	"testing"

	"galaxy/internal/vec"
)

// fixedHydro is a hydro without forces that demands given time steps, by body.
type fixedHydro struct{ dt []float64 }

func (f fixedHydro) Forces(s *State, _ *Params, active []bool, _ []float64, out *HydroOut) {
	for i := range s.Len() {
		if s.Kind[i] == Gas && (active == nil || active[i]) {
			out.Dt[i] = f.dt[i]
		}
	}
}
func (fixedHydro) Bound(*State, *Params, []float64) {}

func TestRungFor(t *testing.T) {
	for _, c := range []struct {
		dt   float64
		want int
	}{{1, 0}, {math.Inf(1), 0}, {0.5, 1}, {0.4, 2}, {0.26, 2}, {0.25, 2}, {1e-12, maxRung}} {
		if r := rungFor(c.dt, 1); r != c.want {
			t.Errorf("rungFor(%v, 1) = %d, want %d", c.dt, r, c.want)
		}
	}
}

// Each gas body moves in steps of Dt/2^r just below its Courant step; only the bodies whose step
// ends are evaluated, and the global step still adds up to Dt.
func TestGasMovesInItsOwnSteps(t *testing.T) {
	s := NewState(2)
	s.Add(Body{Pos: vec.Vec3{X: 1}, Vel: vec.Vec3{Y: 1}, Mass: 1, Kind: Gas})
	s.Add(Body{Pos: vec.Vec3{X: -1}, Vel: vec.Vec3{Y: -1}, Mass: 1, Kind: Gas})
	m := New(s, nopSolver{}, Params{Dt: 0.01})
	m.Hydro = fixedHydro{dt: []float64{0.003, 1}}
	m.Step()
	m.Step()
	// Body 0: rung 2 (0.0025), four evaluations; body 1: one, at the end.
	if m.HydroSubsteps != 4 || math.Abs(m.HydroLoad-2.5) > 1e-12 {
		t.Errorf("%d evaluations, load %v; want 4 and 2.5", m.HydroSubsteps, m.HydroLoad)
	}
	if math.Abs(m.Time-0.02) > 1e-12 || math.Abs(s.PosY[0]-0.02) > 1e-12 || math.Abs(s.PosY[1]+0.02) > 1e-12 {
		t.Errorf("time %v, positions %v %v after two steps of 0.01", m.Time, s.PosY[0], s.PosY[1])
	}
	m.Hydro = fixedHydro{dt: []float64{1e-9, 1e-9}}
	m.Step()
	m.Step()
	if m.HydroSubsteps != fullTicks {
		t.Errorf("%d evaluations, want the cap %d", m.HydroSubsteps, fullTicks)
	}
}

// wakeHydro gives body 0 a short step and wants body 1 woken up whenever body 0 is evaluated;
// body 1 feels a constant acceleration and counts its evaluations.
type wakeHydro struct{ evals *int }

func (w wakeHydro) Forces(s *State, _ *Params, active []bool, _ []float64, out *HydroOut) {
	if active == nil || active[0] {
		out.Dt[0] = 1.0 / 800
		out.Wake[1] = math.Float64bits(out.Dt[0])
	}
	if active == nil || active[1] {
		out.Dt[1], out.AccX[1], out.DuDt[1] = 1, 2, 3
		*w.evals++
	}
}
func (wakeHydro) Bound(*State, *Params, []float64) {}

// A body whose neighbor gets a much shorter step ends its own step early (wake-up); its kicks
// still add up exactly.
func TestGasBodiesAreWokenUp(t *testing.T) {
	s := NewState(2)
	s.Add(Body{Pos: vec.Vec3{X: 1}, Mass: 1, Kind: Gas, U: 1})
	s.Add(Body{Pos: vec.Vec3{X: -1}, Mass: 1, Kind: Gas, U: 1})
	evals := 0
	m := New(s, nopSolver{}, Params{Dt: 0.01})
	m.Hydro = wakeHydro{&evals}
	m.Step() // learns the steps
	evals = 0
	m.Step()
	// Body 0 runs at Dt/8; body 1 may take at most 4 times that, so it is woken up to end at
	// Dt/2 and evaluated twice instead of once.
	if m.HydroSubsteps != 8 || evals != 2 {
		t.Errorf("%d evaluations, body 1 evaluated %d times; want 8 and 2", m.HydroSubsteps, evals)
	}
	if v := s.VelX[1]; math.Abs(v-2*0.02) > 1e-15 {
		t.Errorf("velocity %v after two steps with acceleration 2, want %v", v, 2*0.02)
	}
	if u := s.U[1]; math.Abs(u-(1+3*0.02)) > 1e-12 {
		t.Errorf("internal energy %v, want %v", u, 1+3*0.02)
	}
}
