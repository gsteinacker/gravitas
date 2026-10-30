package scene

import (
	"math"
	"slices"
	"testing"

	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
	"galaxy/internal/vec"
)

// axisRatios returns b/a and c/a of the bodies within the radius holding frac of them, from the
// eigenvalues of their second-moment tensor (Jacobi rotations).
func axisRatios(s *sim.State, frac float64) (ba, ca float64) {
	r := make([]float64, s.Len())
	for i := range r {
		r[i] = math.Sqrt(s.PosX[i]*s.PosX[i] + s.PosY[i]*s.PosY[i] + s.PosZ[i]*s.PosZ[i])
	}
	sorted := slices.Clone(r)
	slices.Sort(sorted)
	rmax := sorted[int(frac*float64(len(sorted)))]
	var m [3][3]float64
	for i := range r {
		if r[i] > rmax {
			continue
		}
		p := [3]float64{s.PosX[i], s.PosY[i], s.PosZ[i]}
		for j := range 3 {
			for k := range 3 {
				m[j][k] += p[j] * p[k]
			}
		}
	}
	for range 50 {
		for p := range 3 {
			for q := p + 1; q < 3; q++ {
				if math.Abs(m[p][q]) < 1e-12 {
					continue
				}
				theta := 0.5 * math.Atan2(2*m[p][q], m[q][q]-m[p][p])
				c, sn := math.Cos(theta), math.Sin(theta)
				for k := range 3 { // columns
					mp, mq := m[k][p], m[k][q]
					m[k][p], m[k][q] = c*mp-sn*mq, sn*mp+c*mq
				}
				for k := range 3 { // rows
					mp, mq := m[p][k], m[q][k]
					m[p][k], m[q][k] = c*mp-sn*mq, sn*mp+c*mq
				}
			}
		}
	}
	ev := []float64{m[0][0], m[1][1], m[2][2]}
	slices.Sort(ev)
	return math.Sqrt(ev[1] / ev[2]), math.Sqrt(ev[0] / ev[2])
}

// A triaxial halo (stretched Plummer sphere, T-084) settles but stays triaxial over two orbits at
// its scale radius.
func TestTriaxialHaloKeepsItsShape(t *testing.T) {
	s := sim.NewState(3000)
	AddHalo(s, 3000, 8.5, 4, vec.Vec3{}, vec.Vec3{}, 7, 0.95)
	ShapeHalo(s, 0, vec.Vec3{X: 1, Y: 0.8, Z: 0.6}, 0, 0, vec.Vec3{}, vec.Vec3{})
	ba0, ca0 := axisRatios(s, 0.999)
	m := sim.New(s, &bh.Solver{}, sim.Params{Dt: 0.02, Softening: 0.1})
	for m.Time < 60 {
		m.Step()
	}
	ba, ca := axisRatios(m.State, 0.999)
	t.Logf("b/a %.2f → %.2f, c/a %.2f → %.2f", ba0, ba, ca0, ca)
	if math.Abs(ba0-0.8) > 0.05 || math.Abs(ca0-0.6) > 0.05 {
		t.Errorf("initial axis ratios %.2f, %.2f, want 0.8, 0.6", ba0, ca0)
	}
	if math.Abs(ba-ba0) > 0.1 || math.Abs(ca-ca0) > 0.1 {
		t.Errorf("axis ratios drifted from %.2f, %.2f to %.2f, %.2f", ba0, ca0, ba, ca)
	}
}
