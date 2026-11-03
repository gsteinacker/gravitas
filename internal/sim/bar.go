package sim

import (
	"cmp"
	"math"
	"slices"
)

// Bar is the m = 2 distortion of the stellar disk (T-090): A2 = |Σ m e^{2iφ}| / Σ m of the stars
// in the plane of their rotation, between 0.15 and 1 stellar half-mass radius around the center.
// A2 ≳ 0.2 with a steady phase is a bar (spirals change phase with radius and time).
type Bar struct {
	A2     float64 // amplitude, 0 = axisymmetric
	Phase  float64 // position angle of the bar in its plane (radians, modulo π)
	Radius float64 // outer radius of the measured ring (the stellar half-mass radius)
}

// MeasureBar measures the bar of the galaxy at the densest stars near the heaviest black hole (or
// near the stars' center of mass if there is none). It is O(N) plus a sort and meant for occasional diagnostics.
func MeasureBar(s *State) Bar {
	return measureBar(s.PosX, s.PosY, s.PosZ, s.VelX, s.VelY, s.VelZ, s.Mass, s.Kind, s.Alive)
}

// Bar measures the bar in a snapshot (see MeasureBar).
func (sn *Snapshot) Bar() Bar {
	return measureBar(sn.X, sn.Y, sn.Z, sn.VX, sn.VY, sn.VZ, sn.Mass, sn.Kind, nil)
}

