package engine

import (
	"path/filepath"
	"testing"
)

func TestParseOptions(t *testing.T) {
	t.Setenv("PWD", "/work")
	o, err := ParseOptions([]string{"-NSDocumentRevisionsDebugMode", "YES", "-preset", "3", "-fx", "glow,dust",
		"-shot", "bin/x.png", "-hud", "-c=2.5", "-psn_0_12345"})
	if err != nil {
		t.Fatal(err)
	}
	if o.Preset != 3 || o.FX != "glow,dust" || !o.HUD || o.C != 2.5 {
		t.Errorf("options %+v", o)
	}
	if o.Shot != filepath.Join("/work", "bin/x.png") {
		t.Errorf("relative shot path %q not resolved against $PWD", o.Shot)
	}
	// Screenshots are reproducible: fixed stars and render scale unless -adapt.
	if o.Stars != defaultStars || o.Scale != 1 {
		t.Errorf("screenshot defaults: stars %d, scale %v", o.Stars, o.Scale)
	}
	a, _ := ParseOptions([]string{"-shot", "/tmp/y.png", "-adapt"})
	if a.Stars != 0 || a.Scale != 0 {
		t.Errorf("-adapt must keep automatic stars and scale: %+v", a)
	}
	app, _ := ParseOptions(nil)
	if app.Screenshot() || app.Stars != 0 || app.Preset != 1 || !app.GPU || app.GPUSet {
		t.Errorf("app defaults %+v", app)
	}
	g, _ := ParseOptions([]string{"-gpu=false"})
	if g.GPU || !g.GPUSet {
		t.Errorf("-gpu=false: %+v", g)
	}
	if _, err := ParseOptions([]string{"-n", "many"}); err == nil {
		t.Error("an invalid number must be an error")
	}
}

func TestStartWithOptions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	o, _ := ParseOptions([]string{"-preset", "2", "-n", "2000", "-fx", "trails", "-c", "-1", "-gravity", "retarded",
		"-explain", "physics:dust", "-shot", "/tmp/never.png"})
	e, err := Start(o)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	s := e.State()
	if !s.FX.Trails || s.FX.Glow || !s.Explain || e.Topics[e.TopicIdx] != "dust" || e.PrefsPath != "" {
		t.Errorf("state %+v, topic %q, prefs %q", s, e.Topics[e.TopicIdx], e.PrefsPath)
	}
	if e.Scene.Name != "Galaxy collision" {
		t.Errorf("scene %q", e.Scene.Name)
	}
	e.Tick(60, 2, 0, false)
	if e.RenderScale() != 1 {
		t.Errorf("screenshot render scale %v", e.RenderScale())
	}
	if _, err := Start(&Options{SceneFile: "/nonexistent.json", Gravity: "instant"}); err == nil {
		t.Error("missing scene file must be an error")
	}
}
