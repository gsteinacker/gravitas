package sim

import (
	"galaxy/internal/vec"

	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// Snapshot is a render-friendly copy of the state, published after every step.
type Snapshot struct {
	X, Y, Z, Mass  []float32
	VX, VY, VZ     []float32  // coordinate velocities v = u/γ
	Age            []float32  // stars born from gas: time since birth; -1 for initial stars
	GasU, GasRho   []float32  // gas: specific internal energy and density (0 otherwise)
	GasH           []float32  // gas: smoothing length
	GasIon         []float32  // gas: ionized fraction
	Spins          []HoleSpin // angular momentum of every black hole
	YoungAge       float64    // stars younger than this are drawn young
	StarsFormed    int
	FormedMass     float64 // mass of the stars formed from gas
	Supernovae     int     // populations exploded (supernova.go)
	SNMass         float64 // their mass
	SNDelayMin     float64 // explosion delays after birth (SupernovaDelay); both 0 without supernovae
	SNDelayMax     float64
	HolesFormed    int     // black holes formed by collapse (collapse.go)
	ScaleFactor    float64 // a in the comoving mode (comoving.go), 0 otherwise
	Kind           []Kind
	ID             []uint64 // stable identity of every body (State.ID)
	Time           float64
	Steps          int
	C              float64 // speed of light of the run, 0 = Newtonian
	Substeps       int     // inner substeps near black holes in the last step
	HydroSubsteps  int     // parts of the last step for the Courant condition of gas
	Version        uint64  // increases with every publish, so consumers can skip unchanged data
	Gravity        GravityMode
	DarkMatterOff  bool
	StepsPerSecond float64
}

func (sn *Snapshot) Len() int { return len(sn.Mass) }

// HoleSpin is the angular momentum J and the recent accretion rate Mdot (mass per time) of the
// black hole with body index Index.
type HoleSpin struct {
	Index int
	J     vec.Vec3
	Mdot  float64
}

func (sn *Snapshot) fill(m *Simulation) {
	s := m.State
	n := s.Len()
	sn.X, sn.Y, sn.Z, sn.Mass = resize(sn.X, n), resize(sn.Y, n), resize(sn.Z, n), resize(sn.Mass, n)
	sn.VX, sn.VY, sn.VZ = resize(sn.VX, n), resize(sn.VY, n), resize(sn.VZ, n)
	sn.Age, sn.GasU, sn.GasRho = resize(sn.Age, n), resize(sn.GasU, n), resize(sn.GasRho, n)
	sn.GasH, sn.GasIon = resize(sn.GasH, n), resize(sn.GasIon, n)
	sn.YoungAge, sn.StarsFormed, sn.HolesFormed = m.Params.Gas.YoungAge, m.StarsFormed, m.HolesFormed
	sn.FormedMass, sn.Supernovae, sn.SNMass = m.FormedMass, m.Supernovae, m.SNMass
	sn.SNDelayMin, sn.SNDelayMax = 0, 0
	if g := m.Params.Gas; g.SNEnergy > 0 {
		sn.SNDelayMin, sn.SNDelayMax = g.SNDelayMin, g.SNDelayMax
	}
	sn.ScaleFactor = m.ScaleFactor
	if sn.ScaleFactor == 0 && m.Params.Cosmology != nil {
		sn.ScaleFactor = m.Params.Cosmology.AStart
	}
	if sn.YoungAge == 0 {
		sn.YoungAge = defaultYoungAge
	}
	now := m.Time
	sn.Kind = append(sn.Kind[:0], s.Kind...)
	sn.ID = append(sn.ID[:0], s.ID...)
	c := m.Params.C
	ParallelFor(n, 8192, 0, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			sn.X[i], sn.Y[i], sn.Z[i] = float32(s.PosX[i]), float32(s.PosY[i]), float32(s.PosZ[i])
			sn.Mass[i] = float32(s.Mass[i])
			g := gamma(s.VelX[i], s.VelY[i], s.VelZ[i], c)
			sn.VX[i], sn.VY[i], sn.VZ[i] = float32(s.VelX[i]/g), float32(s.VelY[i]/g), float32(s.VelZ[i]/g)
			sn.Age[i] = -1
			if !math.IsInf(s.Born[i], -1) {
				sn.Age[i] = float32(now - s.Born[i])
			}
			sn.GasU[i], sn.GasRho[i], sn.GasH[i] = float32(s.U[i]), float32(s.Rho[i]), float32(s.H[i])
			sn.GasIon[i] = float32(s.Ion[i])
		}
	})
	sn.Spins = sn.Spins[:0]
	for i, k := range s.Kind {
		if k == BlackHole {
			sn.Spins = append(sn.Spins, HoleSpin{Index: i, J: s.Spin(i), Mdot: m.AccretionRate(s.ID[i])})
		}
	}
	sn.Time, sn.Steps, sn.C, sn.Substeps, sn.Gravity = m.Time, m.Steps, m.Params.C, m.Substeps, m.Params.Gravity
	sn.HydroSubsteps = m.HydroSubsteps
	sn.DarkMatterOff = m.Params.DarkMatterOff
}

