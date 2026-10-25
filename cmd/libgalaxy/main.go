// Command libgalaxy is the simulation as a C library for the SwiftUI app (T-049):
//
//	go build -buildmode=c-archive -o libgalaxy.a ./cmd/libgalaxy
//
// writes libgalaxy.a and libgalaxy.h. All functions must be called from one thread (the app's main
// thread). Strings returned by the library are freed with galaxy_free.
package main

/*
#include <stdlib.h>
#include <stdint.h>

typedef struct {
	int bodies, stars;
	uint64_t version;
	double time, c, meanStarMass, youngAge;
	double camX, camY, camZ, yaw, pitch, distance, fov;
} GalaxyFrameInfo;

typedef struct {
	int paused, speed, gravity, darkOff, darkBodies, gasBodies, blackHoles, bodies, stars;
	int tool, level, explain, scene, driftQuality, substeps, hydroSubsteps, gpu, showHUD, showHelp, starsFormed;
	int fx; // bits: 1 glow, 2 doppler, 4 lens, 8 trails, 16 dark, 32 gas, 64 dust
	double c, mass, theta, thetaUser, sps, fps, time, yearsPer, drift, renderScale;
	double cloudK, cloudTurb; // next gas cloud: temperature (K), turbulence (fraction of √(GM/R))
	double cloudAlpha, cloudFloorK; // its virial parameter (< 1 collapses, > 2 disperses), effective temperature
	double galaxyTilt;      // next galaxy: tilt of its disk against the launch plane (degrees)
	double massMin;         // smallest sensible mass (about 1 solar mass), simulation units
	int galaxyScale;        // galaxies can be placed in this scene
	int resolved;           // the next object is enlarged to the scene's resolution (2 × softening)
} GalaxyState;

typedef struct {
	double exposure, emNorm, dustKappa;
	float darkAlpha, gasAlpha;
	int holes; // number written to the hole array (at most its capacity)
	int worldHoles; // number written to the world hole array: x, y, z, r_s each
	float popRadius, sfDensity, sfLight; // particles as stellar populations (engine.Population)
} GalaxyLook;

typedef struct {
	int stars;    // fixed stars per scene, 0 = from the profile
	int gpu;      // 0 automatic, 1 on, 2 off
	int capacity; // stars for 30 steps/s on this machine
} GalaxySettings;

typedef struct {
	int active, plane, hasUnits, binary, holeAttractor;
	double height, azimuth, elevation, speed; // length and velocity units, radians
	double circular, escape, radius, impact;  // references at the position
	double holeKick;                          // binary: speed the attractor hole gets
	double kmPerS, pcPerUnit, msunPerUnit;    // physical units of the scene (if hasUnits)
} GalaxyLaunch;

typedef struct {
	float x, y, r, physR, rings; // render pixels; rings: 1 if the ISCO ring is drawn
} GalaxyHole;
*/
import "C"

import (
	"math"
	"strings"
	"unsafe"

	"galaxy/internal/app"
	"galaxy/internal/engine"
	"galaxy/internal/i18n"
	"galaxy/internal/learn"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/view"
)

func main() {}

var eng *engine.Engine

//export galaxy_init
func galaxy_init(preset, stars C.int, lang *C.char) {
	if lang != nil {
		i18n.Set(i18n.FromLocale(C.GoString(lang)))
	}
	if eng != nil {
		eng.Close()
	}
	eng = engine.New(int(preset), int(stars))
	eng.ToggleFullscreen = func() { fullscreen = true }
}

// galaxy_start runs the app with its command line (arguments separated by the unit separator
// \x1f, without the program name); it returns NULL or an error message.
//
//export galaxy_start
func galaxy_start(args *C.char) *C.char {
	var list []string
	if a := C.GoString(args); a != "" {
		list = strings.Split(a, "\x1f")
	}
	o, err := engine.ParseOptions(list)
	if err == nil {
		if eng != nil {
			eng.Close()
		}
		eng, err = engine.Start(o)
	}
	if err != nil {
		return C.CString(err.Error())
	}
	eng.ToggleFullscreen = func() { fullscreen = true }
	return nil
}

// galaxy_shot returns the screenshot path (empty for the interactive app) and the delay.
//
//export galaxy_shot
func galaxy_shot(after *C.double) *C.char {
	o := eng.Options()
	*after = C.double(o.ShotAfter)
	return C.CString(o.Shot)
}

// galaxy_shot_line is the summary printed with a screenshot.
//
//export galaxy_shot_line
func galaxy_shot_line(path *C.char, fps C.double) *C.char {
	return C.CString(eng.ShotLine(C.GoString(path), float64(fps)))
}

