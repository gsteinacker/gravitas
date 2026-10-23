package scene

import "galaxy/internal/sim"

// Preset is a named starting scene for roughly n bodies, built from a Spec.
type Preset struct {
	Name  string
	Spec  *Spec
	Build func(n int) (*sim.State, sim.Params)
}

// FromSpec wraps a scene recipe as a preset.
func FromSpec(sp *Spec) Preset {
	return Preset{Name: sp.Name, Spec: sp, Build: sp.Build}
}

// Presets are the built-in scenes (scenes/*.json), selectable with the number keys.
var Presets = mustBuiltin()

func mustBuiltin() []Preset {
	specs, err := Builtin()
	if err != nil {
		panic("built-in scenes: " + err.Error()) // embedded files, covered by tests
	}
	presets := make([]Preset, len(specs))
	for i, sp := range specs {
		presets[i] = FromSpec(sp)
	}
	return presets
}