// Runner steps a Simulation in its own goroutine, decoupled from rendering.
// All mutations of the simulation go through Do, so they happen between steps.
type Runner struct {
	sim  *Simulation
	cmds chan func(*Simulation)

	// Triple buffer: the simulation fills writing, the renderer reads reading, and latest is the
	// newest complete snapshot. The mutex only guards pointer swaps, so neither side ever waits
	// for the other's work (rendering used to block publishing, see T-024).
	mu                       sync.Mutex
	writing, latest, reading *Snapshot
	fresh                    bool // latest is newer than reading

	paused    atomic.Bool
	targetSPS atomic.Int64 // 0 = as fast as possible
	sps       atomic.Uint64
	drift     atomic.Uint64 // float64 bits of the relative energy drift, NaN if unknown
	quality   atomic.Int32  // EnergyQuality of the last drift value

	// Energy diagnostics, owned by the simulation goroutine.
	epoch       int
	e0, ext0    float64 // baseline energy and ExternalEnergy at that moment
	lastMeasure time.Time
	version     uint64
	measureCost time.Duration // duration of the last energy measurement
}

// energyInterval is the minimum time between two (comparatively expensive) energy measurements.
// energyBudget caps their share of the run: at 1M stars one measurement takes ~0.2–0.3 s, and a
// fixed 2 s interval let them dominate the CPU (T-023).
const (
	energyInterval = 2 * time.Second
	energyBudget   = 0.05
)

func NewRunner(m *Simulation) *Runner {
	r := &Runner{sim: m, cmds: make(chan func(*Simulation), 64), writing: &Snapshot{}, latest: &Snapshot{}, reading: &Snapshot{}}
	r.reading.fill(m)
	r.epoch = m.Epoch - 1 // forces a baseline measurement on the first step
	r.drift.Store(math.Float64bits(math.NaN()))
	return r
}

// Do schedules f to run on the simulation goroutine before the next step.
func (r *Runner) Do(f func(*Simulation)) { r.cmds <- f }

// View calls f with the newest published snapshot. f must not retain it. View must only be
// called from one goroutine (the renderer); the simulation keeps stepping while f runs.
func (r *Runner) View(f func(*Snapshot)) {
	r.mu.Lock()
	if r.fresh {
		r.reading, r.latest, r.fresh = r.latest, r.reading, false
	}
	r.mu.Unlock()
	f(r.reading)
}

func (r *Runner) Paused() bool         { return r.paused.Load() }
func (r *Runner) SetPaused(p bool)     { r.paused.Store(p) }
func (r *Runner) TargetSPS() int       { return int(r.targetSPS.Load()) }
func (r *Runner) SetTargetSPS(sps int) { r.targetSPS.Store(int64(sps)) }

// EnergyDrift returns (E - E0) / |E0| since the last scene change or added body (NaN if unknown)
// and how it was measured.
func (r *Runner) EnergyDrift() (float64, EnergyQuality) {
	return math.Float64frombits(r.drift.Load()), EnergyQuality(r.quality.Load())
}

func (r *Runner) StepsPerSecond() float64 { return math.Float64frombits(r.sps.Load()) }

// Run steps the simulation until ctx is cancelled.
func (r *Runner) Run(ctx context.Context) {
	var next time.Time
	windowStart, windowSteps := time.Now(), 0
	for ctx.Err() == nil {
		changed := r.drain()
		if r.paused.Load() {
			if changed {
				r.publish()
			}
			r.sps.Store(0)
			select {
			case f := <-r.cmds:
				f(r.sim)
				r.publish()
			case <-ctx.Done():
			case <-time.After(20 * time.Millisecond):
			}
			windowStart, windowSteps = time.Now(), 0
			continue
		}

		if r.sim.Epoch != r.epoch {
			r.measureEnergy() // baseline before the first step of a new epoch
		}
		r.sim.Step()
		if r.sim.Finished() { // a comoving run that reached today: stop instead of spinning
			r.paused.Store(true)
		}
		if time.Since(r.lastMeasure) > max(energyInterval, time.Duration(float64(r.measureCost)/energyBudget)) {
			r.measureEnergy()
		}
		r.publish()
		windowSteps++
		if el := time.Since(windowStart); el > 500*time.Millisecond {
			r.sps.Store(math.Float64bits(float64(windowSteps) / el.Seconds()))
			windowStart, windowSteps = time.Now(), 0
		}

		if t := r.targetSPS.Load(); t > 0 {
			now := time.Now()
			if next.Before(now.Add(-100 * time.Millisecond)) {
				next = now // fell far behind, don't try to catch up
			}
			next = next.Add(time.Second / time.Duration(t))
			if d := next.Sub(now); d > 0 {
				time.Sleep(d)
			}
		}
	}
}

func (r *Runner) measureEnergy() {
	if r.sim.Params.Cosmology != nil { // the Newtonian energy does not apply in an expanding box
		r.quality.Store(int32(EnergyUnknown))
		r.drift.Store(math.Float64bits(math.NaN()))
		return
	}
	r.lastMeasure = time.Now()
	defer func() { r.measureCost = time.Since(r.lastMeasure) }()
	e, q := TotalEnergy(r.sim)
	r.quality.Store(int32(q))
	if r.sim.Epoch != r.epoch {
		r.epoch, r.e0, r.ext0 = r.sim.Epoch, e, r.sim.ExternalEnergy
	}
	e -= r.sim.ExternalEnergy - r.ext0 // accretion changes E legitimately
	if q == EnergyUnknown || r.e0 == 0 {
		r.drift.Store(math.Float64bits(math.NaN()))
		return
	}
	r.drift.Store(math.Float64bits((e - r.e0) / math.Abs(r.e0)))
}

func (r *Runner) drain() bool {
	changed := false
	for {
		select {
		case f := <-r.cmds:
			f(r.sim)
			changed = true
		default:
			return changed
		}
	}
}

func (r *Runner) publish() {
	r.version++
	r.writing.fill(r.sim)
	r.writing.Version = r.version
	r.mu.Lock()
	r.latest, r.writing, r.fresh = r.writing, r.latest, true
	r.mu.Unlock()
}
