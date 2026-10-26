package scene

import (
	"math"
	"math/rand/v2"
	"slices"

	"galaxy/internal/cosmo"
	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// rhoCrit is the critical density today, 2.775·10¹¹ h² M☉/Mpc³.
const rhoCrit = 2.775e11

// addCosmicWeb places the matter of a Zel'dovich realization (Planck 2018 ΛCDM) as n dark matter
// particles at positions q + D(z) Ψ(q), centered on the component's position and at rest (T-073).
// The grid is the smallest power of two with g³ ≥ n (at most 128³); if it has more points than the
// budget, a random subset of n keeps the density unbiased. The scene must have physical units;
// lengths are converted from Mpc/h.
//
// With params.cosmology (comoving mode, T-074) the box becomes the periodic cube of the run: the
// bodies get the Zel'dovich velocities u = a² D f H Ψ, the mass of the mean matter density
// Ω_m 3H₀²/(8πG) of the run's units, and the cosmology its box, Ω_m, H₀ and start a = 1/(1 + z).
func (sp *Spec) addCosmicWeb(s *sim.State, c Component, n int, p *sim.Params) {
	msun, okM := sp.MassUnit()
	_, pc, okL := sp.Units()
	if !okM || !okL || c.Box <= 0 || n < 8 {
		return
	}
	g := 2
	for g*g*g < n && g < 128 {
		g *= 2
	}
	n = min(n, g*g*g)
	keep := rand.New(rand.NewPCG(c.Seed, 0xc05)).Perm(g * g * g)[:n]
	slices.Sort(keep)
	cos := cosmo.Planck2018
	f := cosmo.Zeldovich(cosmo.NewPower(cos), g, c.Box, c.Seed)
	d := cos.Growth(1 / (1 + c.Redshift))
	mpc := 1e6 / pc / cos.H // simulation units per Mpc/h
	boxMsun := cos.OmegaM * rhoCrit * cos.H * cos.H * math.Pow(c.Box/cos.H, 3)
	mass := boxMsun / msun / float64(n)
	var velocity float64 // u per unit of Ψ (sim units per Mpc/h), comoving mode only
	if cp := p.Cosmology; cp != nil {
		a := 1 / (1 + c.Redshift)
		l := c.Box * mpc
		cp.OmegaM, cp.Box, cp.AStart = cos.OmegaM, l, a
		cp.H0 = cos.H * 100 / sp.KmPerS * pc / 1e6 // 100h km/s/Mpc in simulation units
		mass = cp.OmegaM * 3 * cp.H0 * cp.H0 / (8 * math.Pi) * l * l * l / float64(n)
		velocity = a * a * d * cos.GrowthRate(a) * cp.Hubble(a) * mpc
	}
	center := v3(c.Position)
	cell := c.Box / float64(g)
	for _, idx := range keep {
		i, j, k := idx/(g*g), idx/g%g, idx%g
		q := vec.Vec3{X: (float64(i) + 0.5) * cell, Y: (float64(j) + 0.5) * cell, Z: (float64(k) + 0.5) * cell}
		pos := vec.Vec3{X: q.X + d*f.Disp[0][idx], Y: q.Y + d*f.Disp[1][idx], Z: q.Z + d*f.Disp[2][idx]}
		pos = pos.Sub(vec.Vec3{X: c.Box / 2, Y: c.Box / 2, Z: c.Box / 2}).Scale(mpc)
		vel := vec.Vec3{X: f.Disp[0][idx], Y: f.Disp[1][idx], Z: f.Disp[2][idx]}.Scale(velocity)
		s.Add(sim.Body{Pos: center.Add(pos), Vel: vel, Mass: mass, Kind: sim.DarkMatter})
	}
}
