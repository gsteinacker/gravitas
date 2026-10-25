package engine

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"galaxy/internal/app"
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
		look = e.Prepare(buf, info, 1440, 900, 1, fx)
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
	if got := e.Prepare(buf, info, 1440, 900, 1, fx); got.Exposure != 42 {
		t.Error("unchanged input must use the cache")
	}
	e.looker.look.Exposure = before
	e.Zoom(0.5)
	if got := e.Prepare(buf, info, 1440, 900, 1, fx); got.Exposure == 42 {
		t.Error("camera change must recompute")
	}
	if got := e.Prepare(buf, info, 1440, 900, 1, view.Effects{}); got.DarkAlpha != 0 {
		t.Error("hidden dark matter needs no haze")
	}
}
