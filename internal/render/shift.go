package render

import "math"

// Relativistic color shift of a star as seen by the camera (T-017). Purely visual.

// frequencyShift returns ν_observed/ν_emitted for a star with coordinate velocity v, seen from
// direction n (unit vector from the star towards the camera), at speed of light c, multiplied by
// the gravitational redshift factor grav = Π √(1 − r_s/r) of the holes around it.
// Doppler: D = 1 / (γ (1 − β·n)), approaching stars (β·n > 0) are blueshifted.
func frequencyShift(vx, vy, vz, nx, ny, nz, c, grav float64) float64 {
	if c <= 0 {
		return grav
	}
	bx, by, bz := vx/c, vy/c, vz/c
	b2 := math.Min(bx*bx+by*by+bz*bz, 0.9999)
	gamma := 1 / math.Sqrt(1-b2)
	return grav / (gamma * (1 - (bx*nx + by*ny + bz*nz)))
}

// gravShift returns √(1 − r_s/r), clamped so stars right at a horizon stay faintly visible.
func gravShift(r, rs float64) float64 {
	if rs <= 0 {
		return 1
	}
	return math.Sqrt(math.Max(0.02, 1-rs/r))
}

// Tints for strongly blue- and redshifted light.
var (
	blueTint = [3]float32{0.55, 0.7, 1.0}
	redTint  = [3]float32{1.0, 0.3, 0.12}
)

// shiftColor tints a color by frequency factor f and returns a brightness factor for relativistic
// beaming (∝ f³, clamped). The tint saturates at a factor of ~2 in either direction.
func shiftColor(r, g, b float32, f float64) (float32, float32, float32, float32) {
	t := float32(math.Max(-1, math.Min(1, 2*math.Log2(f))))
	tint := blueTint
	if t < 0 {
		tint, t = redTint, -t
	}
	mix := func(a, b float32) float32 { return a + (b-a)*t }
	beam := float32(math.Max(0.25, math.Min(2.5, f*f*f)))
	return mix(r, tint[0]), mix(g, tint[1]), mix(b, tint[2]), beam
}
