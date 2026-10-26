package cosmo

import (
	"math"
	"testing"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// lattice places g³ bodies of the mean matter density ρ̄ = Ω_m 3H₀²/(8πG) at q + D Ψ(q) with
// velocities u = a² D f H Ψ, given a displacement function Ψ (today's normalization).
func lattice(c *sim.Cosmology, cos Cosmology, g int, a float64, psi func(q vec.Vec3) vec.Vec3) *sim.State {
	l := c.Box
	m := c.OmegaM * 3 * c.H0 * c.H0 / (8 * math.Pi) * l * l * l / float64(g*g*g)
	d, f, h := cos.Growth(a), cos.GrowthRate(a), c.Hubble(a)
	s := sim.NewState(g * g * g)
	for i := range g {
		for j := range g {
			for k := range g {
				q := vec.Vec3{X: (float64(i)+0.5)/float64(g)*l - l/2, Y: (float64(j)+0.5)/float64(g)*l - l/2, Z: (float64(k)+0.5)/float64(g)*l - l/2}
				p := psi(q)
				s.Add(sim.Body{Pos: q.Add(p.Scale(d)), Vel: p.Scale(a * a * d * f * h), Mass: m, Kind: sim.DarkMatter})
			}
		}
	}
	return s
}

// TestPlaneWaveGrowsLinearly: a small plane-wave displacement grows with the linear growth factor
// D(a) (exact for Zel'dovich in one dimension before shell crossing).
func TestPlaneWaveGrowsLinearly(t *testing.T) {
	cos := Cosmology{H: 1, OmegaM: 0.31, OmegaB: 0.049, Sigma8: 0.8, NS: 1}
	c := &sim.Cosmology{OmegaM: cos.OmegaM, H0: 1, Box: 100, AStart: 0.02, AEnd: 0.25}
	const g = 32
	k := 2 * math.Pi / c.Box * 2
	amp := 0.05 / k // δ = −∇·Ψ has amplitude 0.05 today: linear until the end
	psi := func(q vec.Vec3) vec.Vec3 { return vec.Vec3{X: amp * math.Sin(k*q.X)} }
	s := lattice(c, cos, g, c.AStart, psi)
	q := make([]float64, s.Len())
	for i := range q {
		q[i] = s.PosX[i] - cos.Growth(c.AStart)*psi(vec.Vec3{X: s.PosX[i]}).X // first order is enough here
	}
	measure := func() float64 {
		var sum float64
		for i := range q {
			d := s.PosX[i] - q[i]
			d -= c.Box * math.Round(d/c.Box)
			sum += d * math.Sin(k*q[i])
		}
		return 2 * sum / float64(len(q))
	}
	m := sim.New(s, nil, sim.Params{Dt: 0.02, Cosmology: c})
	a0 := measure()
	for m.ScaleFactor < c.AEnd {
		m.Step()
	}
	got, want := measure()/a0, cos.Growth(c.AEnd)/cos.Growth(c.AStart)
	t.Logf("growth %.3f, linear theory %.3f", got, want)
	if math.Abs(got/want-1) > 0.03 {
		t.Errorf("displacement grew by %.3f, want D ratio %.3f", got, want)
	}
}

// TestComovingConservesMomentumAndLayzerIrvine runs a Zel'dovich realization into the nonlinear
// regime: total momentum stays zero, and the Layzer–Irvine equation d[a(K+W)] = −K da holds.
func TestComovingConservesMomentumAndLayzerIrvine(t *testing.T) {
	cos := Planck2018
	c := &sim.Cosmology{OmegaM: cos.OmegaM, H0: 1, Box: 100, AStart: 0.05, AEnd: 0.5}
	const g = 32
	f := Zeldovich(NewPower(cos), g, c.Box, 7) // Box in Mpc/h with H0 = 1 per time unit: only the shape matters
	idx := func(q vec.Vec3) int {
		cell := func(x float64) int { return int(math.Floor((x + c.Box/2) / c.Box * g)) }
		return (cell(q.X)*g+cell(q.Y))*g + cell(q.Z)
	}
	s := lattice(c, cos, g, c.AStart, func(q vec.Vec3) vec.Vec3 {
		i := idx(q)
		return vec.Vec3{X: f.Disp[0][i], Y: f.Disp[1][i], Z: f.Disp[2][i]}
	})
	pm := NewPM(c.Box, s.Len())
	old := sim.ComovingSolver
	sim.ComovingSolver = func(sim.Params, int) sim.Solver { return pm }
	defer func() { sim.ComovingSolver = old }()
	m := sim.New(s, nil, sim.Params{Dt: 0.01, Cosmology: c})
	energies := func(a float64) (k, w float64) {
		pm.Accelerations(s, m.Params)
		phi := pm.Potential(s)
		for i := range s.Len() {
			u2 := s.VelX[i]*s.VelX[i] + s.VelY[i]*s.VelY[i] + s.VelZ[i]*s.VelZ[i]
			k += 0.5 * s.Mass[i] * u2 / (a * a)
			w += 0.5 * s.Mass[i] * phi[i] / a
		}
		return k, w
	}
	a := c.AStart
	k0, w0 := energies(a)
	i0 := a * (k0 + w0)
	var integral float64
	kPrev, aPrev := k0, a
	for m.ScaleFactor < c.AEnd || m.ScaleFactor == 0 {
		m.Step()
		a = m.ScaleFactor
		k, _ := energies(a)
		integral += 0.5 * (k + kPrev) * (a - aPrev)
		kPrev, aPrev = k, a
	}
	k1, w1 := energies(a)
	drift := (a*(k1+w1) + integral - i0) / math.Abs(a*w1)
	t.Logf("Layzer–Irvine: relative error %.3g (|W| grew from %.3g to %.3g)", drift, math.Abs(w0), math.Abs(w1))
	if math.Abs(drift) > 0.03 {
		t.Errorf("Layzer–Irvine violated by %.3g", drift)
	}
	var p vec.Vec3
	for i := range s.Len() {
		p = p.Add(vec.Vec3{X: s.VelX[i], Y: s.VelY[i], Z: s.VelZ[i]}.Scale(s.Mass[i]))
	}
	if p.Len() > 1e-10 {
		t.Errorf("momentum %v", p)
	}
}