// measureBar implements MeasureBar for State and Snapshot; alive nil means all bodies live.
func measureBar[F float32 | float64](px, py, pz, vx, vy, vz, mass []F, kind []Kind, alive []bool) Bar {
	n := len(mass)
	live := func(i int) bool { return alive == nil || alive[i] }
	pos := func(i int) (float64, float64, float64) { return float64(px[i]), float64(py[i]), float64(pz[i]) }
	vel := func(i int) (float64, float64, float64) { return float64(vx[i]), float64(vy[i]), float64(vz[i]) }
	var c center
	// Start at the heaviest hole (it picks the galaxy in scenes with several), else at the stars'
	// center of mass; then shrink spheres onto the densest stars. A light hole wanders through
	// the heavy dark matter particles by up to a few kpc, so it is no good center itself.
	hole, heaviest := -1, 0.0
	for i := range n {
		if live(i) && kind[i] == BlackHole && float64(mass[i]) > heaviest {
			hole, heaviest = i, float64(mass[i])
		}
	}
	sphere := func(r2 float64) (cx, cy, cz, m float64, count int) {
		for i := range n {
			if !live(i) || kind[i] != Star {
				continue
			}
			x, y, z := pos(i)
			if dx, dy, dz := x-c.X, y-c.Y, z-c.Z; dx*dx+dy*dy+dz*dz > r2 {
				continue
			}
			w := float64(mass[i])
			cx, cy, cz, m, count = cx+w*x, cy+w*y, cz+w*z, m+w, count+1
		}
		return
	}
	if hole >= 0 {
		c.X, c.Y, c.Z = pos(hole)
	}
	cx, cy, cz, m, all := sphere(math.Inf(1))
	if m == 0 || all < 10 {
		return Bar{}
	}
	if hole < 0 {
		c.X, c.Y, c.Z = cx/m, cy/m, cz/m
	}
	var r2 float64 // start with the rms distance of the stars
	for i := range n {
		if live(i) && kind[i] == Star {
			x, y, z := pos(i)
			r2 += float64(mass[i]) * ((x-c.X)*(x-c.X) + (y-c.Y)*(y-c.Y) + (z-c.Z)*(z-c.Z))
		}
	}
	r2 /= m
	for {
		cx, cy, cz, m, count := sphere(r2)
		if count < max(50, all/50) || m == 0 {
			break
		}
		c.X, c.Y, c.Z = cx/m, cy/m, cz/m
		r2 *= 0.8
	}
	// Velocity of the center: the stars within a few final radii.
	var mv float64
	for i := range n {
		if !live(i) || kind[i] != Star {
			continue
		}
		x, y, z := pos(i)
		if dx, dy, dz := x-c.X, y-c.Y, z-c.Z; dx*dx+dy*dy+dz*dz > 9*r2 {
			continue
		}
		u, v, q := vel(i)
		w := float64(mass[i])
		c.VX, c.VY, c.VZ, mv = c.VX+w*u, c.VY+w*v, c.VZ+w*q, mv+w
	}
	if mv > 0 {
		c.VX, c.VY, c.VZ = c.VX/mv, c.VY/mv, c.VZ/mv
	}
	// Half-mass radius of the stars around the center.
	type star struct{ r, m float64 }
	var stars []star
	var total float64
	for i := range n {
		if live(i) && kind[i] == Star {
			x, y, z := pos(i)
			x, y, z = x-c.X, y-c.Y, z-c.Z
			stars = append(stars, star{math.Sqrt(x*x + y*y + z*z), float64(mass[i])})
			total += float64(mass[i])
		}
	}
	if len(stars) < 10 {
		return Bar{}
	}
	slices.SortFunc(stars, func(a, b star) int { return cmp.Compare(a.r, b.r) })
	r50, acc := 0.0, 0.0
	for _, st := range stars {
		if acc += st.m; acc >= total/2 {
			r50 = st.r
			break
		}
	}
	// Rotation axis of the stars inside r50.
	var lx, ly, lz float64
	for i := range n {
		if !live(i) || kind[i] != Star {
			continue
		}
		x, y, z := pos(i)
		x, y, z = x-c.X, y-c.Y, z-c.Z
		if x*x+y*y+z*z > r50*r50 {
			continue
		}
		u, v, w := vel(i)
		u, v, w = u-c.VX, v-c.VY, w-c.VZ
		m := float64(mass[i])
		lx, ly, lz = lx+m*(y*w-z*v), ly+m*(z*u-x*w), lz+m*(x*v-y*u)
	}
	l := math.Sqrt(lx*lx + ly*ly + lz*lz)
	if l == 0 {
		return Bar{}
	}
	lx, ly, lz = lx/l, ly/l, lz/l
	// Orthonormal basis (e1, e2) of the plane; for a disk in the xy plane e1 = x, e2 = y.
	ax, ay, az := 0.0, 1.0, 0.0
	if math.Abs(ly) > 0.9 {
		ax, ay = 1, 0
	}
	e1x, e1y, e1z := ay*lz-az*ly, az*lx-ax*lz, ax*ly-ay*lx
	norm := math.Sqrt(e1x*e1x + e1y*e1y + e1z*e1z)
	e1x, e1y, e1z = e1x/norm, e1y/norm, e1z/norm
	e2x, e2y, e2z := ly*e1z-lz*e1y, lz*e1x-lx*e1z, lx*e1y-ly*e1x
	var re, im, w float64
	for i := range n {
		if !live(i) || kind[i] != Star {
			continue
		}
		x, y, z := pos(i)
		x, y, z = x-c.X, y-c.Y, z-c.Z
		p, q := x*e1x+y*e1y+z*e1z, x*e2x+y*e2y+z*e2z
		r2 := p*p + q*q
		if r2 < 0.0225*r50*r50 || r2 > r50*r50 {
			continue
		}
		// e^{2iφ} = (p + iq)² / r²
		m := float64(mass[i])
		re += m * (p*p - q*q) / r2
		im += m * 2 * p * q / r2
		w += m
	}
	if w == 0 {
		return Bar{}
	}
	return Bar{A2: math.Hypot(re, im) / w, Phase: 0.5 * math.Atan2(im, re), Radius: r50}
}

// center is the position and velocity MeasureBar measures around.
type center struct{ X, Y, Z, VX, VY, VZ float64 }
