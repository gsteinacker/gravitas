package bh

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/direct"
	"galaxy/internal/vec"
)

func TestMortonKeyInterleavesBits(t *testing.T) {
	naive := func(x, y, z uint64) uint64 {
		var k uint64
		for b := range mortonBits {
			k |= (x>>b&1)<<(3*b+2) | (y>>b&1)<<(3*b+1) | (z>>b&1)<<(3*b)
		}
		return k
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		x, y, z := rng.Uint64N(1<<mortonBits), rng.Uint64N(1<<mortonBits), rng.Uint64N(1<<mortonBits)
		if got, want := mortonKey(x, y, z), naive(x, y, z); got != want {
			t.Fatalf("mortonKey(%d, %d, %d) = %x, want %x", x, y, z, got, want)
		}
	}
}

func TestRadixSort(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	n := 5000
	keys, idx := make([]uint64, n), make([]int32, n)
	for i := range keys {
		keys[i], idx[i] = rng.Uint64()>>1, int32(i)
	}
	orig := slices.Clone(keys)
	radixSort(keys, idx, make([]uint64, n), make([]int32, n))
	if !slices.IsSorted(keys) {
		t.Fatal("keys not sorted")
	}
	for i := range keys {
		if orig[idx[i]] != keys[i] {
			t.Fatalf("permutation broken at %d", i)
		}
	}
}

// galaxy returns a clustered test system: a disk with a central black hole plus a second black hole.
func galaxy(n int) (*sim.State, sim.Params) {
	s, p := scene.Presets[0].Build(n)
	s.Add(sim.Body{Pos: vec.Vec3{X: 1.5, Y: 0.3}, Mass: 0.5, Kind: sim.BlackHole})
	return s, p
}

func accelerations(s *sim.State, solver sim.Solver, p sim.Params) [][3]float64 {
	solver.Accelerations(s, p)
	out := make([][3]float64, s.Len())
	for i := range out {
		out[i] = [3]float64{s.AccX[i], s.AccY[i], s.AccZ[i]}
	}
	return out
}

// relErrors returns the median and maximum relative acceleration error of got vs. want.
func relErrors(got, want [][3]float64) (median, maxErr float64) {
	errs := make([]float64, len(got))
	for i := range got {
		dx, dy, dz := got[i][0]-want[i][0], got[i][1]-want[i][1], got[i][2]-want[i][2]
		w := math.Sqrt(want[i][0]*want[i][0] + want[i][1]*want[i][1] + want[i][2]*want[i][2])
		errs[i] = math.Sqrt(dx*dx+dy*dy+dz*dz) / w
	}
	slices.Sort(errs)
	return errs[len(errs)/2], errs[len(errs)-1]
}

func TestTinyThetaMatchesDirect(t *testing.T) {
	s, p := galaxy(3000)
	want := accelerations(s, &direct.Solver{}, p)
	got := accelerations(s, &Solver{Theta: 1e-9}, p)
	if _, maxErr := relErrors(got, want); maxErr > 1e-10 {
		t.Errorf("max relative error %.3g > 1e-10", maxErr)
	}
}

func TestDefaultThetaAccuracy(t *testing.T) {
	s, p := galaxy(5000)
	want := accelerations(s, &direct.Solver{}, p)
	got := accelerations(s, &Solver{}, p)
	median, maxErr := relErrors(got, want)
	t.Logf("θ=%.2f: median relative error %.2g, max %.2g", DefaultTheta, median, maxErr)
	// Measured for monopole-only cells: θ=0.3 → 0.06 %, 0.6 → 0.25 %, 0.9 → 0.5 % median.
	if median > 4e-3 || maxErr > 6e-2 {
		t.Errorf("too inaccurate: median %.3g, max %.3g", median, maxErr)
	}
}

func TestBlackHolesAreSummedDirectly(t *testing.T) {
	// A single star next to a black hole: the tree has one leaf, the hole is not in it.
	s := sim.NewState(2)
	s.Add(sim.Body{Pos: vec.Vec3{X: 1}, Mass: 1})
	s.Add(sim.Body{Mass: 4, Kind: sim.BlackHole})
	got := accelerations(s, &Solver{}, sim.Params{})
	if got[0] != [3]float64{-4, 0, 0} || got[1] != [3]float64{1, 0, 0} {
		t.Errorf("unexpected accelerations %v", got)
	}
}

