package render

import (
	"math"
	"testing"

	"galaxy/internal/vec"
)

func testCamera() Camera {
	c := NewCamera()
	c.W, c.H = 1200, 800
	c.Yaw, c.Pitch = 0.7, 0.4
	c.Target = vec.Vec3{X: 1, Y: -2, Z: 0.5}
	return c
}

func TestUnprojectInvertsProject(t *testing.T) {
	c := testCamera()
	for _, s := range [][2]float64{{600, 400}, {0, 0}, {1100, 50}, {321, 777}} {
		p := c.Unproject(s[0], s[1])
		sx, sy, depth, ok := c.Project(p.X, p.Y, p.Z)
		if !ok || math.Abs(sx-s[0]) > 1e-9 || math.Abs(sy-s[1]) > 1e-9 || math.Abs(depth-c.Distance) > 1e-9 {
			t.Errorf("screen %v -> %v -> (%v, %v, depth %v)", s, p, sx, sy, depth)
		}
	}
}

func TestTopDownViewKeepsOrientation(t *testing.T) {
	c := NewCamera()
	c.W, c.H, c.Pitch = 800, 800, math.Pi/2
	sx, sy, _, _ := c.Project(1, 0, 0)
	if sx <= 400 || math.Abs(sy-400) > 1e-9 {
		t.Errorf("+x should be right of center: %v, %v", sx, sy)
	}
	sx, sy, _, _ = c.Project(0, 1, 0)
	if sy >= 400 || math.Abs(sx-400) > 1e-9 {
		t.Errorf("+y should be above center: %v, %v", sx, sy)
	}
}

func TestPanMovesTargetWithCursor(t *testing.T) {
	c := testCamera()
	p := c.Unproject(500, 300)
	c.Pan(40, -25)
	sx, sy, _, _ := c.Project(p.X, p.Y, p.Z)
	if math.Abs(sx-540) > 1e-9 || math.Abs(sy-275) > 1e-9 {
		t.Errorf("point should follow the drag, got %v, %v", sx, sy)
	}
}
