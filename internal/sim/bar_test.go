package sim

import (
	"math"
	"math/rand/v2"
	"testing"

	"galaxy/internal/vec"
)

// barDisk is a rotating disk of stars around a hole, stretched by stretch along x and turned by
// angle about z.
func barDisk(stretch, angle float64) *State {
	rng := rand.New(rand.NewPCG(1, 2))
	s := NewState(4001)
	s.Add(Body{Mass: 1, Kind: BlackHole})
	for range 4000 {
		r := -math.Log(1 - rng.Float64()*0.95)
		phi := 2 * math.Pi * rng.Float64()
		p := vec.Vec3{X: r * math.Cos(phi) * stretch, Y: r * math.Sin(phi), Z: 0.02 * rng.NormFloat64()}
		v := vec.Vec3{X: -math.Sin(phi), Y: math.Cos(phi)}
		s.Add(Body{Pos: p.RotateZ(angle), Vel: v.RotateZ(angle), Mass: 1e-4})
	}
	return s
}

func TestMeasureBar(t *testing.T) {
	if b := MeasureBar(barDisk(1, 0)); b.A2 > 0.05 {
		t.Errorf("round disk: A2 = %.3f, want ≈ 0", b.A2)
	}
	b := MeasureBar(barDisk(1.6, 0.5))
	if b.A2 < 0.15 {
		t.Errorf("elongated disk: A2 = %.3f, want > 0.15", b.A2)
	}
	if d := math.Abs(math.Remainder(b.Phase-0.5, math.Pi)); d > 0.05 {
		t.Errorf("bar phase %.3f, want 0.5", b.Phase)
	}
	sn := &Snapshot{}
	s := barDisk(1.6, 0.5)
	for i := range s.Len() {
		sn.X, sn.Y, sn.Z = append(sn.X, float32(s.PosX[i])), append(sn.Y, float32(s.PosY[i])), append(sn.Z, float32(s.PosZ[i]))
		sn.VX, sn.VY, sn.VZ = append(sn.VX, float32(s.VelX[i])), append(sn.VY, float32(s.VelY[i])), append(sn.VZ, float32(s.VelZ[i]))
		sn.Mass, sn.Kind = append(sn.Mass, float32(s.Mass[i])), append(sn.Kind, s.Kind[i])
	}
	if sb := sn.Bar(); math.Abs(sb.A2-b.A2) > 1e-4 {
		t.Errorf("snapshot A2 %.4f, state %.4f", sb.A2, b.A2)
	}
}
