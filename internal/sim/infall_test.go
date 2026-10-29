package sim

import (
	"math"
	"testing"

	"galaxy/internal/vec"
)

// rotatingDisk is a black hole of mass 1 at the origin with a ring of stars rotating about +z.
func rotatingDisk(n int) *State {
	s := NewState(n + 1)
	s.Add(Body{Mass: 1, Kind: BlackHole})
	for i := range n {
		phi := 2 * math.Pi * float64(i) / float64(n)
		r := 1 + 0.5*float64(i%5)/5
		v := math.Sqrt(1 / r)
		s.Add(Body{Pos: vec.Vec3{X: r * math.Cos(phi), Y: r * math.Sin(phi)},
			Vel: vec.Vec3{X: -v * math.Sin(phi), Y: v * math.Cos(phi)}, Mass: 1e-4, Kind: Star})
	}
	return s
}

func TestInfallAddsGasAtRateInThePlaneAndBooksEnergy(t *testing.T) {
	s := rotatingDisk(200)
	p := Params{Dt: 1e-3, Softening: 0.05, Gas: GasParams{UFloor: 0.01},
		Infall: []InfallSource{{Center: s.ID[0], Rate: 1, Mass: 0.01, Scale: 0.5, RMax: 1.5, Height: 0.2}}}
	m := New(s, pairSolver{}, p)
	for range 1000 {
		n0 := s.Len()
		e0, x0 := thermalEnergy(s)+exactEnergy(s, p), m.ExternalEnergy
		m.Step()
		if s.Len() > n0 { // an infall step: the energy change is the booked one (plus integration)
			e1 := thermalEnergy(s) + exactEnergy(s, p)
			if d := e1 - e0 - (m.ExternalEnergy - x0); math.Abs(d) > 1e-6 {
				t.Fatalf("energy changed by %.6g, booked %.6g", e1-e0, m.ExternalEnergy-x0)
			}
		}
	}
	gas := 0
	for i := range s.Len() {
		if s.Kind[i] != Gas {
			continue
		}
		gas++
		pos := vec.Vec3{X: s.PosX[i], Y: s.PosY[i], Z: s.PosZ[i]}
		if pos.Len() > 3 {
			t.Errorf("gas body %d at distance %.3g, far outside the disk", i, pos.Len())
		}
		l := pos.Cross(vec.Vec3{X: s.VelX[i], Y: s.VelY[i], Z: s.VelZ[i]})
		if l.Z <= 0 {
			t.Errorf("gas body %d rotates against the disk: L = %+v", i, l)
		}
	}
	if gas < 99 || gas > 100 || math.Abs(m.InfallMass-float64(gas)*0.01) > 1e-12 {
		t.Errorf("%d gas bodies, %.4g mass fallen in after t = 1 at rate 1 (bodies 0.01)", gas, m.InfallMass)
	}
}

func TestInfallFollowsTheHeaviestHoleAfterAMerger(t *testing.T) {
	s := rotatingDisk(50)
	h := s.Add(Body{Pos: vec.Vec3{X: 10}, Mass: 3, Kind: BlackHole})
	f := InfallSource{Center: 999999, Rate: 1, Mass: 0.01, Scale: 0.5, RMax: 1, Height: 0.1}
	m := New(s, pairSolver{}, Params{Dt: 1e-3, Softening: 0.05})
	if c := m.infallCenter(&f); c != h || f.Center != s.ID[h] {
		t.Errorf("center %d (ID %d), want the heaviest hole %d", c, f.Center, h)
	}
}
