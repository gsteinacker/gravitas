// Package save writes and reads complete simulation states (".galaxy" files: gob, gzip).
package save

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// Format is the version of the file layout; Read rejects other versions.
const Format = 1

// Extension of saved states.
const Extension = ".galaxy"

// Camera is the view at the time of saving.
type Camera struct {
	Target               [3]float64
	Yaw, Pitch, Distance float64
}

// File is a complete, exact snapshot of a run.
type File struct {
	Format   int
	Saved    time.Time
	Scene    scene.Spec // name, info, units, effects; components are informational only
	Params   sim.Params
	Time     float64
	Steps    int
	External float64 // Simulation.ExternalEnergy
	Camera   Camera

	PosX, PosY, PosZ []float64
	VelX, VelY, VelZ []float64
	Mass             []float64
	Kind             []sim.Kind
}

// Capture copies the simulation. Call it on the simulation goroutine (sim.Runner.Do).
func Capture(m *sim.Simulation, sp *scene.Spec, cam Camera) *File {
	s := m.State
	return &File{
		Format: Format, Saved: time.Now(), Scene: *sp, Params: m.Params,
		Time: m.Time, Steps: m.Steps, External: m.ExternalEnergy, Camera: cam,
		PosX: slices.Clone(s.PosX), PosY: slices.Clone(s.PosY), PosZ: slices.Clone(s.PosZ),
		VelX: slices.Clone(s.VelX), VelY: slices.Clone(s.VelY), VelZ: slices.Clone(s.VelZ),
		Mass: slices.Clone(s.Mass), Kind: slices.Clone(s.Kind),
	}
}

// State rebuilds the bodies (in the saved order).
func (f *File) State() *sim.State {
	s := sim.NewState(len(f.Mass))
	for i := range f.Mass {
		s.Add(sim.Body{
			Pos:  vec.Vec3{X: f.PosX[i], Y: f.PosY[i], Z: f.PosZ[i]},
			Vel:  vec.Vec3{X: f.VelX[i], Y: f.VelY[i], Z: f.VelZ[i]},
			Mass: f.Mass[i], Kind: f.Kind[i],
		})
	}
	return s
}

// Restore puts the saved state into m.
func (f *File) Restore(m *sim.Simulation) {
	m.Restore(f.State(), f.Params, f.Time, f.Steps, f.External)
}

// Write stores f at path atomically (temporary file, then rename).
func Write(path string, f *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".saving-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	zw, _ := gzip.NewWriterLevel(tmp, gzip.BestSpeed)
	if err := gob.NewEncoder(zw).Encode(f); err != nil {
		tmp.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Read loads a saved state.
func Read(path string) (*File, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	zr, err := gzip.NewReader(fh)
	if err != nil {
		return nil, fmt.Errorf("%s: not a saved state: %w", path, err)
	}
	var f File
	if err := gob.NewDecoder(zr).Decode(&f); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if f.Format != Format {
		return nil, fmt.Errorf("%s: format %d, want %d", path, f.Format, Format)
	}
	if n := len(f.Mass); len(f.PosX) != n || len(f.VelZ) != n || len(f.Kind) != n {
		return nil, fmt.Errorf("%s: inconsistent body arrays", path)
	}
	return &f, nil
}

// NewPath returns a fresh file name in dir, based on the current time.
func NewPath(dir string) string {
	return filepath.Join(dir, "galaxy-"+time.Now().Format("20060102-150405")+Extension)
}

// Latest returns the most recently modified saved state in dir.
func Latest(dir string) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*"+Extension))
	if err != nil {
		return "", err
	}
	var best string
	var bestTime time.Time
	for _, f := range files {
		if fi, err := os.Stat(f); err == nil && fi.ModTime().After(bestTime) {
			best, bestTime = f, fi.ModTime()
		}
	}
	if best == "" {
		return "", fmt.Errorf("no saved states in %s", dir)
	}
	return best, nil
}
