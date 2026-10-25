package engine

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"galaxy/internal/app"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/view"
)

func TestFrameCopiesBodies(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // keep the user's settings out of the test
	e := New(0, 2000)
	defer e.Close()
	buf := make([]float32, 10000*FloatsPerBody)
	deadline := time.Now().Add(5 * time.Second)
	for {
		e.Tick(60, 1, 0, false)
		if info, ok := e.Frame(buf, false); ok && info.Bodies > 0 {
			if info.Stars == 0 || info.MeanStarMass <= 0 || info.Distance <= 0 {
				t.Errorf("frame info %+v", info)
			}
			e.Runner.SetPaused(true)
			e.Frame(buf, true)
			first, _ := e.Frame(buf, true)
			again, copied := e.Frame(buf, false)
			if copied {
				t.Error("an unchanged snapshot must not be copied again")
			}
			// The renderer keeps drawing the last copy: its description must stay valid (a zero
			// mean star mass made every star maximally bright between steps).
			if again.MeanStarMass != first.MeanStarMass || again.Bodies != first.Bodies || again.C != first.C {
				t.Errorf("info without copy %+v, after copy %+v", again, first)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no frame")
		}
		time.Sleep(5 * time.Millisecond)
	}
	// A buffer smaller than the scene is filled, not overrun.
	small := make([]float32, 10*FloatsPerBody)
	if info, _ := e.Frame(small, true); info.Bodies != 10 {
		t.Errorf("small buffer: %d bodies", info.Bodies)
	}
}

func TestActionsAndGovernorRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := New(0, 1000)
	defer e.Close()
	e.Do(app.ActionToggleTrails)
	for range 40 {
		e.Tick(60, 2, 0, false)
	}
	if !e.State().FX.Trails {
		t.Error("action not published")
	}
	if s := e.RenderScale(); s != 2 {
		t.Errorf("render scale %v on a Retina display", s)
	}
}

// TestNoEbitengine: the engine is for front ends without Ebitengine.
func TestNoEbitengine(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "galaxy/internal/engine").Output()
	if err != nil {
		t.Skip("go list unavailable:", err)
	}
	for _, bad := range []string{"hajimehoshi/ebiten", "galaxy/internal/mac", "galaxy/internal/ui"} {
		if strings.Contains(string(out), bad) {
			t.Errorf("engine depends on %s", bad)
		}
	}
}

func TestPrepareAdaptsAndCaches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := New(0, 3000)
	defer e.Close()
	e.Runner.SetPaused(true)
	buf := make([]float32, 10000*FloatsPerBody)
	var info FrameInfo
	deadline := time.Now().Add(5 * time.Second)
	for info.Bodies == 0 && time.Now().Before(deadline) {
		e.Tick(60, 1, 0, false)
		info, _ = e.Frame(buf, true)
		time.Sleep(5 * time.Millisecond)
	}
	fx := view.Effects{Glow: true, Dark: true}
	var look Look
	for range 30 { // the exposure converges over a few recomputations
		look = e.Prepare(buf, nil, info, 1440, 900, 1, fx)
	}
	if look.Exposure <= 0 || look.Exposure > 1 || look.DarkAlpha <= 0 || len(look.Holes) != 1 {
		t.Fatalf("look %+v", look)
	}
	h := look.Holes[0]
	if h.X < 0 || h.X > 1440 || h.Y < 0 || h.Y > 900 || h.R < minHolePixels {
		t.Errorf("hole %+v", h)
	}
	if e.looker.adapting {
		t.Error("exposure did not converge")
	}
	// Cached: same input, same result, no recomputation.
	before := e.looker.look.Exposure
	e.looker.look.Exposure = 42
	if got := e.Prepare(buf, nil, info, 1440, 900, 1, fx); got.Exposure != 42 {
		t.Error("unchanged input must use the cache")
	}
	e.looker.look.Exposure = before
	e.Zoom(0.5)
	if got := e.Prepare(buf, nil, info, 1440, 900, 1, fx); got.Exposure == 42 {
		t.Error("camera change must recompute")
	}
	if got := e.Prepare(buf, nil, info, 1440, 900, 1, view.Effects{}); got.DarkAlpha != 0 {
		t.Error("hidden dark matter needs no haze")
	}
}

func TestDustAttenuatesStarsBehindGas(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := New(0, 20000) // spiral galaxy with a gas disk
	defer e.Close()
	e.Runner.SetPaused(true)
	e.Cam.Pitch = 0.15 // nearly edge-on: much gas in front of stars
	buf := make([]float32, 40000*FloatsPerBody)
	var info FrameInfo
	deadline := time.Now().Add(5 * time.Second)
	for info.Bodies == 0 && time.Now().Before(deadline) {
		e.Tick(60, 1, 0, false)
		info, _ = e.Frame(buf, true)
		time.Sleep(5 * time.Millisecond)
	}
	attn := make([]float32, 4*info.Bodies)
	look := e.Prepare(buf, attn, info, 1440, 900, 1, view.Effects{Dust: true})
	if look.DustKappa <= 0 {
		t.Fatalf("no dust opacity: %+v", look)
	}
	dimmed, reddened := 0, 0
	for i := range info.Bodies {
		if sim.Kind(buf[i*FloatsPerBody+7]+0.5) != sim.Star {
			continue
		}
		r, g, b := attn[4*i], attn[4*i+1], attn[4*i+2]
		if r < 0 || r > 1 || b > g || g > r {
			t.Fatalf("star %d: transmission %v %v %v (blue must be absorbed most)", i, r, g, b)
		}
		if g < 0.5 {
			dimmed++
		}
		if b < 0.9*r {
			reddened++
		}
	}
	if dimmed == 0 || reddened == 0 {
		t.Errorf("edge-on disk: %d stars dimmed, %d reddened", dimmed, reddened)
	}
	// Cached frames still fill the buffer (the front end may pass another one).
	other := make([]float32, len(attn))
	e.Prepare(buf, other, info, 1440, 900, 1, view.Effects{Dust: true})
	for i := range attn {
		if other[i] != attn[i] {
			t.Fatal("cached attenuation not copied")
		}
	}
}

func TestStarSettingsAndCorrection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := New(0, 0) // stars from the profile
	defer e.Close()
	e.PrefsPath = "" // settings are saved in the background; nothing to write here
	e.SetViewport(1440, 900)
	auto := e.stars(e.Scene)
	e.SetStars(25_000)
	if e.N != 25_000 || e.Settings().Stars != 25_000 {
		t.Errorf("fixed stars: N %d, settings %d", e.N, e.Settings().Stars)
	}
	e.SetStars(0)
	if e.N != auto {
		t.Errorf("back to the profile: %d, want %d", e.N, auto)
	}
	// A scene that ran twice as slow as predicted gets fewer stars next time.
	e.correction = map[*scene.Spec]float64{e.Scene: 2}
	if slow := e.stars(e.Scene); slow >= auto && auto < 1_000_000 {
		t.Errorf("corrected stars %d, uncorrected %d", slow, auto)
	}
	if s := e.LaunchSpeed(100, 100, 100, 100); s != 0 {
		t.Errorf("zero drag gives speed %v", s)
	}
}
