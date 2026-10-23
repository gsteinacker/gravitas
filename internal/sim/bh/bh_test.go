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

// galaxy returns a fixed, clustered test system independent of the scene files: a disk around a
// heavy central black hole plus a second black hole (the former "Spiral galaxy" scene).
func galaxy(n int) (*sim.State, sim.Params) {
	s := sim.NewState(n + 2)
	scene.AddDisk(s, scene.DiskConfig{
		N: n, StarMass: 1, BHMass: 2, ScaleRadius: 0.8, MinRadius: 0.15, MaxRadius: 4,
		Thickness: 0.03, Softening: 0.06, Dispersion: 0.12, Seed: 1,
	})
	s.Add(sim.Body{Pos: vec.Vec3{X: 1.5, Y: 0.3}, Mass: 0.5, Kind: sim.BlackHole})
	return s, sim.Params{Dt: 0.004, Softening: 0.06, C: 10}
}

// realisticGalaxy is the dark matter dominated "Spiral galaxy" scene.
func realisticGalaxy(n int) (*sim.State, sim.Params) { return scene.Presets[0].Build(n) }

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

// withHolePull adds the (exactly summed) pull of all holes to each star's acceleration, so errors
// are measured relative to the total force a star feels, as before the force split (T-025).
func withHolePull(s *sim.State, p sim.Params, acc [][3]float64) [][3]float64 {
	eps2 := p.Softening * p.Softening
	for i := range acc {
		for h := range s.Len() {
			if s.Kind[h] != sim.BlackHole || s.Kind[i] != sim.Star {
				continue
			}
			dx, dy, dz := s.PosX[h]-s.PosX[i], s.PosY[h]-s.PosY[i], s.PosZ[h]-s.PosZ[i]
			f := sim.PairForce(s.Mass[h], dx*dx+dy*dy+dz*dz, eps2, sim.SchwarzschildRadius(s.Mass[h], p.C))
			acc[i][0], acc[i][1], acc[i][2] = acc[i][0]+f*dx, acc[i][1]+f*dy, acc[i][2]+f*dz
		}
	}
	return acc
}

func starsOnly(s *sim.State, acc [][3]float64) [][3]float64 {
	var out [][3]float64
	for i, a := range acc {
		if s.Kind[i] == sim.Star {
			out = append(out, a)
		}
	}
	return out
}

func TestDefaultThetaAccuracy(t *testing.T) {
	s, p := galaxy(5000)
	want := withHolePull(s, p, accelerations(s, &direct.Solver{}, p))
	got := withHolePull(s, p, accelerations(s, &Solver{}, p))
	want, got = starsOnly(s, want), starsOnly(s, got) // holes get no solver force
	median, maxErr := relErrors(got, want)
	t.Logf("θ=%.2f: median relative error %.2g, max %.2g", DefaultTheta, median, maxErr)
	// Measured for monopole-only cells: θ=0.3 → 0.06 %, 0.6 → 0.25 %, 0.9 → 0.5 % median.
	if median > 4e-3 || maxErr > 6e-2 {
		t.Errorf("too inaccurate: median %.3g, max %.3g", median, maxErr)
	}
}

func TestSolverLeavesHolesToTheIntegrator(t *testing.T) {
	// Two stars and a hole: the solver only sees the star pair; the hole gets nothing from it.
	s := sim.NewState(3)
	s.Add(sim.Body{Pos: vec.Vec3{X: 1}, Mass: 1})
	s.Add(sim.Body{Pos: vec.Vec3{X: 2}, Mass: 2})
	s.Add(sim.Body{Mass: 4, Kind: sim.BlackHole})
	got := accelerations(s, &Solver{}, sim.Params{})
	if got[0] != [3]float64{2, 0, 0} || got[1] != [3]float64{-1, 0, 0} || got[2] != [3]float64{} {
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
		e0 := modelEnergy(m)
		for range 500 {
			m.Step()
		}
		return math.Abs((modelEnergy(m) - e0) / e0)
	}
	ref, got := drift(&direct.Solver{}), drift(&Solver{})
	t.Logf("relative energy drift after 500 steps: direct %.3g, Barnes-Hut %.3g", ref, got)
	// Before multiple time stepping near holes (T-025) this was 0.3 with c = 10.
	if ref > 5e-3 {
		t.Errorf("direct solver drift %.3g: close encounters with holes are under-resolved", ref)
	}
	if got > 2*ref+1e-4 {
		t.Errorf("Barnes-Hut drift %.3g much larger than direct %.3g", got, ref)
	}
}

