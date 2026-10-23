package sph

import (
	"testing"

	"galaxy/internal/scene"
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
	dudt, divv := make([]float64, n), make([]float64, n)
	var sv Solver
	sv.Forces(s, &p, dudt, divv) // converge h
	b.ResetTimer()
	for range b.N {
		sv.Forces(s, &p, dudt, divv)
	}
}
