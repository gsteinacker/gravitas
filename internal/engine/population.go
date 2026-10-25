package engine

import "math"

// A star particle of a galaxy scene stands for a whole stellar population (10⁵–10⁶ M☉), not one
// star (T-070). The renderer draws it as a soft glow of its unresolved stars plus a few member
// stars scattered within PopRadius: from afar the glows merge into the diffuse light of the disk,
// zoomed in the members resolve into single stars. Gas above the star formation threshold shines
// with the young stars it forms below the resolution (Kennicutt–Schmidt: SFR ∝ ρ^1.5).
const (
	// minPopulationMsun is the smallest mean particle mass drawn as a population; lighter
	// particles are single stars (clusters, the Horsehead).
	minPopulationMsun = 100
	// youngLightRatio is the light per mass of a population younger than youngSpan relative to an
	// old one (Starburst99, bolometric, mostly blue and UV: about 100–1000 in the first 10 Myr).
	youngLightRatio = 300
	// youngSpan is how long the O and early B stars of a new population shine (years).
	youngSpan = 10e6
	// clumping is ⟨ρ^1.5⟩ / ⟨ρ⟩^1.5 of the gas below the resolution: stars form in dense clumps the
	// smoothed density does not show (typically 3–10).
	clumping = 5
	// maxStarFormingLight caps the young light of a gas particle in units of an old star of the
	// same mass.
	maxStarFormingLight = 20
)

// Population is what the renderer needs to draw particles as stellar populations.
type Population struct {
	Radius    float32 // scatter radius of the member stars (world units), 0: particles are stars
	SFDensity float32 // gas denser than this forms stars (0: no star formation)
	// SFLight times √ρ is the young light of star-forming gas in units of an old star of the same
	// mass: clumping · youngLightRatio · ε · τ / t_ff with τ = youngSpan and t_ff = √(3π / (32 ρ)) (G = 1).
	SFLight float32
}

// population derives the population drawing from the scene: the softening is the size the
// simulation resolves, so the member stars scatter within it.
func (e *Engine) population(meanStarMass float64) Population {
	sp := e.Scene
	msun, ok := sp.MassUnit()
	if !ok || meanStarMass*msun < minPopulationMsun {
		return Population{}
	}
	p := Population{Radius: float32(sp.Params.Softening)}
	g := sp.Params.Gas
	if g.SFDensity > 0 && sp.YearsPer > 0 {
		eff := g.SFEfficiency
		if eff == 0 {
			eff = 0.02 // sim.defaultSFEfficiency
		}
		p.SFDensity = float32(g.SFDensity)
		p.SFLight = float32(clumping * youngLightRatio * eff * youngSpan / sp.YearsPer * math.Sqrt(32/(3*math.Pi)))
	}
	return p
}

// youngBoost returns the alpha and size factors of a star particle formed age ago, for stars
// younger than youngAge: blue OB stars make a fresh population about ten times brighter. The
// shader (spriteVertex) uses the same factors.
func youngBoost(age, youngAge float64) (alpha, size float64) {
	if age < 0 || age >= youngAge {
		return 1, 1
	}
	f := 1 - age/youngAge
	return 1 + 3*f, 1 + 0.6*f
}

// starFormingLight is the young light of a gas particle of density rho and ionized fraction ion in
// units of an old star particle of the same mass (0 below the threshold or when ionized).
func (p Population) starFormingLight(rho, ion float64) float64 {
	if p.SFLight == 0 || rho <= float64(p.SFDensity) || ion >= 0.5 {
		return 0
	}
	// Faded in above the threshold, so the glow does not flicker with the density of a particle.
	t := math.Min(1, rho/float64(p.SFDensity)-1)
	return math.Min(maxStarFormingLight, float64(p.SFLight)*math.Sqrt(rho)) * t * t * (3 - 2*t)
}
