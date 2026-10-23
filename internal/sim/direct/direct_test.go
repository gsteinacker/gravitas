package direct

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

func randomState(n int, seed uint64) *sim.State {
	rng := rand.New(rand.NewPCG(seed, 1))
	s := sim.NewState(n)
	for range n {
		s.Add(sim.Body{
			Pos:  vec.Vec3{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()},
			Vel:  vec.Vec3{X: rng.NormFloat64() * 0.1, Y: rng.NormFloat64() * 0.1, Z: rng.NormFloat64() * 0.1},
			Mass: 0.5 + rng.Float64(),
		})
	}
	return s
}

// keplerState builds a two-body orbit with eccentricity e and semi-major axis 1, total mass 1.
func keplerState(e float64) (s *sim.State, period float64) {
	m1, m2 := 0.999, 0.001
	rp := 1 - e
	vp := math.Sqrt((1 + e) / rp) // vis-viva at pericenter for a = 1, M = 1
	s = sim.NewState(2)
	s.Add(sim.Body{Pos: vec.Vec3{X: -rp * m2}, Vel: vec.Vec3{Y: -vp * m2}, Mass: m1})
	s.Add(sim.Body{Pos: vec.Vec3{X: rp * m1}, Vel: vec.Vec3{Y: vp * m1}, Mass: m2})
	return s, 2 * math.Pi
}

func TestKeplerConservation(t *testing.T) {
	s, period := keplerState(0.3)
	const stepsPerOrbit, orbits = 2000, 1000
	p := sim.Params{Dt: period / stepsPerOrbit, Softening: 0}
	m := sim.New(s, &Solver{}, p)

	k, u := sim.Energy(s, 0)
	e0, l0 := k+u, sim.AngularMomentum(s).Z
	var maxDrift float64
	for i := range stepsPerOrbit * orbits {
		m.Step()
		if i%100 == 0 {
			k, u := sim.Energy(s, 0)
			maxDrift = max(maxDrift, math.Abs((k+u-e0)/e0))
		}
	}
	if maxDrift > 1e-4 {
		t.Errorf("relative energy drift %.3g > 1e-4", maxDrift)
	}
	if dl := math.Abs(sim.AngularMomentum(s).Z-l0) / l0; dl > 1e-10 {
		t.Errorf("relative angular momentum drift %.3g", dl)
	}
	// After an integer number of orbits the light body is back near pericenter.
	if b := s.Body(1); math.Abs(b.Pos.Len()-0.7) > 0.05 {
		t.Errorf("orbit not closed: r = %.4f, want ≈ 0.7", b.Pos.Len())
	}
}

func TestMomentumConservationParallel(t *testing.T) {
	s := randomState(1000, 42)
	m := sim.New(s, &Solver{}, sim.Params{Dt: 1e-3, Softening: 0.05})
	p0 := sim.Momentum(s)
	for range 50 {
		m.Step()
	}
	if d := sim.Momentum(s).Sub(p0).Len(); d > 1e-12 {
		t.Errorf("momentum changed by %.3g", d)
	}
}

func TestParallelMatchesSerial(t *testing.T) {
	a, b := randomState(1000, 7), randomState(1000, 7)
	p := sim.Params{Softening: 0.01}
	(&Solver{Workers: 1}).Accelerations(a, p)
	(&Solver{Workers: 8}).Accelerations(b, p)
	for i := range a.Len() {
		if a.AccX[i] != b.AccX[i] || a.AccY[i] != b.AccY[i] || a.AccZ[i] != b.AccZ[i] {
			t.Fatalf("body %d differs", i)
		}
	}
}

func BenchmarkAccelerations(b *testing.B) {
	for _, n := range []int{1000, 10000} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			s := randomState(n, 1)
			solver := &Solver{}
			p := sim.Params{Softening: 0.01}
			b.ReportAllocs()
			for b.Loop() {
				solver.Accelerations(s, p)
			}
			b.ReportMetric(float64(n)*float64(n)*float64(b.N)/b.Elapsed().Seconds()/1e9, "Gpairs/s")
		})
	}
}
