package sph

import (
	"testing"

	"galaxy/internal/scene"
	"galaxy/internal/sim"
)

// BenchmarkHorsehead measures one hydro evaluation of the Horsehead scene, the most
// inhomogeneous gas distribution of the built-in scenes.
func BenchmarkHorsehead(b *testing.B) {
	var spec *scene.Spec
	for _, p := range scene.Presets {
		if p.Spec.Name == "Horsehead Nebula" {
			spec = p.Spec
		}
	}
	s, p := spec.Build(60000)
	n := s.Len()
	var out sim.HydroOut
	out.Resize(n)
	var sv Solver
	sv.Forces(s, &p, nil, nil, &out) // converge h
	b.ResetTimer()
	for range b.N {
		sv.Forces(s, &p, nil, nil, &out)
	}
}
