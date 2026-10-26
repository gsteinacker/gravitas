// Package engine runs the simulation and the app controller without any user interface, for front
// ends in other languages (the SwiftUI app, T-049/T-050). The front end calls every method from one
// thread (its main thread), renders the bodies Frame copies out and feeds back the measured frame
// rate, from which the governor adapts quality (perf.Governor).
package engine

import (
	"context"
	"log"
	"math"
	"time"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/perf"
	"galaxy/internal/prefs"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
	"galaxy/internal/sph"
)

// defaultStars is the star count of uncalibrated machines.
const defaultStars = 50000

// Engine is one running simulation with its controller.
type Engine struct {
	*app.Controller
	cancel     context.CancelFunc
	settings   *prefs.Prefs
	cal        prefs.Calibration
	hw         perf.Hardware
	profile    perf.Profile
	gov        *perf.Governor
	scale      float64 // render pixels per point, chosen by the governor
	ticks      int
	spsCap     int
	version    uint64
	last       FrameInfo // snapshot part of the last copy, returned again while nothing changed
	looker     looker
	opts       *Options
	launch     Launch
	launchDone bool
	drag       drag

	correction map[*scene.Spec]float64 // measured/predicted step time per scene (observe)
	running    *scene.Spec
	since      time.Time
}

// New starts built-in scene preset (0-based) with stars bodies; stars ≤ 0 chooses them from the
// performance profile and the calibration of this machine (measured once, cached in the settings).
func New(preset, stars int) *Engine {
	e, err := Start(&Options{Preset: preset + 1, Stars: stars, Gravity: "instant", Theta: bh.DefaultTheta, GPU: true})
	if err != nil {
		panic(err) // built-in scenes
	}
	return e
}

// Start runs the app with its command line options. Screenshots (o.Shot without o.Adapt) ignore the
// user's settings and keep stars, render scale and θ fixed, so measurements are reproducible.
func Start(o *Options) (*Engine, error) {
	if o.Lang != "" {
		i18n.Set(i18n.FromLocale(o.Lang)) // before the scene: its messages are translated
	}
	fixed := o.Screenshot() && !o.Adapt
	e := &Engine{settings: prefs.Load(prefs.Path()), hw: perf.Probe(), opts: o, drag: drag{handle: HandleNone}}
	if fixed {
		e.settings = prefs.Default()
	}
	e.profile = perf.ProfileByID(e.settings.Profile)
	key := e.hw.Key()
	if c, ok := e.settings.Calibration[key]; ok && c.SecPerNL > 0 {
		e.cal = c
	} else {
		e.cal = perf.Calibrate(e.settings.GPU != prefs.GPUOff && (o.GPU || !o.GPUSet))
		if e.settings.Calibration == nil {
			e.settings.Calibration = map[string]prefs.Calibration{}
		}
		e.settings.Calibration[key] = e.cal
	}
	sp, err := o.spec()
	if err != nil {
		return nil, err
	}
	stars := o.Stars
	auto := stars <= 0
	if auto {
		stars = e.stars(sp)
	}
	s, p := sp.Build(stars)
	if err := o.params(&p); err != nil {
		return nil, err
	}
	gpu := e.gpu()
	if o.GPUSet {
		gpu = o.GPU && bh.GPUAvailable() == nil
	}
	m := sim.New(s, &bh.Solver{Theta: o.Theta, GPU: gpu}, p)
	m.Hydro = &sph.Solver{}
	r := sim.NewRunner(m)
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	go r.Run(ctx)

	e.Controller = app.New(r, stars, sp)
	e.Theta, e.ThetaUser, e.GPU = o.Theta, o.Theta, gpu
	e.ApplyPrefs(e.settings)
	if !fixed {
		e.PrefsPath = prefs.Path()
	}
	if auto {
		e.StarsFor = e.stars
	}
	if err := o.apply(e); err != nil {
		cancel()
		return nil, err
	}
	if e.PrefsPath != "" {
		if err := e.settings.Save(e.PrefsPath); err != nil { // synchronously: keep the calibration
			log.Printf("engine: settings: %v", err)
		}
	}
	e.Publish()
	log.Printf("engine: %s, %d stars, GPU %v", e.hw, stars, gpu)
	return e, nil
}

