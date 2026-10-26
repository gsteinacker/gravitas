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
func (sp *Spec) addCosmicWeb(s *sim.State, c Component, n int) {
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
	center := v3(c.Position)
	cell := c.Box / float64(g)
	for _, idx := range keep {
		i, j, k := idx/(g*g), idx/g%g, idx%g
		q := vec.Vec3{X: (float64(i) + 0.5) * cell, Y: (float64(j) + 0.5) * cell, Z: (float64(k) + 0.5) * cell}
		p := vec.Vec3{X: q.X + d*f.Disp[0][idx], Y: q.Y + d*f.Disp[1][idx], Z: q.Z + d*f.Disp[2][idx]}
		p = p.Sub(vec.Vec3{X: c.Box / 2, Y: c.Box / 2, Z: c.Box / 2}).Scale(mpc)
		s.Add(sim.Body{Pos: center.Add(p), Mass: mass, Kind: sim.DarkMatter})
	}
}
