// Package render projects the 3D simulation onto the screen.
package render

import (
	"math"

	"galaxy/internal/vec"
)

// Camera orbits around Target. Z is "up" in world space; the galactic plane is XY.
type Camera struct {
	Target   vec.Vec3
	Yaw      float64 // rotation around world z
	Pitch    float64 // 0 = edge-on, π/2 = top-down
	Distance float64 // from Target along the view axis
	FOV      float64 // vertical field of view, radians

	W, H float64 // screen size in pixels
}

func NewCamera() Camera {
	return Camera{Pitch: 0.9, Distance: 10, FOV: 0.9}
}

func (c *Camera) focal() float64 { return c.H / (2 * math.Tan(c.FOV/2)) }

// Project maps a world position to screen coordinates. depth is the distance from the camera
// along the view axis; ok is false for points behind the near plane.
func (c *Camera) Project(x, y, z float64) (sx, sy, depth float64, ok bool) {
	x, y, z = x-c.Target.X, y-c.Target.Y, z-c.Target.Z
	sy1, cy1 := math.Sincos(c.Yaw)
	sp, cp := math.Sincos(c.Pitch)
	x1 := cy1*x - sy1*y
	y1 := sy1*x + cy1*y
	fwd := cp*y1 - sp*z // along the view direction
	up := sp*y1 + cp*z
	depth = c.Distance + fwd
	if depth < 0.05 {
		return 0, 0, depth, false
	}
	f := c.focal() / depth
	return c.W/2 + f*x1, c.H/2 - f*up, depth, true
}

// Unproject maps a screen position to the world point in the focal plane, the plane through
// Target perpendicular to the view direction.
func (c *Camera) Unproject(sx, sy float64) vec.Vec3 {
	f := c.Distance / c.focal()
	x1, up := (sx-c.W/2)*f, -(sy-c.H/2)*f
	sy1, cy1 := math.Sincos(c.Yaw)
	sp, cp := math.Sincos(c.Pitch)
	// Invert the pitch rotation with fwd = 0, then the yaw rotation.
	y1, z := sp*up, cp*up
	x, y := cy1*x1+sy1*y1, -sy1*x1+cy1*y1
	return vec.Vec3{X: x, Y: y, Z: z}.Add(c.Target)
}

// Pan moves the target by a screen-space delta in pixels.
func (c *Camera) Pan(dx, dy float64) {
	a := c.Unproject(c.W/2, c.H/2)
	b := c.Unproject(c.W/2+dx, c.H/2+dy)
	c.Target = c.Target.Sub(b.Sub(a))
}

// Zoom scales the distance; factor < 1 zooms in.
func (c *Camera) Zoom(factor float64) {
	c.Distance = math.Min(500, math.Max(0.5, c.Distance*factor))
}

// Rotate changes yaw and pitch by screen-space drag deltas in pixels.
func (c *Camera) Rotate(dx, dy float64) {
	c.Yaw += dx * 0.005
	c.Pitch = math.Min(math.Pi/2, math.Max(-math.Pi/2, c.Pitch+dy*0.005))
}
