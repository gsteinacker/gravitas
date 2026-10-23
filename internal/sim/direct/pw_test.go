package direct

import (
	"math"
	"testing"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// circularOrbit sets up a test star on a circular orbit of radius r = k·r_s around a hole of mass 1,
// with its proper velocity scaled by perturb. In this model a circular orbit needs
// u·v / r = M / (r − r_s)², with u = γv; solved for v².
func circularOrbit(k, c, perturb float64) (*sim.Simulation, float64) {
	rs := sim.SchwarzschildRadius(1, c)
	r := k * rs
	a := r / ((r - rs) * (r - rs))
	x := (-a*a/(c*c) + math.Sqrt(a*a*a*a/(c*c*c*c)+4*a*a)) / 2
	v := math.Sqrt(x)
	s := sim.NewState(2)
	s.Add(sim.Body{Mass: 1, Kind: sim.BlackHole})
	s.Add(sim.Body{Pos: vec.Vec3{X: r}, Vel: sim.ProperVelocity(vec.Vec3{Y: v * perturb}, c), Mass: 1e-9})
	period := 2 * math.Pi * r / v
	return sim.New(s, &Solver{}, sim.Params{Dt: period / 2000, Softening: 1e-4 * rs, C: c}), period
}

func TestCircularOrbitIsCircular(t *testing.T) {
	m, period := circularOrbit(6, 10, 1)
	r0 := m.State.Body(1).Pos.Len()
	for range int(10 * period / m.Params.Dt) {
		m.Step()
		if r := m.State.Body(1).Pos.Sub(m.State.Body(0).Pos).Len(); math.Abs(r-r0) > 1e-4*r0 {
			t.Fatalf("radius %.6f drifted from %.6f", r, r0)
		}
	}
}

func TestOrbitOutsideISCOIsStable(t *testing.T) {
	m, period := circularOrbit(6, 10, 0.97)
	r0 := m.State.Body(1).Pos.Len()
	e0, _ := sim.TotalEnergy(m)
	minR := r0
	for range int(30 * period / m.Params.Dt) {
		m.Step()
		if m.State.Len() < 2 {
			t.Fatal("star at 6 r_s plunged, but orbits outside the ISCO must be stable")
		}
		minR = math.Min(minR, m.State.Body(1).Pos.Sub(m.State.Body(0).Pos).Len())
	}
	if minR < 0.7*r0 {
		t.Errorf("perturbed orbit dipped to %.2f r0, expected a mildly eccentric orbit", minR/r0)
	}
	// Leapfrog energy errors oscillate with amplitude ∝ dt² (here ~1e-6 at dt = P/2000) but don't grow.
	if e, _ := sim.TotalEnergy(m); math.Abs((e-e0)/e0) > 1e-5 {
		t.Errorf("Paczyński-Wiita orbit energy drift %.3g", (e-e0)/e0)
	}
}

func TestOrbitInsideISCOPlunges(t *testing.T) {
	m, period := circularOrbit(2, 10, 0.97)
	for range int(30 * period / m.Params.Dt) {
		m.Step()
		if m.State.Len() == 1 {
			return // swallowed
		}
	}
	t.Error("perturbed orbit at 2 r_s did not plunge into the hole")
}

func TestNewtonianOrbitAt2RsWouldNotPlunge(t *testing.T) {
	// Control: the same perturbed orbit with the Paczyński-Wiita term switched off (no hole kind)
	// stays bound, so the plunge above is caused by the potential, not by numerics.
	m, period := circularOrbit(2, 10, 0.97)
	m.State.Kind[0] = sim.Star
	m.Params.C = 1e9 // Newtonian kinematics and potential
	v := math.Sqrt(1 / m.State.Body(1).Pos.Len())
	m.State.VelY[1] = 0.97 * v
	r0 := m.State.Body(1).Pos.Len()
	for range int(30 * period / m.Params.Dt) {
		m.Step()
	}
	if r := m.State.Body(1).Pos.Sub(m.State.Body(0).Pos).Len(); r < 0.5*r0 || r > 1.5*r0 {
		t.Errorf("Newtonian control orbit left the band: r = %.2f r0", r/r0)
	}
}

func TestStarHoleForcesConserveMomentum(t *testing.T) {
	s := randomState(300, 9)
	s.Kind[0], s.Mass[0] = sim.BlackHole, 20
	s.Kind[1], s.Mass[1] = sim.BlackHole, 10
	m := sim.New(s, &Solver{}, sim.Params{Dt: 1e-4, Softening: 0.05, C: 8})
	p0 := sim.Momentum(s)
	for range 100 {
		m.Step()
	}
	if d := sim.Momentum(s).Sub(p0).Len(); d > 1e-10 {
		t.Errorf("momentum changed by %.3g", d)
	}
}
