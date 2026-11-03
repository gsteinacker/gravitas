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

func TestDirTakesOverTheOldDirectory(t *testing.T) {
	base := t.TempDir()
	old := filepath.Join(base, legacyName)
	if err := Default().Save(filepath.Join(old, "prefs.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(old, "saves"), 0o755); err != nil {
		t.Fatal(err)
	}
	dir := dirIn(base)
	if dir != filepath.Join(base, AppName) {
		t.Fatalf("dir = %q", dir)
	}
	for _, name := range []string{"prefs.json", "saves"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s not taken over: %v", name, err)
		}
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("old directory still there: %v", err)
	}
}

func TestDirLeavesForeignAndNewerDirectoriesAlone(t *testing.T) {
	base := t.TempDir()
	foreign := filepath.Join(base, legacyName, "other.db") // another app called Galaxy
	if err := os.MkdirAll(filepath.Dir(foreign), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(foreign, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	dirIn(base)
	if _, err := os.Stat(foreign); err != nil {
		t.Errorf("foreign directory moved: %v", err)
	}

	base = t.TempDir()
	for _, d := range []string{legacyName, AppName} {
		if err := Default().Save(filepath.Join(base, d, "prefs.json")); err != nil {
			t.Fatal(err)
		}
	}
	dirIn(base)
	if _, err := os.Stat(filepath.Join(base, legacyName, "prefs.json")); err != nil {
		t.Errorf("old directory moved over an existing new one: %v", err)
	}
}
