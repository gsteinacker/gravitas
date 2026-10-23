package sim

import (
	"math"
	"testing"

	"galaxy/internal/vec"
)

// movingHole: a light hole moving uniformly along x at 0.3c, no relevant forces.
func movingHole(mode GravityMode) *Simulation {
	s := NewState(2)
	s.Add(Body{Vel: ProperVelocity(vec.Vec3{X: 3}, 10), Mass: 1e-12, Kind: BlackHole})
	s.Add(Body{Pos: vec.Vec3{X: 50}, Mass: 1e-12})
	m := New(s, nopSolver{}, Params{Dt: 0.004, Softening: 0.01, C: 10, Gravity: mode})
	for range 500 {
		m.Step()
	}
	return m
}

func TestRetardedPositionLiesOnTheLightCone(t *testing.T) {
	m := movingHole(GravityRetarded)
	obs := vec.Vec3{X: 1, Y: 4, Z: -2}
	m.collectHoles()
	m.bindHistories()
	x, y, z, ok := m.sourcePos(0, obs.X, obs.Y, obs.Z)
	if !ok {
		t.Fatal("hole not visible")
	}
	tr := x / 3 // uniform motion from the origin at v = 3
	if d := obs.Sub(vec.Vec3{X: x, Y: y, Z: z}).Len(); math.Abs(d-10*(m.Time-tr)) > 1e-6 {
		t.Errorf("distance %.9f, light travel %.9f", d, 10*(m.Time-tr))
	}
	if x >= m.State.PosX[0] {
		t.Errorf("retarded position %.4f should lag behind the current %.4f", x, m.State.PosX[0])
	}
}

func TestExtrapolatedRetardationPointsAtCurrentPositionForUniformMotion(t *testing.T) {
	m := movingHole(GravityRetardedExtrapolated)
	m.collectHoles()
	m.bindHistories()
	x, y, z, _ := m.sourcePos(0, 1, 4, -2)
	if d := (vec.Vec3{X: x - m.State.PosX[0], Y: y - m.State.PosY[0], Z: z - m.State.PosZ[0]}).Len(); d > 1e-6 {
		t.Errorf("extrapolated position is %.3g away from the current one", d)
	}
}

func TestGravityOfNewHoleSpreadsAtSpeedOfLight(t *testing.T) {
	s := NewState(2)
	s.Add(Body{Pos: vec.Vec3{X: 2}, Mass: 1e-9})
	m := New(s, nopSolver{}, Params{Dt: 0.004, Softening: 0.01, C: 10, Gravity: GravityRetardedExtrapolated})
	for range 10 {
		m.Step()
	}
	m.AddBody(Body{Mass: 1, Kind: BlackHole})
	placed := m.Time
	eps2 := m.Params.Softening * m.Params.Softening
	arrived := math.NaN()
	for range 100 {
		m.collectHoles()
		m.bindHistories()
		if ax, _, _ := m.holePull(0, eps2); ax != 0 && math.IsNaN(arrived) {
			arrived = m.Time
		}
		m.Step()
	}
	// The star sits at distance 2 (it barely moves): the front needs 2/c = 0.2.
	if math.Abs(arrived-placed-0.2) > 2*m.Params.Dt {
		t.Errorf("gravity arrived after %.3f, want 0.2", arrived-placed)
	}
}

