package mac

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"galaxy/internal/app"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/direct"
)

func testController(t *testing.T) *app.Controller {
	t.Helper()
	sp := scene.Presets[0].Spec
	s, p := sp.Build(200)
	r := sim.NewRunner(sim.New(s, &direct.Solver{}, p))
	r.SetPaused(true)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go r.Run(ctx)
	c := app.New(r, 200, sp)
	shell.mu.Lock()
	shell.c, shell.hooks, shell.pending = c, Hooks{}, nil
	shell.mu.Unlock()
	t.Cleanup(func() { shell.c = nil })
	return c
}

func TestMenuActionGoesThroughTheQueue(t *testing.T) {
	c := testController(t)
	glow := c.FX.Glow
	onMenu(int(app.ActionToggleGlow))
	if c.FX.Glow != glow {
		t.Fatal("menu callbacks must not touch the controller directly")
	}
	c.Drain()
	if c.FX.Glow == glow {
		t.Error("action not executed after Drain")
	}
}

func TestPanelResultsOpenSceneAndState(t *testing.T) {
	c := testController(t)
	dir := t.TempDir()

	// Scene from JSON: the built-in "Star cluster" file.
	data, err := json.Marshal(scene.Presets[2].Spec)
	if err != nil {
		t.Fatal(err)
	}
	js := filepath.Join(dir, "cluster.json")
	if err := os.WriteFile(js, data, 0o644); err != nil {
		t.Fatal(err)
	}
	onPanel(int(CmdOpenScene), js)
	c.Drain()
	if c.Scene.Name != scene.Presets[2].Spec.Name {
		t.Errorf("scene %q not opened", c.Scene.Name)
	}

	// Save through the panel, then open the file as if double-clicked in the Finder.
	state := filepath.Join(dir, "s.galaxy")
	onPanel(int(CmdSaveState), state)
	c.Drain()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(state); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("state not saved")
		}
		time.Sleep(10 * time.Millisecond)
	}
	c.LoadScene(scene.Presets[0].Spec)
	onOpenFile(state)
	c.Drain()
	if c.Scene.Name != scene.Presets[2].Spec.Name {
		t.Errorf("state not opened: scene %q", c.Scene.Name)
	}

	// Cancelled panels and broken files leave the scene alone.
	onPanel(int(CmdOpenScene), "")
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{"), 0o644)
	onPanel(int(CmdOpenScene), bad)
	c.Drain()
	if c.Scene.Name != scene.Presets[2].Spec.Name {
		t.Error("a broken scene must not replace the running one")
	}
}

func TestFilesOpenedBeforeInstallAreKept(t *testing.T) {
	shell.mu.Lock()
	shell.c, shell.pending = nil, nil
	shell.mu.Unlock()
	onOpenFile("/tmp/x.galaxy")
	if len(shell.pending) != 1 {
		t.Errorf("pending = %v", shell.pending)
	}
	shell.pending = nil
}

func TestExportGoesToHook(t *testing.T) {
	testController(t)
	var got string
	shell.hooks.ExportImage = func(p string) { got = p }
	onPanel(int(CmdExportImage), "/tmp/a.png")
	shell.c.Drain()
	if got != "/tmp/a.png" {
		t.Errorf("export hook got %q", got)
	}
}
