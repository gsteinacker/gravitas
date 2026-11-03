// Package prefs stores the user's settings as JSON in the per-user config directory
// (~/Library/Application Support/Gravitas/prefs.json on macOS).
package prefs

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// AppName is the directory of the app's files in the per-user config directory.
const AppName = "Gravitas"

// legacyName is the directory before the app was renamed from Galaxy to Gravitas (T-092).
const legacyName = "Galaxy"

// GPU modes.
const (
	GPUAuto = "auto" // use Metal if available (and, with calibration, if faster)
	GPUOn   = "on"
	GPUOff  = "off"
)

// Prefs are the persistent settings. Zero values mean "automatic" or "default".
type Prefs struct {
	Profile   string `json:"profile"`   // performance profile, see internal/perf; "" = auto
	Stars     int    `json:"stars"`     // stars per scene; 0 = from the profile
	GPU       string `json:"gpu"`       // GPUAuto, GPUOn, GPUOff
	Level     int    `json:"level"`     // audience of the learning mode (learn.Level)
	Inspector *bool  `json:"inspector"` // inspector visible; nil = default (shown)

	// Calibration results per hardware (T-046), keyed by the hardware description.
	Calibration map[string]Calibration `json:"calibration,omitempty"`
}

// Calibration is the measured speed of one machine.
type Calibration struct {
	UseGPU        bool    `json:"useGPU"`
	SecPerNL      float64 `json:"secPerNLogN"`    // seconds per M·log2(M) of one force evaluation of M bodies
	BodiesPerStar float64 `json:"bodiesPerStar"`  // bodies (stars, dark matter, gas) per star of a scene
	SecPerGasNL   float64 `json:"secPerGasNLogN"` // seconds per G·log2(G) of one SPH evaluation of G gas bodies
}

// Default returns the settings of a first start.
func Default() *Prefs { return &Prefs{GPU: GPUAuto} }

// Dir is the app's directory for settings and saved states, or "" if the system has no config
// directory. On first use it takes over the directory of the app's old name.
func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return dirIn(base)
}

// dirIn returns AppName in base. If it does not exist yet, it moves base/Galaxy there, but only if
// that holds the files of this app (another app could have used the generic name).
func dirIn(base string) string {
	dir := filepath.Join(base, AppName)
	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		return dir
	}
	old := filepath.Join(base, legacyName)
	if ours(old) {
		_ = os.Rename(old, dir) // if it fails, the app starts with defaults
	}
	return dir
}

// ours reports whether dir holds the settings or saved states of this app.
func ours(dir string) bool {
	for _, name := range []string{"prefs.json", "saves"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// Path is the settings file, or "" if the system has no config directory.
func Path() string {
	dir := Dir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "prefs.json")
}

// Load reads the settings; a missing or broken file gives the defaults.
func Load(path string) *Prefs {
	p := Default()
	if path == "" {
		return p
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return p
	}
	if err := json.Unmarshal(data, p); err != nil {
		return Default()
	}
	switch p.GPU {
	case GPUAuto, GPUOn, GPUOff:
	default:
		p.GPU = GPUAuto
	}
	p.Stars = max(0, p.Stars)
	return p
}

// Save writes the settings atomically.
func (p *Prefs) Save(path string) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// InspectorVisible reports whether the inspector should be shown.
func (p *Prefs) InspectorVisible() bool { return p.Inspector == nil || *p.Inspector }

// SetInspectorVisible remembers the inspector visibility.
func (p *Prefs) SetInspectorVisible(v bool) { p.Inspector = &v }
