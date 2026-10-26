package sim

import (
	"math"
	"math/rand/v2"
	"testing"

	"galaxy/internal/vec"
)

// TestFrameDraggingPrecessesTheNode: an inclined circular orbit around a spinning hole turns its
// ascending node at Ω_LT = 2GJ/(c² a³) (Lense & Thirring 1918), absent without spin.
func TestFrameDraggingPrecessesTheNode(t *testing.T) {
	const c, a, inc, tEnd = 10.0, 1.0, 30 * math.Pi / 180, 300.0
	run := func(spin float64) float64 {
		p := Params{Dt: 0.005, Softening: 1e-4, C: c}
		s := NewState(2)
		s.Add(Body{Mass: 1, Kind: BlackHole, Spin: vec.Vec3{Z: spin / c}})
		rs := SchwarzschildRadius(1, c)
		v := math.Sqrt(a) / (a - rs) // circular speed in the Paczyński–Wiita potential
		s.Add(Body{
			Pos:  vec.Vec3{X: a},
			Vel:  ProperVelocity(vec.Vec3{Y: v * math.Cos(inc), Z: v * math.Sin(inc)}, c),
			Mass: 1e-9,
		})
		m := New(s, pairSolver{}, p)
		node := func() float64 {
			l := vec.Vec3{X: s.PosX[1], Y: s.PosY[1], Z: s.PosZ[1]}.Cross(vec.Vec3{X: s.VelX[1], Y: s.VelY[1], Z: s.VelZ[1]})
			return math.Atan2(l.X, -l.Y)
		}
		n0 := node()
		for m.Time < tEnd {
			m.Step()
		}
		return math.Remainder(node()-n0, 2*math.Pi)
	}
	if d := run(0); math.Abs(d) > 1e-6 {
		t.Errorf("node moved by %.3g without spin", d)
	}
	const spin = 0.9
	want := 2 * (spin / c) / (c * c * a * a * a) * tEnd
	got := run(spin)
	t.Logf("node precession %.4f rad, Lense–Thirring %.4f", got, want)
	if math.Abs(got/want-1) > 0.1 {
		t.Errorf("node precession %.4f rad, want %.4f (Lense–Thirring)", got, want)
	}
}

// TestFrameDraggingConserves: with frame dragging, momentum and the total angular momentum
// (orbits plus spins) stay constant, and the speeds are not changed by the rotation itself.
func TestFrameDraggingConserves(t *testing.T) {
	p := Params{Dt: 0.002, Softening: 0.01, C: 5}
	s := NewState(64)
	s.Add(Body{Mass: 1, Kind: BlackHole, Spin: vec.Vec3{X: 0.05, Z: 0.15}})
	rng := rand.New(rand.NewPCG(4, 4))
	for range 50 {
		r := 0.6 + rng.Float64() // circular orbits in a thick disk, outside the ISCO (0.24)
		d := vec.Vec3{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: 0.2 * rng.NormFloat64()}
		pos := d.Scale(r / d.Len())
		vel := vec.Vec3{Z: 1}.Cross(pos.Scale(1 / pos.Len())).Scale(math.Sqrt(r) / (r - 0.08))
		s.Add(Body{Pos: pos, Vel: vel, Mass: 1e-4})
	}
	m := New(s, pairSolver{}, p)
	p0, l0 := Momentum(s), angularMomentum(s)
	for range 2000 {
		m.Step()
	}
	if d := Momentum(s).Sub(p0).Len(); d > 1e-12 {
		t.Errorf("momentum changed by %.3g", d)
	}
	if d := angularMomentum(s).Sub(l0).Len(); d > 1e-9*l0.Len() {
		t.Errorf("angular momentum changed by %.3g of %.3g", d, l0.Len())
	}
	// The rotation alone keeps |u| (accretion, which may have taken a body, conserves the above too).
	u := vec.Vec3{X: 0.3, Y: -0.2, Z: 0.1}
	s.VelX[1], s.VelY[1], s.VelZ[1] = u.X, u.Y, u.Z
	m.collectHoles()
	m.dragKick(1, 0.01, 1e-4)
	if got := (vec.Vec3{X: s.VelX[1], Y: s.VelY[1], Z: s.VelZ[1]}).Len(); math.Abs(got-u.Len()) > 1e-15 {
		t.Errorf("|u| changed from %v to %v", u.Len(), got)
	}
}
