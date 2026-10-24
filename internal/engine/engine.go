// Package engine runs the simulation and the app controller without any user interface, for front
// ends in other languages (the SwiftUI app, T-049/T-050). The front end calls every method from one
// thread (its main thread), renders the bodies Frame copies out and feeds back the measured frame
// rate, from which the governor adapts quality like in the Ebitengine app.
package engine

import (
	"context"
	"log"
	"math"

	"galaxy/internal/app"
	"galaxy/internal/perf"
	"galaxy/internal/prefs"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
	"galaxy/internal/sph"
	"galaxy/internal/vec"
)

// defaultStars is the star count of uncalibrated machines.
const defaultStars = 50000

// Engine is one running simulation with its controller.
type Engine struct {
	*app.Controller
	cancel   context.CancelFunc
	settings *prefs.Prefs
	cal      prefs.Calibration
	hw       perf.Hardware
	profile  perf.Profile
	gov      *perf.Governor
	scale    float64 // render pixels per point, chosen by the governor
	ticks    int
	spsCap   int
	version  uint64
	last     FrameInfo // snapshot part of the last copy, returned again while nothing changed
}

// New starts built-in scene preset (0-based) with stars bodies; stars ≤ 0 chooses them from the
// performance profile and the calibration of this machine (measured once, cached in the settings).
func New(preset, stars int) *Engine {
	e := &Engine{settings: prefs.Load(prefs.Path()), hw: perf.Probe()}
	e.profile = perf.ProfileByID(e.settings.Profile)
	key := e.hw.Key()
	if c, ok := e.settings.Calibration[key]; ok && c.SecPerNL > 0 {
		e.cal = c
	} else {
		e.cal = perf.Calibrate(e.settings.GPU != prefs.GPUOff)
		if e.settings.Calibration == nil {
			e.settings.Calibration = map[string]prefs.Calibration{}
		}
		e.settings.Calibration[key] = e.cal
	}
	sp := scene.Presets[max(0, min(len(scene.Presets)-1, preset))].Spec
	auto := stars <= 0
	if auto {
		stars = e.stars(sp)
	}
	s, p := sp.Build(stars)
	gpu := e.gpu()
	theta := bh.DefaultTheta
	m := sim.New(s, &bh.Solver{Theta: theta, GPU: gpu}, p)
	m.Hydro = &sph.Solver{}
	r := sim.NewRunner(m)
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	go r.Run(ctx)

	e.Controller = app.New(r, stars, sp)
	e.Theta, e.ThetaUser, e.GPU = theta, theta, gpu
	e.ApplyPrefs(e.settings)
	e.PrefsPath = prefs.Path()
	if auto {
		e.StarsFor = e.stars
	}
	if err := e.settings.Save(e.PrefsPath); err != nil { // synchronously: keep the calibration
		log.Printf("engine: settings: %v", err)
	}
	log.Printf("engine: %s, %d stars, GPU %v", e.hw, stars, gpu)
	return e
}

// Close stops the simulation.
func (e *Engine) Close() { e.cancel() }

func (e *Engine) stars(sp *scene.Spec) int {
	if e.settings.Stars > 0 {
		return e.settings.Stars
	}
	return e.profile.SceneStars(e.cal, sp, 1, defaultStars)
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

// Click places the current tool's object at (x, y) in render pixels; Launch with the velocity of a
// drag from (x0, y0) to (x1, y1).
func (e *Engine) Click(x, y float64) { e.Place(x, y, vec.Vec3{}) }
func (e *Engine) Launch(x0, y0, x1, y1 float64) {
	e.Place(x0, y0, app.LaunchVelocity(&e.Cam, x0, y0, x1, y1))
}

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
				starMass += float64(sn.Mass[i])
				info.Stars++
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
