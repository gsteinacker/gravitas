package sim

import (
	"math"

	"galaxy/internal/vec"
)

// Frame dragging of rotating black holes (T-072, ADR-0013): the gravitomagnetic (Lense–Thirring)
// acceleration of order 1/c² on a body moving with v at r from a hole with angular momentum J,
//
//	a = (2G / (c² r³)) [3 (J·r̂)(r̂ × v) + v × J] = v × B,   B = (2G / (c² r³)) (J − 3 (J·r̂) r̂).
//
// It is perpendicular to v and does no work, so it is integrated as a rotation of the velocity
// (Boris 1970) inside the kicks: the speed is kept exactly. The hole takes the opposite momentum
// and the opposite angular momentum (into its spin), so momentum and total angular momentum
// (orbits plus spins) are conserved. It makes prograde orbits more stable than retrograde ones
// and lets orbital planes precess about J with Ω = 2GJ/(c² a³ (1 − e²)^{3/2}).

// spinning reports whether any hole has a spin, i.e. whether frame dragging acts at all.
func (m *Simulation) spinning() bool {
	if m.Params.C <= 0 {
		return false
	}
	s := m.State
	for _, h := range m.holes {
		if s.SpinX[h] != 0 || s.SpinY[h] != 0 || s.SpinZ[h] != 0 {
			return true
		}
	}
	return false
}

// gravitomagnetic returns B at body i from all spinning holes except i itself (softened like the
// Newtonian force).
func (m *Simulation) gravitomagnetic(i int, eps2 float64) vec.Vec3 {
	s, c := m.State, m.Params.C
	var b vec.Vec3
	for _, h := range m.holes {
		if h == i || !s.Alive[h] {
			continue
		}
		j := s.Spin(h)
		if j.Len2() == 0 {
			continue
		}
		r := vec.Vec3{X: s.PosX[i] - s.PosX[h], Y: s.PosY[i] - s.PosY[h], Z: s.PosZ[i] - s.PosZ[h]}
		r2 := r.Len2() + eps2
		inv := 1 / math.Sqrt(r2)
		k := 2 / (c * c) * inv * inv * inv
		b = b.Add(j.Sub(r.Scale(3 * j.Dot(r) / r2)).Scale(k)) // 3 (J·r̂) r̂ with the softened length
	}
	return b
}

// dragKick rotates the velocity of body i by the gravitomagnetic field over time h (Boris
// rotation, exact in |u|) and gives the holes the opposite momentum and angular momentum.
func (m *Simulation) dragKick(i int, h, eps2 float64) {
	s := m.State
	b := m.gravitomagnetic(i, eps2)
	if b.Len2() == 0 {
		return
	}
	u := vec.Vec3{X: s.VelX[i], Y: s.VelY[i], Z: s.VelZ[i]}
	g := gamma(u.X, u.Y, u.Z, m.Params.C)
	t := b.Scale(0.5 * h / g) // a = v × B with v = u/γ
	up := u.Add(u.Cross(t))
	un := u.Add(up.Cross(t.Scale(2 / (1 + t.Len2()))))
	du := un.Sub(u)
	s.VelX[i], s.VelY[i], s.VelZ[i] = un.X, un.Y, un.Z
	m.dragBack(i, du)
}

// dragBack books the momentum m·du that body i gained from frame dragging on the holes, in
// proportion to their share of B at i, as momentum and as spin (angular momentum about the
// body: −r_ih × m du).
func (m *Simulation) dragBack(i int, du vec.Vec3) {
	s := m.State
	p := du.Scale(s.Mass[i])
	var total float64
	weights := m.dragWeights[:0]
	for _, h := range m.holes {
		w := 0.0
		if h != i && s.Alive[h] {
			r := vec.Vec3{X: s.PosX[i] - s.PosX[h], Y: s.PosY[i] - s.PosY[h], Z: s.PosZ[i] - s.PosZ[h]}
			if j := s.Spin(h).Len(); j > 0 {
				w = j / math.Pow(r.Len2(), 1.5)
			}
		}
		weights = append(weights, w)
		total += w
	}
	m.dragWeights = weights
	if total == 0 {
		return
	}
	for k, h := range m.holes {
		if weights[k] == 0 {
			continue
		}
		f := weights[k] / total
		ph := p.Scale(-f)
		s.VelX[h] += ph.X / s.Mass[h]
		s.VelY[h] += ph.Y / s.Mass[h]
		s.VelZ[h] += ph.Z / s.Mass[h]
		// The pair's angular momentum: body i gains r_i × p, the hole r_h × (−p); the rest goes
		// into the spin so that the total (orbits plus spins) stays the same.
		r := vec.Vec3{X: s.PosX[i] - s.PosX[h], Y: s.PosY[i] - s.PosY[h], Z: s.PosZ[i] - s.PosZ[h]}
		dj := r.Cross(p.Scale(f)).Scale(-1)
		s.SpinX[h] += dj.X
		s.SpinY[h] += dj.Y
		s.SpinZ[h] += dj.Z
	}
}

// dragOuter applies frame dragging over time h to all bodies that are neither holes nor near
// (those get it in the substeps, dragFast).
func (m *Simulation) dragOuter(h float64) {
	if !m.spinning() {
		return
	}
	s := m.State
	eps2 := m.Params.Softening * m.Params.Softening
	for i := range s.Len() {
		if s.Kind[i] != BlackHole && !m.isNear[i] && s.Alive[i] {
			m.dragKick(i, h, eps2)
		}
	}
}

// dragFast applies frame dragging over time h to the near bodies and to holes in the field of
// other spinning holes.
func (m *Simulation) dragFast(h float64) {
	if !m.spinning() {
		return
	}
	eps2 := m.Params.Softening * m.Params.Softening
	for _, idx := range [][]int{m.near, m.holes} {
		for _, i := range idx {
			m.dragKick(i, h, eps2)
		}
	}
}
