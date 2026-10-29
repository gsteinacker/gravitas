package perf

import (
	"math"
	"sync"
	"time"

	"galaxy/internal/prefs"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
	"galaxy/internal/sph"
)

// StepBudget is the time one simulation step may take for 30 steps/s (docs/architecture.md).
const StepBudget = time.Second / 30

// stepOverhead is the cost of a full step in an evolved galaxy relative to one force evaluation
// of the initial state: 28 ms against 13.4 ms at 100k stars (docs/architecture.md, M4 Pro).
const stepOverhead = 2.1

// Star counts the calibration may recommend.
const (
	MinStars = 5_000
	MaxStars = 1_000_000
)

// gasOverhead is the number of SPH evaluations per step: 60 ms steps of the Horsehead scene at
// 60k gas against 39.5 ms per evaluation (perf-log T-038), rounded up for Courant substeps.
const gasOverhead = 1.6

// calibrationGas is the size of the measured gas scene.
const calibrationGas = 10_000

// calibrationStars is the size of the measured scene: large enough for stable timings, small
// enough to measure in well under a second on slow machines.
const calibrationStars = 20_000

// Calibrate measures one force evaluation of the default scene on the CPU and, if allowed and
// available, on the GPU, and keeps the faster. It takes 0.1–1 s.
func Calibrate(gpuAllowed bool) prefs.Calibration {
	s, p := scene.Presets[0].Spec.Build(calibrationStars)
	measure := func(gpu bool) float64 {
		solver := &bh.Solver{GPU: gpu}
		solver.Accelerations(s, p) // warm up: buffers, caches
		best := math.Inf(1)
		for range 3 {
			t0 := time.Now()
			solver.Accelerations(s, p)
			best = min(best, time.Since(t0).Seconds())
		}
		return best
	}
	best, useGPU := measure(false), false
	if gpuAllowed && bh.GPUAvailable() == nil {
		if g := measure(true); g < best {
			best, useGPU = g, true
		}
	}
	m := float64(s.Len())
	return prefs.Calibration{
		UseGPU: useGPU, SecPerNL: best / nlogn(m), BodiesPerStar: m / calibrationStars,
		SecPerGasNL: measureGas(),
	}
}

// measureGas times one SPH evaluation of the gas scene per G·log2(G); 0 without such a scene.
func measureGas() float64 {
	var sp *scene.Spec
	for _, p := range scene.Presets {
		if f := GasFraction(p.Spec); f > 0.5 && sp == nil {
			sp = p.Spec
		}
	}
	if sp == nil {
		return 0
	}
	s, p := sp.Build(calibrationGas)
	n := s.Len()
	var out sim.HydroOut
	out.Resize(n)
	var h sph.Solver
	h.Forces(s, &p, nil, nil, &out) // converges the smoothing lengths
	best := math.Inf(1)
	for range 2 {
		t0 := time.Now()
		h.Forces(s, &p, nil, nil, &out)
		best = min(best, time.Since(t0).Seconds())
	}
	return best / nlogn(float64(countGas(s)))
}

func countGas(s *sim.State) int {
	g := 0
	for _, k := range s.Kind {
		if k == sim.Gas {
			g++
		}
	}
	return g
}

var gasFractions sync.Map // *scene.Spec → float64

// GasFraction is the share of gas bodies of a scene, measured on a small build and cached.
func GasFraction(sp *scene.Spec) float64 {
	if f, ok := gasFractions.Load(sp); ok {
		return f.(float64)
	}
	s, _ := sp.Build(2000)
	f := float64(countGas(s)) / float64(max(1, s.Len()))
	gasFractions.Store(sp, f)
	return f
}

func nlogn(m float64) float64 { return m * math.Log2(math.Max(2, m)) }

// StarsFor is the largest star count of a gas-free scene whose evolved steps fit into budget,
// rounded down to a readable value and clamped to [MinStars, MaxStars].
func StarsFor(c prefs.Calibration, budget time.Duration) int { return StarsForGas(c, budget, 0) }

// StarsForGas is StarsFor for a scene whose bodies are gas to the fraction gas.
func StarsForGas(c prefs.Calibration, budget time.Duration, gas float64) int {
	if c.SecPerNL <= 0 || c.BodiesPerStar <= 0 {
		return 0
	}
	cost := func(n float64) float64 { return StepSeconds(c, n, gas) }
	lo, hi := float64(MinStars), float64(MaxStars)
	if cost(hi) <= budget.Seconds() {
		return MaxStars
	}
	if cost(lo) >= budget.Seconds() {
		return MinStars
	}
	for hi-lo > 100 {
		mid := (lo + hi) / 2
		if cost(mid) <= budget.Seconds() {
			lo = mid
		} else {
			hi = mid
		}
	}
	return roundDown(int(lo))
}

// StepSeconds is the predicted time of one evolved step of a scene with the given stars and gas
// fraction.
func StepSeconds(c prefs.Calibration, stars, gas float64) float64 {
	gasCost := c.SecPerGasNL
	if gasCost <= 0 {
		gasCost = 4 * c.SecPerNL // uncalibrated: SPH costs a few gravity evaluations
	}
	m := c.BodiesPerStar * stars
	return stepOverhead*c.SecPerNL*nlogn(m) + gasOverhead*gasCost*nlogn(gas*m)
}

// roundDown keeps two significant digits: 87 341 → 87 000, 312 000 → 310 000.
func roundDown(n int) int {
	unit := 1
	for n/unit >= 100 {
		unit *= 10
	}
	return max(MinStars, n/unit*unit)
}