// binaryDrift returns the relative change of the separation of an equal-mass hole binary on a
// circular orbit at v ≈ 0.1c after the given number of orbits.
func binaryDrift(mode GravityMode, orbits int) float64 {
	const m, d, c = 0.5, 1.0, 5.0
	v := math.Sqrt(m / (2 * d))
	s := NewState(2)
	s.Add(Body{Pos: vec.Vec3{X: -d / 2}, Vel: ProperVelocity(vec.Vec3{Y: -v}, c), Mass: m, Kind: BlackHole})
	s.Add(Body{Pos: vec.Vec3{X: d / 2}, Vel: ProperVelocity(vec.Vec3{Y: v}, c), Mass: m, Kind: BlackHole})
	period := 2 * math.Pi * (d / 2) / v
	sim := New(s, nopSolver{}, Params{Dt: period / 1000, Softening: 0.001, C: c, Gravity: mode})
	var sum float64
	steps := 0
	for range orbits * 1000 {
		sim.Step()
		if sim.State.Len() < 2 {
			return math.Inf(-1) // merged
		}
		if steps++; steps > (orbits-1)*1000 { // average over the last orbit
			sum += sim.State.Body(0).Pos.Sub(sim.State.Body(1).Pos).Len()
		}
	}
	return sum/1000/d - 1
}

// Measured (v ≈ 0.1c, ωΔt ≈ 0.2): separation change after 1 orbit instant +0.5 % (constant offset
// from the initial conditions), retarded +57 %, retarded+v +4.3 %; then retarded+v drifts ~2 % per
// orbit, about (ωΔt)² ≈ 1/25 of the naive rate. Extrapolation suppresses aberration but, lacking the
// other velocity-dependent terms of GR, does not remove the drift.
func TestNaiveRetardationDestabilizesOrbitsAndExtrapolationSuppressesIt(t *testing.T) {
	instant1, instant20 := binaryDrift(GravityInstant, 1), binaryDrift(GravityInstant, 20)
	naive1 := binaryDrift(GravityRetarded, 1)
	extrap1 := binaryDrift(GravityRetardedExtrapolated, 1)
	t.Logf("after 1 orbit: instant %+.2e, retarded %+.2e, retarded+v %+.2e", instant1, naive1, extrap1)
	if math.Abs(instant20-instant1) > 1e-3 {
		t.Errorf("instant binary must be stable: %+.2e after 1, %+.2e after 20 orbits", instant1, instant20)
	}
	if naive1 < 0.2 {
		t.Errorf("naive retardation should widen the orbit strongly within one orbit, got %+.2e", naive1)
	}
	if math.Abs(extrap1) > naive1/10 {
		t.Errorf("extrapolation should suppress the drift by an order of magnitude: %+.2e vs %+.2e", extrap1, naive1)
	}
}

func TestParseGravityMode(t *testing.T) {
	for _, g := range []GravityMode{GravityInstant, GravityRetarded, GravityRetardedExtrapolated} {
		if got, err := ParseGravityMode(g.String()); err != nil || got != g {
			t.Errorf("round trip %v: %v, %v", g, got, err)
		}
	}
	if _, err := ParseGravityMode("faster"); err == nil {
		t.Error("unknown mode must be an error")
	}
}

func TestRetardedGalaxyStaysBound(t *testing.T) {
	// A small disk around a hole in retarded+v mode: the hole moves slowly, so the model must stay
	// close to the instantaneous one.
	s := NewState(201)
	s.Add(Body{Mass: 1, Kind: BlackHole})
	for i := range 200 {
		r := 0.3 + 0.01*float64(i)
		phi := float64(i) * 2.4
		v := math.Sqrt(1 / r)
		s.Add(Body{Pos: vec.Vec3{X: r * math.Cos(phi), Y: r * math.Sin(phi)}, Vel: vec.Vec3{X: -v * math.Sin(phi), Y: v * math.Cos(phi)}, Mass: 1e-6})
	}
	m := New(s, pairSolver{}, Params{Dt: 0.002, Softening: 0.01, C: 10, Gravity: GravityRetardedExtrapolated})
	for range 2000 {
		m.Step()
	}
	for i := range s.Len() {
		if r := s.Body(i).Pos.Sub(s.Body(0).Pos).Len(); s.Kind[i] == Star && (r < 0.15 || r > 5) {
			t.Fatalf("star %d left its orbit band: r = %.3f", i, r)
		}
	}
}
