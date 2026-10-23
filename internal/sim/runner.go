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
	Kind           []Kind
	Time           float64
	Steps          int
	StepsPerSecond float64
}

func (sn *Snapshot) Len() int { return len(sn.Mass) }

func (sn *Snapshot) fill(m *Simulation) {
	s := m.State
	n := s.Len()
	sn.X, sn.Y, sn.Z, sn.Mass = sn.X[:0], sn.Y[:0], sn.Z[:0], sn.Mass[:0]
	sn.Kind = append(sn.Kind[:0], s.Kind...)
	for i := range n {
		sn.X = append(sn.X, float32(s.PosX[i]))
		sn.Y = append(sn.Y, float32(s.PosY[i]))
		sn.Z = append(sn.Z, float32(s.PosZ[i]))
		sn.Mass = append(sn.Mass, float32(s.Mass[i]))
	}
	sn.Time, sn.Steps = m.Time, m.Steps
}

// Runner steps a Simulation in its own goroutine, decoupled from rendering.
// All mutations of the simulation go through Do, so they happen between steps.
type Runner struct {
	sim  *Simulation
	cmds chan func(*Simulation)

	mu          sync.Mutex
	front, back *Snapshot

	paused    atomic.Bool
	targetSPS atomic.Int64 // 0 = as fast as possible
	sps       atomic.Uint64
}

func NewRunner(m *Simulation) *Runner {
	r := &Runner{sim: m, cmds: make(chan func(*Simulation), 64), front: &Snapshot{}, back: &Snapshot{}}
	r.front.fill(m)
	return r
}

// Do schedules f to run on the simulation goroutine before the next step.
func (r *Runner) Do(f func(*Simulation)) { r.cmds <- f }

// View calls f with the latest snapshot. f must not retain it.
func (r *Runner) View(f func(*Snapshot)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f(r.front)
}

func (r *Runner) Paused() bool            { return r.paused.Load() }
func (r *Runner) SetPaused(p bool)        { r.paused.Store(p) }
func (r *Runner) TargetSPS() int          { return int(r.targetSPS.Load()) }
func (r *Runner) SetTargetSPS(sps int)    { r.targetSPS.Store(int64(sps)) }
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

		r.sim.Step()
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
	r.back.fill(r.sim)
	r.mu.Lock()
	r.front, r.back = r.back, r.front
	r.mu.Unlock()
}