// Options returns the start options.
func (e *Engine) Options() *Options { return e.opts }

// Close stops the simulation.
func (e *Engine) Close() { e.cancel() }

func (e *Engine) stars(sp *scene.Spec) int {
	if e.settings.Stars > 0 {
		return e.settings.Stars
	}
	corr := 1.0
	if c, ok := e.correction[sp]; ok {
		corr = c
	}
	return e.profile.SceneStars(e.cal, sp, corr, defaultStars)
}

func (e *Engine) gpu() bool {
	switch e.settings.GPU {
	case prefs.GPUOff:
		return false
	case prefs.GPUOn:
		return bh.GPUAvailable() == nil
	}
	return e.cal.UseGPU && bh.GPUAvailable() == nil
}

// Hardware describes the machine and its measured capacity.
func (e *Engine) Hardware() string { return e.hw.String() }

// SetProfile switches the performance profile ("" = automatic); it applies to the next scene.
func (e *Engine) SetProfile(id string) {
	e.settings.Profile = id
	e.profile = perf.ProfileByID(id)
	e.gov = nil
	e.N = e.stars(e.Scene)
	e.SavePrefs()
}

// Tick runs once per displayed frame: it executes posted work, adapts quality about twice per
// second and publishes the state. fps is the measured frame rate, deviceScale the pixels per point
// of the display, thermal the macOS thermal state (0–3).
func (e *Engine) Tick(fps, deviceScale float64, thermal int, lowPower bool) {
	e.Drain()
	e.FPS = fps
	if fx, fy, preset, launch, ok := e.opts.pendingLaunch(); ok && e.Cam.W > 0 && !e.launchDone {
		e.launchDone = true // -launch: once the view has a size
		e.BeginLaunch(fx*e.Cam.W, fy*e.Cam.H)
		if preset >= 0 {
			e.LaunchPreset(preset)
		}
		if launch {
			e.GoLaunch()
		}
	}
	if o := e.opts; o.Scale > 0 && (!o.Screenshot() || !o.Adapt) {
		// A fixed render scale (-scale, screenshots) also keeps θ and the step rate fixed.
		e.scale, e.RenderScaleNow = o.Scale, o.Scale
		e.Runner.SetTargetSPS(e.SpeedLimit())
		e.Publish()
		return
	}
	if e.gov == nil {
		e.gov = perf.NewGovernor(e.profile.MaxRenderScale, deviceScale, e.ThetaUser)
		e.scale = e.gov.Current().RenderScale
	}
	e.ticks++
	if e.ticks%30 == 1 {
		sps := e.Runner.StepsPerSecond()
		if e.Runner.Paused() {
			sps = 0
		}
		e.gov.ThetaBase = e.ThetaUser
		set, _ := e.gov.Tick(perf.Inputs{
			FPS: fps, StepsPerSec: sps, TargetSPS: e.profile.TargetSPS, UserLimit: e.SpeedLimit(),
			Thermal: thermal, LowPower: lowPower,
		})
		e.scale = set.RenderScale
		if set.Theta != e.Theta {
			e.ApplyTheta(set.Theta)
		}
		e.spsCap = set.SPSCap
		e.RenderScaleNow = e.scale
		e.observe()
	}
	limit := e.SpeedLimit()
	if e.spsCap > 0 && (limit == 0 || e.spsCap < limit) {
		limit = e.spsCap
	}
	e.Runner.SetTargetSPS(limit)
	e.Publish()
}

