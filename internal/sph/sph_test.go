package sph

import (
	"math"
	"math/rand/v2"
	"testing"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

func TestKernelNormalizationAndDerivative(t *testing.T) {
	const h = 0.7
	var integral float64
	const n = 20000
	dr := 2 * h / n
	for k := range n {
		r := (float64(k) + 0.5) * dr
		integral += 4 * math.Pi * r * r * W(r, h) * dr
	}
	if math.Abs(integral-1) > 1e-6 {
		t.Errorf("∫W dV = %.9f, want 1", integral)
	}
	for _, r := range []float64{0.1, 0.5, 0.69, 0.71, 1.2, 1.39} {
		num := (W(r+1e-7, h) - W(r-1e-7, h)) / 2e-7
		if math.Abs(num-DW(r, h)) > 1e-5 {
			t.Errorf("DW(%v) = %v, numerical %v", r, DW(r, h), num)
		}
	}
}

func cloud(n int, seed uint64) *sim.State {
	rng := rand.New(rand.NewPCG(seed, 3))
	s := sim.NewState(n)
	for range n {
		s.Add(sim.Body{
			Pos:  vec.Vec3{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: 0.5 * rng.NormFloat64()},
			Vel:  vec.Vec3{X: rng.NormFloat64() * 0.3, Y: rng.NormFloat64() * 0.3, Z: rng.NormFloat64() * 0.3},
			Mass: (0.5 + rng.Float64()) / float64(n), Kind: sim.Gas, U: 0.1 + 0.2*rng.Float64(),
		})
	}
	return s
}

func TestNeighborSearchMatchesBruteForce(t *testing.T) {
	s := cloud(2000, 1)
	rng := rand.New(rand.NewPCG(5, 6))
	for i := range s.Len() {
		s.H[i] = 0.05 + 0.3*rng.Float64()
	}
	var gas []int32
	for i := range s.Len() {
		gas = append(gas, int32(i))
	}
	var tr tree
	tr.build(gas, s.PosX, s.PosY, s.PosZ, s.H)
	for _, i := range []int{0, 17, 999, 1500} {
		for _, sym := range []bool{false, true} {
			got := map[int32]bool{}
			tr.each(s.PosX[i], s.PosY[i], s.PosZ[i], 2*s.H[i], sym, nil, func(j int32, _ float64) { got[j] = true })
			for j := range s.Len() {
				r := s.Body(i).Pos.Sub(s.Body(j).Pos).Len()
				lim := 2 * s.H[i]
				if sym {
					lim = math.Max(lim, 2*s.H[j])
				}
				if (r < lim) != got[int32(j)] {
					t.Fatalf("i=%d j=%d sym=%v: r=%.4f lim=%.4f, found %v", i, j, sym, r, lim, got[int32(j)])
				}
			}
		}
	}
}

func TestDensityOfLattice(t *testing.T) {
	const k, spacing = 16, 0.1
	s := sim.NewState(k * k * k)
	for a := range k {
		for b := range k {
			for c := range k {
				s.Add(sim.Body{Pos: vec.Vec3{X: float64(a) * spacing, Y: float64(b) * spacing, Z: float64(c) * spacing}, Mass: 1, Kind: sim.Gas, U: 1})
			}
		}
	}
	dudt, divv := make([]float64, s.Len()), make([]float64, s.Len())
	(&Solver{}).Forces(s, &sim.Params{}, dudt, divv)
	want := 1 / (spacing * spacing * spacing)
	center := (k/2)*k*k + (k/2)*k + k/2
	if rho := s.Rho[center]; math.Abs(rho/want-1) > 0.03 {
		t.Errorf("lattice density %.1f, want %.1f", rho, want)
	}
	if a := math.Hypot(s.AccX[center], s.AccY[center]); a > 1e-6*want {
		t.Errorf("interior of a uniform lattice should feel no pressure force, got %.3g", a)
	}
}

func TestHydroConservesMomentumAndEnergy(t *testing.T) {
	s := cloud(3000, 2)
	dudt, divv := make([]float64, s.Len()), make([]float64, s.Len())
	(&Solver{}).Forces(s, &sim.Params{}, dudt, divv)
	var px, py, pz, dE, scale float64
	for i := range s.Len() {
		m := s.Mass[i]
		px, py, pz = px+m*s.AccX[i], py+m*s.AccY[i], pz+m*s.AccZ[i]
		dE += m * (s.VelX[i]*s.AccX[i] + s.VelY[i]*s.AccY[i] + s.VelZ[i]*s.AccZ[i] + dudt[i])
		scale += m * math.Abs(dudt[i])
	}
	if p := math.Sqrt(px*px + py*py + pz*pz); p > 1e-12*scale*10 {
		t.Errorf("total hydro force %.3g, want 0", p)
	}
	if math.Abs(dE) > 1e-10*scale {
		t.Errorf("dE/dt = %.3g (scale %.3g), want 0", dE, scale)
	}
}
