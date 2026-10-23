package sim

import (
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