//export galaxy_tick
func galaxy_tick(fps, deviceScale C.double, thermal, lowPower C.int) {
	eng.Tick(float64(fps), float64(deviceScale), int(thermal), lowPower != 0)
}

//export galaxy_render_scale
func galaxy_render_scale() C.double { return C.double(eng.RenderScale()) }

//export galaxy_viewport
func galaxy_viewport(w, h C.double) { eng.SetViewport(float64(w), float64(h)) }

//export galaxy_frame
func galaxy_frame(buf *C.float, capacity C.int, force C.int, info *C.GalaxyFrameInfo) C.int {
	dst := unsafe.Slice((*float32)(unsafe.Pointer(buf)), int(capacity)*engine.FloatsPerBody)
	f, ok := eng.Frame(dst, force != 0)
	*info = C.GalaxyFrameInfo{
		bodies: C.int(f.Bodies), stars: C.int(f.Stars), version: C.uint64_t(f.Version),
		time: C.double(f.Time), c: C.double(f.C), meanStarMass: C.double(f.MeanStarMass), youngAge: C.double(f.YoungAge),
		camX: C.double(f.CamX), camY: C.double(f.CamY), camZ: C.double(f.CamZ),
		yaw: C.double(f.Yaw), pitch: C.double(f.Pitch), distance: C.double(f.Distance), fov: C.double(f.FOV),
	}
	if ok {
		return 1
	}
	return 0
}

func b(v bool) C.int {
	if v {
		return 1
	}
	return 0
}

func fxBits(fx view.Effects) C.int {
	return b(fx.Glow) | b(fx.Doppler)<<1 | b(fx.Lensing)<<2 | b(fx.Trails)<<3 | b(fx.Dark)<<4 | b(fx.Gas)<<5 | b(fx.Dust)<<6
}

//export galaxy_state
func galaxy_state(out *C.GalaxyState) {
	s := eng.State()
	alpha, floorK := eng.CloudForecast()
	massMin := 0.01
	if mu, ok := eng.Scene.MassUnit(); ok {
		massMin = math.Max(app.MassMin, 1/mu)
	}
	*out = C.GalaxyState{
		paused: b(s.Paused), speed: C.int(s.Speed), gravity: C.int(s.Gravity), darkOff: b(s.DarkMatterOff),
		darkBodies: C.int(s.Sim.Dark), gasBodies: C.int(s.Sim.Gas), blackHoles: C.int(s.Sim.BlackHoles),
		bodies: C.int(s.Sim.Bodies), stars: C.int(s.Stars), tool: C.int(s.Tool), level: C.int(s.Level),
		explain: b(s.Explain), scene: C.int(s.Scene), driftQuality: C.int(s.DriftQuality),
		substeps: C.int(s.Sim.Substeps), hydroSubsteps: C.int(s.Sim.HydroSubsteps), gpu: b(s.GPU),
		showHUD: b(s.ShowHUD), showHelp: b(s.ShowHelp), starsFormed: C.int(s.Sim.StarsFormed), fx: fxBits(s.FX),
		c: C.double(s.C), mass: C.double(s.Mass), theta: C.double(s.Theta), thetaUser: C.double(s.ThetaUser),
		sps: C.double(s.StepsPerSec), fps: C.double(s.FPS), time: C.double(s.Sim.Time),
		yearsPer: C.double(s.YearsPer), drift: C.double(s.Drift), renderScale: C.double(s.RenderScale),
		cloudK: C.double(s.CloudK), cloudTurb: C.double(s.CloudTurb),
		cloudAlpha: C.double(alpha), cloudFloorK: C.double(floorK), galaxyTilt: C.double(eng.GalaxyTilt),
		massMin: C.double(massMin), galaxyScale: b(eng.GalaxyScale()), resolved: b(s.Resolved),
	}
}

