package sim

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// Snapshot is a render-friendly copy of the state, published after every step.
type Snapshot struct {
	X, Y, Z, Mass  []float32
	VX, VY, VZ     []float32 // coordinate velocities v = u/γ
	Kind           []Kind
	Time           float64
	Steps          int
	C              float64 // speed of light of the run, 0 = Newtonian
	Substeps       int     // inner substeps near black holes in the last step
	StepsPerSecond float64
}

func (sn *Snapshot) Len() int { return len(sn.Mass) }

func (sn *Snapshot) fill(m *Simulation) {
	s := m.State
	n := s.Len()
	sn.X, sn.Y, sn.Z, sn.Mass = sn.X[:0], sn.Y[:0], sn.Z[:0], sn.Mass[:0]
	sn.VX, sn.VY, sn.VZ = sn.VX[:0], sn.VY[:0], sn.VZ[:0]
	sn.Kind = append(sn.Kind[:0], s.Kind...)
	for i := range n {
		sn.X = append(sn.X, float32(s.PosX[i]))
		sn.Y = append(sn.Y, float32(s.PosY[i]))
		sn.Z = append(sn.Z, float32(s.PosZ[i]))
		sn.Mass = append(sn.Mass, float32(s.Mass[i]))
		g := gamma(s.VelX[i], s.VelY[i], s.VelZ[i], m.Params.C)
		sn.VX = append(sn.VX, float32(s.VelX[i]/g))
		sn.VY = append(sn.VY, float32(s.VelY[i]/g))
		sn.VZ = append(sn.VZ, float32(s.VelZ[i]/g))
	}
	sn.Time, sn.Steps, sn.C, sn.Substeps = m.Time, m.Steps, m.Params.C, m.Substeps
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
}

// energyInterval is how often the (comparatively expensive) total energy is measured.
const energyInterval = 2 * time.Second

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
		if time.Since(r.lastMeasure) > energyInterval {
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
	r.lastMeasure = time.Now()
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
	r.writing.fill(r.sim)
	r.mu.Lock()
	r.latest, r.writing, r.fresh = r.writing, r.latest, true
	r.mu.Unlock()
}
