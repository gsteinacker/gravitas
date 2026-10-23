package sim

import (
	"math"
	"testing"

	"galaxy/internal/vec"
)

func TestAddAndCompact(t *testing.T) {
	s := NewState(4)
	for i := range 4 {
		s.Add(Body{Pos: vec.Vec3{X: float64(i)}, Mass: float64(i + 1)})
	}
	s.Alive[1], s.Alive[2] = false, false
	s.Compact()
	if s.Len() != 2 || s.Mass[0] != 1 || s.Mass[1] != 4 || s.PosX[1] != 3 || len(s.AccZ) != 2 {
		t.Fatalf("unexpected state after compact: %+v", s)
	}
}

func TestStepDoesNotAllocate(t *testing.T) {
	s := NewState(3)
	for i := range 3 {
		s.Add(Body{Pos: vec.Vec3{X: float64(i)}, Mass: 1})
	}
	m := New(s, nopSolver{}, Params{Dt: 0.01})
	if allocs := testing.AllocsPerRun(100, m.Step); allocs != 0 {
		t.Errorf("Step allocates %v times", allocs)
	}
}

type nopSolver struct{}

func (nopSolver) Accelerations(*State, Params) {}

func TestPotentialEnergyParallelMatchesSerial(t *testing.T) {
	s := NewState(500)
	for i := range 500 {
		x := float64(i)
		s.Add(Body{Pos: vec.Vec3{X: x * 0.01, Y: x * x * 1e-4, Z: -x * 0.003}, Mass: 1 + x*0.001})
	}
	_, want := Energy(s, 0.05)
	if got := PotentialEnergyParallel(s, 0.05); math.Abs((got-want)/want) > 1e-12 {
		t.Errorf("parallel U = %v, serial %v", got, want)
	}
}