// modelEnergy is the exact energy of the full model (relativistic kinetic, pair potential) minus
// the energy changes booked for accretion, i.e. what integration must conserve.
func modelEnergy(m *sim.Simulation) float64 {
	return sim.KineticEnergy(m.State, m.Params.C) + sim.PotentialEnergyParallel(m.State, m.Params) - m.ExternalEnergy
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

// BenchmarkStep measures a full integration step (tree forces, hole forces with substeps) of a
// 100k galaxy with two holes.
func BenchmarkStep(b *testing.B) {
	s, p := galaxy(100_000)
	m := sim.New(s, &Solver{}, p)
	m.Step()
	for b.Loop() {
		m.Step()
	}
	b.ReportMetric(float64(m.Substeps), "substeps")
}

func BenchmarkBuild(b *testing.B) {
	for _, n := range []int{100_000, 1_000_000} {
		for _, workers := range []int{1, 0} {
			b.Run(fmt.Sprintf("N=%d/workers=%d", n, workers), func(b *testing.B) {
				s, _ := galaxy(n)
				solver := &Solver{Workers: workers}
				solver.build(s, &sim.Params{}, DefaultTheta)
				b.ReportAllocs()
				for b.Loop() {
					solver.build(s, &sim.Params{}, DefaultTheta)
				}
			})
		}
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

// farCluster: a clumpy cluster of radius ~1 around (10, 3, -2) and a probe star at the origin.
func farCluster() *sim.State {
	rng := rand.New(rand.NewPCG(5, 6))
	s := sim.NewState(401)
	s.Add(sim.Body{Mass: 1e-9})
	for i := range 400 {
		off := vec.Vec3{X: 0.5 * float64(i%2), Y: 0.3 * float64(i%3)}
		s.Add(sim.Body{
			Pos:  vec.Vec3{X: 10 + rng.NormFloat64()*0.4, Y: 3 + rng.NormFloat64()*0.2, Z: -2 + rng.NormFloat64()*0.3}.Add(off),
			Mass: 0.5 + rng.Float64(),
		})
	}
	return s
}

func TestQuadrupoleImprovesFarField(t *testing.T) {
	s := farCluster()
	p := sim.Params{Softening: 1e-3}
	want := accelerations(s, &direct.Solver{}, p)[0]
	_, wantU := sim.Energy(s, p.Softening)
	errOf := func(solver *Solver) (float64, float64) {
		got := accelerations(s, solver, p)[0]
		d := math.Sqrt((got[0]-want[0])*(got[0]-want[0]) + (got[1]-want[1])*(got[1]-want[1]) + (got[2]-want[2])*(got[2]-want[2]))
		w := math.Sqrt(want[0]*want[0] + want[1]*want[1] + want[2]*want[2])
		// Probe star's potential: U(all) − U(cluster alone) is dominated by the cluster–cluster
		// part, so compare the probe's own ψ via the full tree potential instead.
		u := solver.PotentialEnergy(s, p)
		return d / w, math.Abs((u - wantU) / wantU)
	}
	// θ = 0.95 lets the probe accept the whole cluster (and its big subnodes) as single nodes.
	mono, _ := errOf(&Solver{Theta: 0.95, MonopoleOnly: true})
	quad, _ := errOf(&Solver{Theta: 0.95})
	t.Logf("far-field force error: monopole %.2g, quadrupole %.2g", mono, quad)
	if quad > mono/5 {
		t.Errorf("quadrupole error %.3g should be well below monopole error %.3g", quad, mono)
	}
}

func TestQuadrupoleForceIsGradientOfPotential(t *testing.T) {
	// Probe the tree field (root accepted as a single node) numerically.
	s := farCluster()
	p := sim.Params{Softening: 1e-3}
	b := &Solver{Theta: 0.95}
	b.build(s, &p, 0.95)
	eps2 := p.Softening * p.Softening
	const h = 1e-6
	x, y, z := 0.0, 0.5, 0.2
	ax, ay, az := b.walk(x, y, z, -1, eps2)
	// a = ∇ψ (ψ = −Φ).
	gx := (b.walkPotential(x+h, y, z, -1, eps2) - b.walkPotential(x-h, y, z, -1, eps2)) / (2 * h)
	gy := (b.walkPotential(x, y+h, z, -1, eps2) - b.walkPotential(x, y-h, z, -1, eps2)) / (2 * h)
	gz := (b.walkPotential(x, y, z+h, -1, eps2) - b.walkPotential(x, y, z-h, -1, eps2)) / (2 * h)
	for _, c := range [][2]float64{{ax, gx}, {ay, gy}, {az, gz}} {
		if math.Abs(c[0]-c[1]) > 1e-6*math.Abs(ax) {
			t.Errorf("force %v vs gradient %v", c[0], c[1])
		}
	}
}

func TestParallelRadixSortIsStableAndCorrect(t *testing.T) {
	rng := rand.New(rand.NewPCG(8, 9))
	n := 200_000
	keys, idx := make([]uint64, n), make([]int32, n)
	for i := range keys {
		keys[i], idx[i] = rng.Uint64N(1<<20)<<40, int32(i) // many duplicates, low bits constant
	}
	want, wantIdx := slices.Clone(keys), slices.Clone(idx)
	radixSort(want, wantIdx, make([]uint64, n), make([]int32, n))
	var hist [][256]int
	radixSortParallel(keys, idx, make([]uint64, n), make([]int32, n), 7, &hist)
	if !slices.Equal(keys, want) || !slices.Equal(idx, wantIdx) {
		t.Fatal("parallel radix sort differs from the serial (stable) one")
	}
}

func TestBuildIndependentOfWorkers(t *testing.T) {
	s, p := galaxy(60_000) // above parallelMin
	a := accelerations(s, &Solver{Workers: 1}, p)
	b := accelerations(s, &Solver{Workers: 12}, p)
	if !slices.Equal(a, b) {
		t.Error("result depends on the number of workers")
	}
}

func TestParallelBuildIsBitIdentical(t *testing.T) {
	for _, n := range []int{40_000, 300_000} {
		s, p := galaxy(n)
		serial, parallel := &Solver{Workers: 1}, &Solver{Workers: 12}
		a, b := accelerations(s, serial, p), accelerations(s, parallel, p)
		if !slices.Equal(a, b) {
			t.Fatalf("N=%d: parallel build changes the forces", n)
		}
		if len(serial.nodes) != len(parallel.nodes) {
			t.Fatalf("N=%d: %d vs %d nodes", n, len(serial.nodes), len(parallel.nodes))
		}
		// The potential sums chunk results in completion order, so only round-off may differ.
		if us, up := serial.PotentialEnergy(s, p), parallel.PotentialEnergy(s, p); math.Abs((us-up)/us) > 1e-12 {
			t.Fatalf("N=%d: potential %v vs %v", n, us, up)
		}
	}
}