// actions maps the names the Swift app uses to actions.
var actions = map[string]app.Action{
	"pause": app.ActionTogglePause, "step": app.ActionStep, "faster": app.ActionFaster, "slower": app.ActionSlower,
	"lightFaster": app.ActionLightFaster, "lightSlower": app.ActionLightSlower, "cycleGravity": app.ActionCycleGravity,
	"darkMatter": app.ActionToggleDarkMatter, "toolHole": app.ActionToolBlackHole, "toolCluster": app.ActionToolCluster, "toolCloud": app.ActionToolGasCloud, "toolGalaxy": app.ActionToolGalaxy,
	"massUp": app.ActionMassUp, "massDown": app.ActionMassDown, "glow": app.ActionToggleGlow,
	"trails": app.ActionToggleTrails, "lens": app.ActionToggleLens, "doppler": app.ActionToggleDoppler,
	"dark": app.ActionToggleDark, "gas": app.ActionToggleGas, "dust": app.ActionToggleDust,
	"center": app.ActionCenterView, "explain": app.ActionToggleExplain, "nextTopic": app.ActionNextTopic,
	"prevTopic": app.ActionPrevTopic, "nextLevel": app.ActionNextLevel, "prevLevel": app.ActionPrevLevel,
	"restart": app.ActionRestart, "quickSave": app.ActionQuickSave, "quickLoad": app.ActionQuickLoad,
	"hud": app.ActionToggleHUD, "help": app.ActionToggleHelp,
}

//export galaxy_action
func galaxy_action(name *C.char) C.int {
	n := C.GoString(name)
	if i, ok := strings.CutPrefix(n, "scene"); ok {
		var k int
		for _, r := range i {
			k = 10*k + int(r-'0')
		}
		eng.Do(app.SceneAction(k))
		return 1
	}
	a, ok := actions[n]
	if !ok {
		return 0
	}
	eng.Do(a)
	return 1
}

//export galaxy_set
func galaxy_set(name *C.char, v C.double) {
	x := float64(v)
	switch C.GoString(name) {
	case "c":
		eng.SetC(x)
	case "speed":
		eng.SetSpeed(int(x))
	case "mass":
		eng.SetMass(x)
	case "theta":
		eng.SetTheta(x)
	case "gravity":
		eng.SetGravity(sim.GravityMode(max(0, min(2, int(x)))))
	case "tool":
		eng.SetTool(app.Tool(max(0, min(3, int(x)))))
	case "galaxyTilt":
		eng.GalaxyTilt = math.Max(0, math.Min(180, x))
	case "cloudK":
		eng.SetCloud(x, eng.CloudTurb)
	case "cloudTurb":
		eng.SetCloud(eng.CloudK, x)
	case "darkOff":
		eng.SetDarkMatterOff(x != 0)
	case "fx":
		k := int(x)
		eng.FX = view.Effects{Glow: k&1 != 0, Doppler: k&2 != 0, Lensing: k&4 != 0, Trails: k&8 != 0,
			Dark: k&16 != 0, Gas: k&32 != 0, Dust: k&64 != 0}
	case "level":
		eng.SetLevel(learn.Level(max(0, min(int(learn.Physics), int(x)))))
	case "topic":
		eng.TopicIdx = max(0, min(len(eng.Topics)-1, int(x)))
	case "explain":
		eng.Explain = x != 0
	case "paused":
		eng.SetPaused(x != 0)
	}
}

//export galaxy_set_profile
func galaxy_set_profile(id *C.char) { eng.SetProfile(C.GoString(id)) }

//export galaxy_rotate
func galaxy_rotate(dx, dy C.double) { eng.Rotate(float64(dx), float64(dy)) }

//export galaxy_pan
func galaxy_pan(dx, dy C.double) { eng.Pan(float64(dx), float64(dy)) }

//export galaxy_zoom
func galaxy_zoom(f C.double) { eng.Zoom(float64(f)) }

// galaxy_text returns a text (freed with galaxy_free): presets, sceneName, sceneInfo, message,
// topics (titles, one per line), title and body of the focused explanation, hardware, profile.
//
//export galaxy_text
func galaxy_text(what *C.char) *C.char {
	var s string
	switch C.GoString(what) {
	case "presets":
		names := make([]string, len(scene.Presets))
		for i, p := range scene.Presets {
			names[i] = p.Name
		}
		s = strings.Join(names, "\n")
	case "sceneName":
		s = eng.Scene.Name
	case "sceneInfo":
		s = eng.Scene.Info
	case "message":
		s, _ = eng.Message()
	case "topics":
		titles := make([]string, len(eng.Topics))
		for i, id := range eng.Topics {
			if id == app.IntroTopic {
				titles[i] = i18n.T("Einführung: ", "Introduction: ") + eng.Scene.Name
			} else {
				titles[i] = learn.Get(id).Title
			}
		}
		s = strings.Join(titles, "\n")
	case "title":
		s, _ = eng.Explanation()
	case "body":
		_, s = eng.Explanation()
	case "hardware":
		s = eng.Hardware()
	case "lang":
		s = "en"
		if i18n.Current() == i18n.German {
			s = "de"
		}
	}
	return C.CString(s)
}

