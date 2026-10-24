// Package prefs stores the user's settings as JSON in the per-user config directory
// (~/Library/Application Support/Galaxy/prefs.json on macOS).
package prefs

import (
	"encoding/json"
	"os"
	"path/filepath"
)

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
	UseGPU   bool    `json:"useGPU"`
	SecPerNL float64 `json:"secPerNLogN"` // seconds per N·log2(N) of one force evaluation
	Stars    int     `json:"stars"`       // recommended stars for the step budget
}

// Default returns the settings of a first start.
func Default() *Prefs { return &Prefs{GPU: GPUAuto} }

// Path is the settings file, or "" if the system has no config directory.
func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Galaxy", "prefs.json")
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