// RenderScale is the number of render pixels per point the governor wants.
func (e *Engine) RenderScale() float64 {
	if e.scale <= 0 {
		return 1
	}
	return e.scale
}

// SetViewport sets the drawable size in render pixels.
func (e *Engine) SetViewport(w, h float64) { e.Cam.W, e.Cam.H = w, h }

// Rotate, Pan and Zoom move the camera; deltas in render pixels.
func (e *Engine) Rotate(dx, dy float64) { e.Cam.Rotate(dx/e.RenderScale(), dy/e.RenderScale()) }
func (e *Engine) Pan(dx, dy float64)    { e.Cam.Pan(dx, dy) }
func (e *Engine) Zoom(f float64)        { e.Cam.Zoom(f) }

// LastFrame describes the bodies of the last Frame copy.
func (e *Engine) LastFrame() FrameInfo { return e.last }

// FloatsPerBody is the layout of Frame: x, y, z, mass, vx, vy, vz, kind, age, ion, h, rho.
const FloatsPerBody = 12

// FrameInfo describes the copied bodies.
type FrameInfo struct {
	Bodies, Stars        int
	Version              uint64
	Time, C              float64
	MeanStarMass         float64
	YoungAge             float64
	CamX, CamY, CamZ     float64 // camera target
	Yaw, Pitch, Distance float64
	FOV                  float64
}

// Frame copies the latest snapshot into dst (FloatsPerBody floats per body) if it is newer than
// the last copy or force is set; it returns false if nothing was copied. Bodies beyond the
// capacity of dst are dropped; Info.Bodies says how many were copied. Without a copy, the info
// describes the last copied snapshot (the caller keeps drawing it) with the current camera.
func (e *Engine) Frame(dst []float32, force bool) (FrameInfo, bool) {
	info := e.last
	copied := false
	e.Runner.View(func(sn *sim.Snapshot) {
		e.Observe(sn)
		if sn.Version == e.version && !force {
			return
		}
		e.version = sn.Version
		info = FrameInfo{}
		n := min(sn.Len(), len(dst)/FloatsPerBody)
		var starMass float64
		for i := range n {
			d := dst[i*FloatsPerBody : i*FloatsPerBody+FloatsPerBody : i*FloatsPerBody+FloatsPerBody]
			d[0], d[1], d[2], d[3] = sn.X[i], sn.Y[i], sn.Z[i], sn.Mass[i]
			d[4], d[5], d[6], d[7] = sn.VX[i], sn.VY[i], sn.VZ[i], float32(sn.Kind[i])
			d[8], d[9], d[10], d[11] = sn.Age[i], sn.GasIon[i], sn.GasH[i], sn.GasRho[i]
			if sn.Kind[i] == sim.Star {
				// Stars have no gas: slot 9 carries their identity (24 bits, exact in a float), from
				// which the renderer draws the stars of a population (Galaxy.metal population).
				d[9] = float32(sn.ID[i] & 0xffffff)
				starMass += float64(sn.Mass[i])
				info.Stars++
			}
		}
		for _, h := range sn.Spins { // holes carry spin J and accretion rate in the gas slots (engine.Prepare)
			if h.Index < n {
				d := dst[h.Index*FloatsPerBody:]
				d[8], d[9], d[10], d[11] = float32(h.J.X), float32(h.J.Y), float32(h.J.Z), float32(h.Mdot)
			}
		}
		info.Bodies, info.Version, info.Time, info.C, info.YoungAge = n, sn.Version, sn.Time, sn.C, sn.YoungAge
		info.MeanStarMass = starMass / math.Max(1, float64(info.Stars))
		e.last = info
		copied = true
	})
	k := e.Cam
	info.CamX, info.CamY, info.CamZ = k.Target.X, k.Target.Y, k.Target.Z
	info.Yaw, info.Pitch, info.Distance, info.FOV = k.Yaw, k.Pitch, k.Distance, k.FOV
	return info, copied
}
