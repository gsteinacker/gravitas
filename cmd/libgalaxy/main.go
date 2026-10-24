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
	int tool, level, explain, scene, driftQuality, substeps, hydroSubsteps, gpu, showHUD;
	int fx; // bits: 1 glow, 2 doppler, 4 lens, 8 trails, 16 dark, 32 gas, 64 dust
	double c, mass, theta, thetaUser, sps, fps, time, yearsPer, drift, renderScale;
} GalaxyState;
*/
import "C"

import (
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
	*out = C.GalaxyState{
		paused: b(s.Paused), speed: C.int(s.Speed), gravity: C.int(s.Gravity), darkOff: b(s.DarkMatterOff),
		darkBodies: C.int(s.Sim.Dark), gasBodies: C.int(s.Sim.Gas), blackHoles: C.int(s.Sim.BlackHoles),
		bodies: C.int(s.Sim.Bodies), stars: C.int(s.Stars), tool: C.int(s.Tool), level: C.int(s.Level),
		explain: b(s.Explain), scene: C.int(s.Scene), driftQuality: C.int(s.DriftQuality),
		substeps: C.int(s.Sim.Substeps), hydroSubsteps: C.int(s.Sim.HydroSubsteps), gpu: b(s.GPU),
		showHUD: b(s.ShowHUD), fx: fxBits(s.FX),
		c: C.double(s.C), mass: C.double(s.Mass), theta: C.double(s.Theta), thetaUser: C.double(s.ThetaUser),
		sps: C.double(s.StepsPerSec), fps: C.double(s.FPS), time: C.double(s.Sim.Time),
		yearsPer: C.double(s.YearsPer), drift: C.double(s.Drift), renderScale: C.double(s.RenderScale),
	}
}

// actions maps the names the Swift app uses to actions.
var actions = map[string]app.Action{
	"pause": app.ActionTogglePause, "step": app.ActionStep, "faster": app.ActionFaster, "slower": app.ActionSlower,
	"lightFaster": app.ActionLightFaster, "lightSlower": app.ActionLightSlower, "cycleGravity": app.ActionCycleGravity,
	"darkMatter": app.ActionToggleDarkMatter, "toolHole": app.ActionToolBlackHole, "toolCluster": app.ActionToolCluster,
	"massUp": app.ActionMassUp, "massDown": app.ActionMassDown, "glow": app.ActionToggleGlow,
	"trails": app.ActionToggleTrails, "lens": app.ActionToggleLens, "doppler": app.ActionToggleDoppler,
	"dark": app.ActionToggleDark, "gas": app.ActionToggleGas, "dust": app.ActionToggleDust,
	"center": app.ActionCenterView, "explain": app.ActionToggleExplain, "nextTopic": app.ActionNextTopic,
	"prevTopic": app.ActionPrevTopic, "nextLevel": app.ActionNextLevel, "prevLevel": app.ActionPrevLevel,
	"restart": app.ActionRestart, "quickSave": app.ActionQuickSave, "quickLoad": app.ActionQuickLoad,
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
		eng.Tool = app.Tool(max(0, min(1, int(x))))
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

//export galaxy_click
func galaxy_click(x, y C.double) { eng.Click(float64(x), float64(y)) }

//export galaxy_launch
func galaxy_launch(x0, y0, x1, y1 C.double) {
	eng.Launch(float64(x0), float64(y0), float64(x1), float64(y1))
}

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

// galaxy_key runs the action bound to a typed character (same bindings as the Ebitengine app);
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