func TestCoincidentStarsDoNotRecurseForever(t *testing.T) {
	s := sim.NewState(100)
	for range 100 {
		s.Add(sim.Body{Pos: vec.Vec3{X: 1, Y: 2, Z: 3}, Mass: 1})
	}
	s.Add(sim.Body{Mass: 1})
	(&Solver{}).Accelerations(s, sim.Params{Softening: 0.01})
	for i := range s.Len() {
		if math.IsNaN(s.AccX[i]) {
			t.Fatalf("NaN acceleration for body %d", i)
		}
	}
}

// The galaxy's energy drift is dominated by close encounters with the black holes at a fixed
// time step, not by the tree. So the tree must not drift notably more than the exact solver.
func TestEnergyDriftComparableToDirect(t *testing.T) {
	drift := func(solver sim.Solver) float64 {
		s, p := galaxy(2000)
		m := sim.New(s, solver, p)
		k, u := sim.Energy(s, p.Softening)
		e0 := k + u
		for range 500 {
			m.Step()
		}
		k, u = sim.Energy(s, p.Softening)
		return math.Abs((k + u - e0) / e0)
	}
	ref, got := drift(&direct.Solver{}), drift(&Solver{})
	t.Logf("relative energy drift after 500 steps: direct %.3g, Barnes-Hut %.3g", ref, got)
	if got > 2*ref+1e-4 {
		t.Errorf("Barnes-Hut drift %.3g much larger than direct %.3g", got, ref)
	}
}

func TestParallelMatchesSerial(t *testing.T) {
	s, p := galaxy(4000)
	a := accelerations(s, &Solver{Workers: 1}, p)
	b := accelerations(s, &Solver{Workers: 8}, p)
	if !slices.Equal(a, b) {
		t.Error("parallel result differs from serial")
	}
}

func TestAllocationsDoNotGrowWithN(t *testing.T) {
	allocs := func(n int) float64 {
		s, p := galaxy(n)
		solver := &Solver{}
		solver.Accelerations(s, p) // warm up buffers
		return testing.AllocsPerRun(5, func() { solver.Accelerations(s, p) })
	}
	small, large := allocs(2000), allocs(20000)
	if large > small || large > 64 {
		t.Errorf("allocations per call: N=2000 → %v, N=20000 → %v", small, large)
	}
}

func BenchmarkAccelerations(b *testing.B) {
	for _, n := range []int{10_000, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			s, p := galaxy(n)
			solver := &Solver{}
			solver.Accelerations(s, p)
			b.ReportAllocs()
			for b.Loop() {
				solver.Accelerations(s, p)
			}
		})
	}
}

// BenchmarkAccelerationsEvolved measures a galaxy after it has developed a dense core, which is
// markedly more expensive than the initial state (t = 0: ~19 ms, t = 2.4: ~36 ms at N = 100k).
func BenchmarkAccelerationsEvolved(b *testing.B) {
	s, p := galaxy(100_000)
	m := sim.New(s, &Solver{}, p)
	for range 600 {
		m.Step()
	}
	solver := &Solver{}
	solver.Accelerations(s, p)
	b.ResetTimer()
	for b.Loop() {
		solver.Accelerations(s, p)
	}
}

func BenchmarkPotentialEnergy(b *testing.B) {
	s, p := galaxy(100_000)
	solver := &Solver{}
	for b.Loop() {
		solver.PotentialEnergy(s, p)
	}
}

func BenchmarkBuild(b *testing.B) {
	s, _ := galaxy(100_000)
	solver := &Solver{}
	solver.build(s, DefaultTheta)
	b.ReportAllocs()
	for b.Loop() {
		solver.build(s, DefaultTheta)
	}
}

func TestPotentialEnergyMatchesExact(t *testing.T) {
	s, p := galaxy(3000)
	want := sim.PotentialEnergyParallel(s, p) // includes Paczyński-Wiita star–hole pairs (c = 10)
	if got := (&Solver{Theta: 1e-9}).PotentialEnergy(s, p); math.Abs((got-want)/want) > 1e-10 {
		t.Errorf("tiny θ: U = %v, want %v", got, want)
	}
	got := (&Solver{}).PotentialEnergy(s, p)
	rel := math.Abs((got - want) / want)
	t.Logf("diagnostics θ=%.1f: relative potential energy error %.2g", diagTheta, rel)
	if rel > 1e-3 {
		t.Errorf("default θ: U = %v, want %v (rel. error %.3g)", got, want, rel)
	}
}
