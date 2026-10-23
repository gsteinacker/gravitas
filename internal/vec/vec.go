// Package vec provides a minimal 3D vector type for setup and diagnostics code.
// Hot loops work on struct-of-arrays slices instead.
package vec

import "math"

type Vec3 struct{ X, Y, Z float64 }

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(f float64) Vec3 { return Vec3{a.X * f, a.Y * f, a.Z * f} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}
func (a Vec3) Len2() float64 { return a.Dot(a) }
func (a Vec3) Len() float64  { return math.Sqrt(a.Dot(a)) }

// RotateX rotates a around the x axis by angle (radians).
func (a Vec3) RotateX(angle float64) Vec3 {
	s, c := math.Sincos(angle)
	return Vec3{a.X, c*a.Y - s*a.Z, s*a.Y + c*a.Z}
}

// RotateZ rotates a around the z axis by angle (radians).
func (a Vec3) RotateZ(angle float64) Vec3 {
	s, c := math.Sincos(angle)
	return Vec3{c*a.X - s*a.Y, s*a.X + c*a.Y, a.Z}
}
