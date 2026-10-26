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

// Format is the version of the file layout. Versions 1 (no gas fields) and 2 (no radiation
// fields) are still readable.
const Format = 4

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
	Formed   int     // Simulation.StarsFormed (since format 3)
	Camera   Camera

	PosX, PosY, PosZ []float64
	VelX, VelY, VelZ []float64
	Mass             []float64
	Kind             []sim.Kind
	U, Born          []float64   // since format 2: gas internal energy, birth time of stars from gas
	Lum, Ion, H      []float64   // since format 3: ionizing luminosity, ionized fraction, smoothing length
	Forces           *sim.Forces // since format 3: last force evaluation, for an exact continuation
	SpinX, SpinY     []float64   // since format 4: angular momentum of black holes
	SpinZ            []float64
	ScaleFactor      float64 // since format 4: a of the comoving mode, 0 otherwise
}

// Capture copies the simulation. Call it on the simulation goroutine (sim.Runner.Do).
func Capture(m *sim.Simulation, sp *scene.Spec, cam Camera) *File {
	s := m.State
	return &File{
		Format: Format, Saved: time.Now(), Scene: *sp, Params: m.Params,
		Time: m.Time, Steps: m.Steps, ScaleFactor: m.ScaleFactor, External: m.ExternalEnergy, Formed: m.StarsFormed, Camera: cam,
		PosX: slices.Clone(s.PosX), PosY: slices.Clone(s.PosY), PosZ: slices.Clone(s.PosZ),
		VelX: slices.Clone(s.VelX), VelY: slices.Clone(s.VelY), VelZ: slices.Clone(s.VelZ),
		Mass: slices.Clone(s.Mass), Kind: slices.Clone(s.Kind),
		U: slices.Clone(s.U), Born: slices.Clone(s.Born),
		Lum: slices.Clone(s.Lum), Ion: slices.Clone(s.Ion), H: slices.Clone(s.H),
		Forces: m.LastForces(),
		SpinX:  slices.Clone(s.SpinX), SpinY: slices.Clone(s.SpinY), SpinZ: slices.Clone(s.SpinZ),
	}
}

// State rebuilds the bodies (in the saved order).
func (f *File) State() *sim.State {
	s := sim.NewState(len(f.Mass))
	for i := range f.Mass {
		b := sim.Body{
			Pos:  vec.Vec3{X: f.PosX[i], Y: f.PosY[i], Z: f.PosZ[i]},
			Vel:  vec.Vec3{X: f.VelX[i], Y: f.VelY[i], Z: f.VelZ[i]},
			Mass: f.Mass[i], Kind: f.Kind[i],
		}
		if i < len(f.U) {
			b.U = f.U[i]
		}
		if i < len(f.Lum) {
			b.Lum = f.Lum[i]
		}
		if i < len(f.SpinX) {
			b.Spin = vec.Vec3{X: f.SpinX[i], Y: f.SpinY[i], Z: f.SpinZ[i]}
		}
		s.Add(b)
		if i < len(f.Born) {
			s.Born[i] = f.Born[i]
		}
		if i < len(f.Ion) {
			s.Ion[i], s.H[i] = f.Ion[i], f.H[i]
		}
	}
	return s
}

// Restore puts the saved state into m.
func (f *File) Restore(m *sim.Simulation) {
	m.Restore(f.State(), f.Params, f.Time, f.Steps, f.External)
	m.RestoreForces(f.Forces)
	m.StarsFormed = f.Formed
	m.ScaleFactor = f.ScaleFactor
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
	if f.Format < 1 || f.Format > Format {
		return nil, fmt.Errorf("%s: format %d, want 1 to %d", path, f.Format, Format)
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
