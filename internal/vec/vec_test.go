package vec

import (
	"math"
	"testing"
)

func near(a, b Vec3) bool { return a.Sub(b).Len() < 1e-12 }

func TestBasics(t *testing.T) {
	a, b := Vec3{1, 2, 3}, Vec3{4, 5, 6}
	if got := a.Add(b); got != (Vec3{5, 7, 9}) {
		t.Errorf("Add = %v", got)
	}
	if got := b.Sub(a); got != (Vec3{3, 3, 3}) {
		t.Errorf("Sub = %v", got)
	}
	if got := a.Dot(b); got != 32 {
		t.Errorf("Dot = %v", got)
	}
	if got := (Vec3{1, 0, 0}).Cross(Vec3{0, 1, 0}); got != (Vec3{0, 0, 1}) {
		t.Errorf("Cross = %v", got)
	}
	if got := (Vec3{3, 4, 0}).Len(); got != 5 {
		t.Errorf("Len = %v", got)
	}
}

func TestRotations(t *testing.T) {
	if got := (Vec3{1, 0, 0}).RotateZ(math.Pi / 2); !near(got, Vec3{0, 1, 0}) {
		t.Errorf("RotateZ = %v", got)
	}
	if got := (Vec3{0, 1, 0}).RotateX(math.Pi / 2); !near(got, Vec3{0, 0, 1}) {
		t.Errorf("RotateX = %v", got)
	}
}
