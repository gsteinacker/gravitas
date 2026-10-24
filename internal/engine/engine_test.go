package engine

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"galaxy/internal/app"
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