// galaxy_open opens a scene (.json) or saved state; it returns NULL or an error message.
//
//export galaxy_open
func galaxy_open(path *C.char) *C.char {
	p := C.GoString(path)
	if strings.HasSuffix(strings.ToLower(p), ".json") {
		sp, err := scene.LoadSpec(p)
		if err != nil {
			return C.CString(err.Error())
		}
		eng.LoadScene(sp)
		return nil
	}
	if err := eng.Load(p); err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export galaxy_save
func galaxy_save(path *C.char) { eng.Save(C.GoString(path)) }

//export galaxy_topic_index
func galaxy_topic_index() C.int { return C.int(eng.TopicIdx) }

//export galaxy_free
func galaxy_free(s *C.char) { C.free(unsafe.Pointer(s)) }

// galaxy_key runs the action bound to a typed character (app.ActionForChar);
// it returns 0 if the character is not bound.
//
//export galaxy_key
func galaxy_key(r C.int) C.int {
	a, ok := app.ActionForChar(rune(r))
	if !ok {
		return 0
	}
	eng.Do(a)
	return 1
}

// galaxy_fullscreen_requested reports and clears a pending full screen toggle (key f).
//
//export galaxy_fullscreen_requested
func galaxy_fullscreen_requested() C.int {
	r := fullscreen
	fullscreen = false
	return b(r)
}

var fullscreen bool

// galaxy_prepare computes the look of the bodies last copied into buf by galaxy_frame (see
// engine.Prepare) for a w×h viewport in render pixels at pixel scale ps with effect bits fx. With
// dust it writes the transmission of every body (r, g, b, 1) to attn (4 floats per body).
//
//export galaxy_prepare
func galaxy_prepare(buf *C.float, attnBuf *C.float, capacity C.int, w, h, ps C.double, fx C.int, out *C.GalaxyLook, holes *C.GalaxyHole, maxHoles C.int, world *C.float, maxWorld C.int) {
	src := unsafe.Slice((*float32)(unsafe.Pointer(buf)), int(capacity)*engine.FloatsPerBody)
	var attn []float32
	if attnBuf != nil {
		attn = unsafe.Slice((*float32)(unsafe.Pointer(attnBuf)), int(capacity)*4)
	}
	k := int(fx)
	effects := view.Effects{Glow: k&1 != 0, Doppler: k&2 != 0, Lensing: k&4 != 0, Trails: k&8 != 0,
		Dark: k&16 != 0, Gas: k&32 != 0, Dust: k&64 != 0}
	look := eng.Prepare(src, attn, eng.LastFrame(), float64(w), float64(h), float64(ps), effects)
	dst := unsafe.Slice(holes, int(maxHoles))
	n := min(len(look.Holes), len(dst))
	for i := range n {
		hl := look.Holes[i]
		rings := C.float(0)
		if hl.Rings {
			rings = 1
		}
		dst[i] = C.GalaxyHole{x: C.float(hl.X), y: C.float(hl.Y), r: C.float(hl.R), physR: C.float(hl.PhysR), rings: rings}
	}
	wd := unsafe.Slice((*float32)(unsafe.Pointer(world)), int(maxWorld)*4)
	nw := min(len(look.World), int(maxWorld))
	for i := range nw {
		copy(wd[4*i:4*i+4], look.World[i][:])
	}
	*out = C.GalaxyLook{worldHoles: C.int(nw), exposure: C.double(look.Exposure), emNorm: C.double(look.EmNorm), dustKappa: C.double(look.DustKappa),
		darkAlpha: C.float(look.DarkAlpha), gasAlpha: C.float(look.GasAlpha), holes: C.int(n),
		popRadius: C.float(look.Pop.Radius), sfDensity: C.float(look.Pop.SFDensity), sfLight: C.float(look.Pop.SFLight)}
}

//export galaxy_settings
func galaxy_settings(out *C.GalaxySettings) {
	p := eng.Settings()
	gpu := 0
	for i, m := range engine.GPUModes {
		if m == p.GPU {
			gpu = i
		}
	}
	*out = C.GalaxySettings{stars: C.int(p.Stars), gpu: C.int(gpu), capacity: C.int(eng.Capacity())}
}

//export galaxy_set_stars
func galaxy_set_stars(n C.int) { eng.SetStars(int(n)) }

// galaxy_set_gpu sets 0 automatic, 1 on, 2 off.
//
//export galaxy_set_gpu
func galaxy_set_gpu(mode C.int) {
	eng.SetGPUMode(engine.GPUModes[max(0, min(len(engine.GPUModes)-1, int(mode)))])
}

//export galaxy_recalibrate
func galaxy_recalibrate() { eng.Recalibrate() }

// galaxy_launch_begin starts planning a launch at (x, y) in render pixels (engine.Launch).
//
//export galaxy_launch_begin
func galaxy_launch_begin(x, y C.double) { eng.BeginLaunch(float64(x), float64(y)) }

// galaxy_launch_set changes plane, height, azimuth, elevation or speed (simulation units, radians).
//
//export galaxy_launch_set
func galaxy_launch_set(name *C.char, v C.double) { eng.SetLaunch(C.GoString(name), float64(v)) }

// galaxy_launch_preset: 0 rest, 1 circular orbit, 2 escape, 3 towards the center, 4 binary.
//
//export galaxy_launch_preset
func galaxy_launch_preset(p C.int) { eng.LaunchPreset(int(p)) }

// galaxy_launch_drag sets direction and speed from a drag to (x1, y1).
//
//export galaxy_launch_drag
func galaxy_launch_drag(x1, y1 C.double) { eng.DragLaunch(float64(x1), float64(y1)) }

//export galaxy_launch_go
func galaxy_launch_go() { eng.GoLaunch() }

//export galaxy_launch_cancel
func galaxy_launch_cancel() { eng.CancelLaunch() }

//export galaxy_launch
func galaxy_launch(out *C.GalaxyLaunch) {
	l := eng.LaunchPlan()
	v, pc, ok := eng.Scene.Units()
	msun, _ := eng.Scene.MassUnit()
	*out = C.GalaxyLaunch{active: b(l.Active), plane: C.int(l.Plane), hasUnits: b(ok), binary: b(l.Binary), holeAttractor: b(l.HoleMass > 0),
		height: C.double(l.Height), azimuth: C.double(l.Azimuth), elevation: C.double(l.Elevation),
		speed: C.double(l.Speed), circular: C.double(l.Circular), escape: C.double(l.Escape),
		radius: C.double(l.Radius), impact: C.double(l.Impact), holeKick: C.double(eng.HoleKick()), kmPerS: C.double(v), pcPerUnit: C.double(pc),
		msunPerUnit: C.double(msun)}
}

// galaxy_gizmo writes the lines (x0, y0, x1, y1, color) and markers (x, y, kind) of the launch pad
// for the current view, in render pixels; it returns the number of lines and sets *nMarks.
//
//export galaxy_gizmo
func galaxy_gizmo(segOut *C.float, maxSegs C.int, markOut *C.float, maxMarks C.int, nMarks *C.int) C.int {
	gizmoSegs, gizmoMarks = eng.Gizmo(gizmoSegs, gizmoMarks)
	dst := unsafe.Slice((*float32)(unsafe.Pointer(segOut)), int(maxSegs)*5)
	n := min(len(gizmoSegs), int(maxSegs))
	for i, sg := range gizmoSegs[:n] {
		copy(dst[5*i:5*i+5], []float32{sg.X0, sg.Y0, sg.X1, sg.Y1, float32(sg.Color)})
	}
	md := unsafe.Slice((*float32)(unsafe.Pointer(markOut)), int(maxMarks)*3)
	m := min(len(gizmoMarks), int(maxMarks))
	for i, mk := range gizmoMarks[:m] {
		copy(md[3*i:3*i+3], []float32{mk.X, mk.Y, float32(mk.Kind)})
	}
	*nMarks = C.int(m)
	return C.int(n)
}

var (
	gizmoSegs  []engine.Segment
	gizmoMarks []engine.Marker
)

// galaxy_launch_hit returns the handle of the launch pad at (x, y), -1 for none (engine.Handle).
//
//export galaxy_launch_hit
func galaxy_launch_hit(x, y C.double) C.int { return C.int(eng.HitLaunch(float64(x), float64(y))) }

//export galaxy_launch_drag_begin
func galaxy_launch_drag_begin(h C.int, x, y C.double) {
	eng.BeginDrag(engine.Handle(h), float64(x), float64(y))
}

// galaxy_launch_drag_to moves the dragged handle; snap 1 snaps to reference speeds and axes.
//
//export galaxy_launch_drag_to
func galaxy_launch_drag_to(x, y C.double, snap C.int) { eng.DragTo(float64(x), float64(y), snap != 0) }

// galaxy_launch_drag_end ends the drag; it returns 1 if a slingshot was released (then launch).
//
//export galaxy_launch_drag_end
func galaxy_launch_drag_end() C.int { return b(eng.EndDrag()) }
