package save

import (
	"path/filepath"
	"slices"
	"testing"

	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
	"galaxy/internal/sph"
)

func TestSaveLoadContinuesBitIdentically(t *testing.T) {
	sp := scene.Presets[1].Spec // galaxy collision: holes, substeps, accretion bookkeeping
	s, p := sp.Build(3000)
	p.Gravity = sim.GravityRetardedExtrapolated
	orig := sim.New(s, &bh.Solver{}, p)
	for range 20 {
		orig.Step()
	}
	path := filepath.Join(t.TempDir(), "x"+Extension)
	cam := Camera{Target: [3]float64{1, 2, 3}, Yaw: 0.5, Pitch: 0.7, Distance: 9}
	if err := Write(path, Capture(orig, sp, cam)); err != nil {
		t.Fatal(err)
	}
	f, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Camera != cam || f.Scene.Name != sp.Name || f.Params != p || f.Steps != 20 {
		t.Fatalf("metadata not restored: %+v", f)
	}
	loaded := sim.New(sim.NewState(0), &bh.Solver{}, sim.Params{})
	f.Restore(loaded)
	// Retarded histories are not saved; compare in instant mode from here on.
	orig.Params.Gravity, loaded.Params.Gravity = sim.GravityInstant, sim.GravityInstant
	for range 20 {
		orig.Step()
		loaded.Step()
	}
	a, b := orig.State, loaded.State
	if !slices.Equal(a.PosX, b.PosX) || !slices.Equal(a.VelZ, b.VelZ) || !slices.Equal(a.Mass, b.Mass) ||
		orig.Time != loaded.Time || orig.ExternalEnergy != loaded.ExternalEnergy {
		t.Fatal("loaded simulation diverges from the original")
	}
}

// Gas with an ionizing source (Horsehead): luminosity, ionization and smoothing lengths must
// survive saving, or the loaded run loses its source and diverges.
func TestSaveLoadKeepsGasAndRadiation(t *testing.T) {
	var sp *scene.Spec
	for _, p := range scene.Presets {
		if p.Spec.Name == "Horsehead Nebula" {
			sp = p.Spec
		}
	}
	s, p := sp.Build(2000)
	orig := sim.New(s, &bh.Solver{}, p)
	orig.Hydro = &sph.Solver{}
	for range 6 {
		orig.Step()
	}
	path := filepath.Join(t.TempDir(), "gas"+Extension)
	if err := Write(path, Capture(orig, sp, Camera{})); err != nil {
		t.Fatal(err)
	}
	f, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded := sim.New(sim.NewState(0), &bh.Solver{}, sim.Params{})
	loaded.Hydro = &sph.Solver{}
	f.Restore(loaded)
	for range 6 {
		orig.Step()
		loaded.Step()
	}
	a, b := orig.State, loaded.State
	if !slices.Equal(a.Lum, b.Lum) || !slices.Equal(a.Ion, b.Ion) || !slices.Equal(a.U, b.U) || !slices.Equal(a.PosX, b.PosX) ||
		orig.StarsFormed != loaded.StarsFormed {
		t.Fatal("loaded gas simulation diverges from the original")
	}
}

func TestReadRejectsGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad"+Extension)
	if err := Write(path, &File{Format: 99}); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Error("wrong format version must be rejected")
	}
	if _, err := Read(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("missing file must be an error")
	}
}

func TestLatest(t *testing.T) {
	dir := t.TempDir()
	if _, err := Latest(dir); err == nil {
		t.Error("empty dir must be an error")
	}
	for _, name := range []string{"a", "b"} {
		if err := Write(filepath.Join(dir, name+Extension), &File{Format: Format}); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := Latest(dir); err != nil || filepath.Base(got) != "b"+Extension {
		t.Errorf("Latest = %q, %v", got, err)
	}
}
