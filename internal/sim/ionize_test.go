package sim

import (
	"testing"

	"galaxy/internal/vec"
)

// A dense blob between a source and gas behind it casts a shadow; gas beside it is ionized.
func TestIonizationCastsShadows(t *testing.T) {
	s := NewState(3000)
	s.Add(Body{Pos: vec.Vec3{X: -10}, Mass: 1, Lum: 1}) // source far to the left
	for k := range 1000 {                               // dense blob at the origin
		f := float64(k)
		s.Add(Body{Pos: vec.Vec3{X: 0.3 * fract(f*0.618), Y: 0.3 * fract(f*0.414), Z: 0.3 * fract(f*0.732)}, Mass: 1, Kind: Gas})
	}
	addBackground(s)
	behind := s.Add(Body{Pos: vec.Vec3{X: 3, Y: 0.15, Z: 0.15}, Mass: 1e-6, Kind: Gas})
	beside := s.Add(Body{Pos: vec.Vec3{X: 3, Y: 4, Z: 0.15}, Mass: 1e-6, Kind: Gas})
	front := s.Add(Body{Pos: vec.Vec3{X: -3, Y: 0.15, Z: 0.15}, Mass: 1e-6, Kind: Gas})
	m := New(s, nopSolver{}, Params{Gas: GasParams{IonOpacity: 1}})
	if !m.ionize() {
		t.Fatal("source not found")
	}
	if s.Ion[behind] > 0.01 {
		t.Errorf("gas behind the blob is ionized (%.3f), expected shadow", s.Ion[behind])
	}
	if s.Ion[beside] < 0.99 || s.Ion[front] < 0.99 {
		t.Errorf("exposed gas must be ionized: beside %.3f, in front %.3f", s.Ion[beside], s.Ion[front])
	}
}

func fract(x float64) float64 { return x - float64(int(x)) }

// addBackground fills [-5, 5]³ with negligible gas, so that the grid spans the whole test region
// as it does in a scene.
func addBackground(s *State) {
	for k := range 2000 {
		f := float64(k)
		s.Add(Body{Pos: vec.Vec3{X: 10*fract(f*0.618) - 5, Y: 10*fract(f*0.414) - 5, Z: 10*fract(f*0.732) - 5}, Mass: 1e-9, Kind: Gas})
	}
}

// Ionized gas is transparent apart from recombinations: an already ionized blob does not shadow
// unless IonRecomb is set.
func TestIonizedGasIsTransparentWithoutRecombinations(t *testing.T) {
	build := func(recomb float64) float64 {
		s := NewState(1100)
		s.Add(Body{Pos: vec.Vec3{X: -10}, Mass: 1, Lum: 1})
		for k := range 1000 {
			f := float64(k)
			i := s.Add(Body{Pos: vec.Vec3{X: 0.3 * fract(f*0.618), Y: 0.3 * fract(f*0.414), Z: 0.3 * fract(f*0.732)}, Mass: 1, Kind: Gas})
			s.Ion[i] = 1 // already ionized
		}
		addBackground(s)
		behind := s.Add(Body{Pos: vec.Vec3{X: 3, Y: 0.15, Z: 0.15}, Mass: 1e-6, Kind: Gas})
		m := New(s, nopSolver{}, Params{Gas: GasParams{IonOpacity: 1, IonRecomb: recomb}})
		m.ionize()
		return s.Ion[behind]
	}
	if ion := build(0); ion < 0.99 {
		t.Errorf("ionized blob without recombinations shadows: %.3f", ion)
	}
	if ion := build(1); ion > 0.01 {
		t.Errorf("dense ionized blob with recombinations should shadow: %.3f", ion)
	}
}
