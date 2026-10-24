package prefs

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "prefs.json")
	p := Default()
	p.Profile, p.Stars, p.GPU, p.Level = "high", 120000, GPUOff, 2
	p.SetInspectorVisible(false)
	p.Calibration = map[string]Calibration{"M4": {UseGPU: true, SecPerNL: 1e-9, BodiesPerStar: 1.5}}
	if err := p.Save(path); err != nil {
		t.Fatal(err)
	}
	q := Load(path)
	if !reflect.DeepEqual(p, q) {
		t.Errorf("loaded %+v, want %+v", q, p)
	}
	if q.InspectorVisible() {
		t.Error("inspector visibility lost")
	}
}

func TestBrokenOrMissingFileGivesDefaults(t *testing.T) {
	dir := t.TempDir()
	if p := Load(filepath.Join(dir, "missing.json")); !reflect.DeepEqual(p, Default()) {
		t.Errorf("missing file: %+v", p)
	}
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{not json"), 0o644)
	if p := Load(bad); !reflect.DeepEqual(p, Default()) {
		t.Errorf("broken file: %+v", p)
	}
	odd := filepath.Join(dir, "odd.json")
	os.WriteFile(odd, []byte(`{"gpu":"maybe","stars":-5}`), 0o644)
	if p := Load(odd); p.GPU != GPUAuto || p.Stars != 0 {
		t.Errorf("invalid values not repaired: %+v", p)
	}
	if !Default().InspectorVisible() {
		t.Error("inspector should be shown by default")
	}
}
