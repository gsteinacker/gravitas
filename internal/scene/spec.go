package scene

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"math/rand/v2"
	"os"
	"path"
	"slices"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// A Spec is a scene recipe, stored as JSON (see scenes/*.json and docs/scenes.md). Building it for
// a particle budget n gives the initial state. Angles are in degrees, everything else in simulation
// units (G = 1) unless a field says otherwise.
type Spec struct {
	Name       string            `json:"name"`
	Info       string            `json:"info,omitempty"`            // shown in the HUD, e.g. the physical units
	Intro      map[string]string `json:"intro,omitempty"`           // introduction per audience: school, lay, physics
	Topics     []string          `json:"topics,omitempty"`          // related topics of the learning mode (internal/learn)
	YearsPer   float64           `json:"yearsPerUnit,omitempty"`    // physical years per simulation time unit
	KmPerS     float64           `json:"velocityUnit,omitempty"`    // km/s per simulation velocity unit (gas temperatures)
	GasFloorK  float64           `json:"gasFloorK,omitempty"`       // cooling floor temperature in K (sets params.gas.uFloor)
	IonK       float64           `json:"ionTemperatureK,omitempty"` // temperature of photoionized gas in K (sets params.gas.uIon)
	Params     sim.Params        `json:"params"`
	Camera     *CameraSpec       `json:"camera,omitempty"`
	Effects    string            `json:"effects,omitempty"` // render effects to switch on, e.g. "glow,trails"
	BulkRest   bool              `json:"removeBulkMotion,omitempty"`
	Components []Component       `json:"components"`
}

// CameraSpec is the initial view.
type CameraSpec struct {
	Target   [3]float64 `json:"target"`
	Distance float64    `json:"distance"`
	Pitch    float64    `json:"pitch"` // degrees, 90 = top-down
	Yaw      float64    `json:"yaw"`   // degrees
}

// Component is one part of a scene; Type selects which fields apply.
type Component struct {
	Type     string     `json:"type"` // galaxy | plummer | hole | orbits | star | gasbox | gasclump
	Name     string     `json:"name,omitempty"`
	Note     string     `json:"note,omitempty"`   // free text: sources, assumptions
	Weight   float64    `json:"weight,omitempty"` // share of the particle budget; the last weighted component gets the rest
	Mass     float64    `json:"mass"`
	Position [3]float64 `json:"position"`
	Velocity [3]float64 `json:"velocity"`
	Seed     uint64     `json:"seed,omitempty"`

	// galaxy: exponential disk (+ optional dark matter halo, + optional central hole)
	DiskFraction float64 `json:"diskFraction,omitempty"` // share of Mass in the disk; 0 means 1 (no halo)
	DiskScale    float64 `json:"diskScale,omitempty"`
	DiskMin      float64 `json:"diskMin,omitempty"`
	DiskMax      float64 `json:"diskMax,omitempty"`
	Thickness    float64 `json:"thickness,omitempty"`
	Dispersion   float64 `json:"dispersion,omitempty"`
	ToomreQ      float64 `json:"toomreQ,omitempty"`        // > 0: dispersions from the Toomre criterion
	ColdFraction float64 `json:"coldFraction,omitempty"`   // young, cold population tracing spiral arms
	GasFraction  float64 `json:"gasFraction,omitempty"`    // share of the disk in gas (needs toomreQ)
	GasK         float64 `json:"gasTemperature,omitempty"` // initial gas temperature in K
	GasScale     float64 `json:"gasScale,omitempty"`       // scale length of the gas disk (0: diskScale)
	GasMax       float64 `json:"gasMax,omitempty"`         // outer radius of the gas disk (0: diskMax)
	HaloScale    float64 `json:"haloScale,omitempty"`      // scale radius of the halo
	HaloProfile  string  `json:"haloProfile,omitempty"`    // "plummer" (default) or "hernquist"
	HaloCut      float64 `json:"haloCut,omitempty"`        // plummer: halo mass fraction kept (0 means 0.99)
	HaloMax      float64 `json:"haloMax,omitempty"`        // hernquist: truncation radius (0 means 10 haloScale)
	DarkFraction float64 `json:"darkFraction,omitempty"`   // share of the galaxy's particles in the halo
	BulgeFrac    float64 `json:"bulgeFraction,omitempty"`  // share of the stellar mass (Mass·diskFraction) in a Hernquist bulge
	BulgeScale   float64 `json:"bulgeScale,omitempty"`     // its scale radius
	HoleMass     float64 `json:"holeMass,omitempty"`
	Inclination  float64 `json:"inclination,omitempty"` // degrees, around x
	Orientation  float64 `json:"orientation,omitempty"` // degrees, around z afterwards

	// plummer: star cluster
	Scale float64 `json:"scale,omitempty"`

	// star: a single star, a source of ionizing radiation if Luminosity > 0
	Luminosity float64 `json:"luminosity,omitempty"`

	// gasbox: uniform gas in a box of Size (full extents) around Position, with density
	// fluctuations of relative amplitude Perturb; gasbox and gasclump use GasK and Turbulence (km/s).
	Size       [3]float64 `json:"size,omitempty"`
	Perturb    float64    `json:"perturb,omitempty"`
	Turbulence float64    `json:"turbulence,omitempty"`

	// orbits: Keplerian orbits around a central hole of Mass at Position
	AScale     float64    `json:"aScale,omitempty"` // multiplies the semi-major axes (unit conversion)
	Orbits     []Orbit    `json:"orbits,omitempty"`
	StarMass   float64    `json:"starMass,omitempty"`
	BackMass   float64    `json:"backgroundMass,omitempty"` // mass of background stars (0: StarMass)
	Background int        `json:"background,omitempty"`     // additional random orbits (capped by the budget)
	BackA      [2]float64 `json:"backgroundA,omitempty"`    // their semi-major axis range (log-uniform)
}

// Orbit holds Keplerian elements; angles in degrees, Anomaly is the mean anomaly.
type Orbit struct {
	Name    string  `json:"name"`
	A       float64 `json:"a"`
	E       float64 `json:"e"`
	I       float64 `json:"i"`
	Node    float64 `json:"node"`
	Peri    float64 `json:"peri"`
	Anomaly float64 `json:"anomaly"`
}

const deg = math.Pi / 180

func v3(a [3]float64) vec.Vec3 { return vec.Vec3{X: a[0], Y: a[1], Z: a[2]} }

// ParseSpec reads a scene recipe.
func ParseSpec(data []byte) (*Spec, error) {
	var s Spec
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Params.Dt <= 0 {
		return nil, fmt.Errorf("scene %q: params.dt must be positive", s.Name)
	}
	for i, c := range s.Components {
		switch c.Type {
		case "galaxy", "plummer", "hole", "orbits", "star", "gasbox", "gasclump":
			if p := c.HaloProfile; p != "" && p != "plummer" && p != "hernquist" {
				return nil, fmt.Errorf("scene %q: component %d: unknown halo profile %q", s.Name, i, p)
			}
		default:
			return nil, fmt.Errorf("scene %q: component %d: unknown type %q", s.Name, i, c.Type)
		}
	}
	return &s, nil
}

// LoadSpec reads a scene recipe from a file.
func LoadSpec(file string) (*Spec, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return ParseSpec(data)
}

// Build creates the initial state for about n particles.
func (sp *Spec) Build(n int) (*sim.State, sim.Params) {
	p := sp.Params
	if sp.GasFloorK > 0 {
		p.Gas.UFloor = sp.specificEnergy(sp.GasFloorK)
	}
	if sp.IonK > 0 {
		p.Gas.UIon = sp.specificEnergy(sp.IonK)
	}
	counts := sp.counts(n)
	s := sim.NewState(n + len(sp.Components))
	for i, c := range sp.Components {
		switch c.Type {
		case "galaxy":
			addGalaxy(s, c, counts[i], p, sp.specificEnergy(c.GasK))
		case "plummer":
			AddPlummer(s, counts[i], c.Mass, c.Scale, v3(c.Position), v3(c.Velocity), c.Seed)
		case "hole":
			s.Add(sim.Body{Pos: v3(c.Position), Vel: v3(c.Velocity), Mass: c.Mass, Kind: sim.BlackHole})
		case "orbits":
			addOrbits(s, c, counts[i])
		case "star":
			s.Add(sim.Body{Pos: v3(c.Position), Vel: v3(c.Velocity), Mass: c.Mass, Lum: c.Luminosity})
		case "gasbox", "gasclump":
			addGasCloud(s, c, counts[i], sp.specificEnergy(c.GasK), c.Turbulence/math.Max(sp.KmPerS, 1e-30))
		}
	}
	if sp.BulkRest {
		RemoveBulkMotion(s)
	}
	return s, p
}

// pcPerKmSYr is one km/s · yr in parsec.
const pcPerKmSYr = 1.0 / 977_792.2

// Units returns the physical scale of the scene: km/s per velocity unit and parsec per length unit
// (L = v · t); ok is false for scenes without physical units.
func (sp *Spec) Units() (kmPerS, pcPerUnit float64, ok bool) {
	if sp.KmPerS <= 0 || sp.YearsPer <= 0 {
		return 0, 0, false
	}
	return sp.KmPerS, sp.KmPerS * sp.YearsPer * pcPerKmSYr, true
}

// SpecificEnergy converts a gas temperature in K into specific internal energy in simulation units
// (0 for scenes without velocityUnit).
func (sp *Spec) SpecificEnergy(kelvin float64) float64 { return sp.specificEnergy(kelvin) }

// MassUnit is the mass unit of the scene in solar masses, M = v² L / G; ok is false without units.
func (sp *Spec) MassUnit() (msun float64, ok bool) {
	v, l, ok := sp.Units()
	if !ok {
		return 0, false
	}
	const g = 4.30091e-3 // pc (km/s)² / M☉
	return v * v * l / g, true
}

// AddGasClump adds a gas cloud of n particles with a Plummer profile of the given mass and scale
// radius at pos moving with vel: specific internal energy u, turbulent velocity dispersion
// turbulence (simulation units per component) and density fluctuations of relative amplitude
// perturb (as the scene component "gasclump").
func AddGasClump(s *sim.State, n int, mass, scale float64, pos, vel vec.Vec3, u, turbulence, perturb float64, seed uint64) {
	c := Component{
		Type: "gasclump", Mass: mass, Scale: scale, Seed: seed, Perturb: perturb,
		Position: [3]float64{pos.X, pos.Y, pos.Z}, Velocity: [3]float64{vel.X, vel.Y, vel.Z},
		Size: [3]float64{4 * scale, 4 * scale, 4 * scale},
	}
	addGasCloud(s, c, n, u, turbulence)
}

// specificEnergy converts a gas temperature in K into specific internal energy in simulation
// units: u = 3 k T / (2 μ m_p), μ = 0.6 for ionized (T ≥ 8000 K) and 1.27 for neutral gas.
func (sp *Spec) specificEnergy(kelvin float64) float64 {
	if kelvin <= 0 || sp.KmPerS <= 0 {
		return 0
	}
	const kB, mp = 1.380649e-23, 1.67262192e-27
	mu := 1.27
	if kelvin >= 8000 {
		mu = 0.6
	}
	u := 1.5 * kB * kelvin / (mu * mp) / 1e6 // (km/s)²
	return u / (sp.KmPerS * sp.KmPerS)
}

// DustOpacity returns the visual (V band) optical depth of neutral gas per unit column density in
// simulation units, or 0 if the scene does not define its units. The length unit follows from
// velocityUnit · yearsPerUnit, the mass unit from G = 1; the column is converted to hydrogen
// atoms (mass fraction X = 0.71) and to extinction with N_H / A_V = 1.9·10²¹ cm⁻² (Bohlin et al.
// 1978), τ_V = A_V / 1.086. This assumes the Milky Way dust-to-gas ratio.
func (sp *Spec) DustOpacity() float64 {
	if sp.KmPerS <= 0 || sp.YearsPer <= 0 {
		return 0
	}
	const g, yr, mp, x, nhPerAV = 6.674e-8, 3.15576e7, 1.67262e-24, 0.71, 1.9e21
	v := sp.KmPerS * 1e5             // cm/s
	l := v * sp.YearsPer * yr        // cm
	sigma := v * v * l / g / (l * l) // g/cm² per unit column
	return sigma * x / mp / nhPerAV / 1.086
}

// counts splits n by weight; the last weighted component takes the remainder.
func (sp *Spec) counts(n int) []int {
	counts := make([]int, len(sp.Components))
	last, used := -1, 0
	for i, c := range sp.Components {
		if c.Weight > 0 {
			counts[i] = int(c.Weight * float64(n))
			used += counts[i]
			last = i
		}
	}
	if last >= 0 {
		counts[last] += n - used
	}
	return counts
}

// bulgeParticleMass is the mean bulge particle mass relative to the disk's (masses 0.1–1 vs 0.1–10).
const bulgeParticleMass = 0.5

func addGalaxy(s *sim.State, c Component, n int, p sim.Params, gasU float64) {
	diskFrac := c.DiskFraction
	if diskFrac == 0 {
		diskFrac = 1
	}
	diskMass, haloMass := c.Mass*diskFrac, c.Mass*(1-diskFrac)
	nDark := 0
	if haloMass > 0 {
		nDark = int(float64(n) * c.DarkFraction)
	}
	hernquist := c.HaloProfile == "hernquist"
	haloMax := c.HaloMax
	if haloMax == 0 {
		haloMax = 10 * c.HaloScale
	}
	bulgeMass := diskMass * c.BulgeFrac
	diskMass -= bulgeMass
	nStars := n - nDark
	nBulge := 0
	if bulgeMass > 0 {
		// Bulge stars are lighter (old population): bulgeParticleMass times the disk's mean.
		wb := bulgeMass / bulgeParticleMass
		nBulge = int(float64(nStars) * wb / (wb + diskMass))
	}
	dc := DiskConfig{
		N: nStars - nBulge, StarMass: diskMass, BHMass: c.HoleMass,
		HaloHernquist: hernquist, HaloMax: haloMax,
		ScaleRadius: c.DiskScale, MinRadius: c.DiskMin, MaxRadius: c.DiskMax,
		Thickness: c.Thickness, Softening: p.Softening, Dispersion: c.Dispersion,
		Q: c.ToomreQ, ColdFraction: c.ColdFraction, GasFraction: c.GasFraction, GasU: gasU,
		GasScale: c.GasScale, GasMax: c.GasMax,
		HaloMass: haloMass, HaloScale: c.HaloScale, BulgeMass: bulgeMass, BulgeScale: c.BulgeScale,
		Center: v3(c.Position), Velocity: v3(c.Velocity),
		Inclination: c.Inclination * deg, Orientation: c.Orientation * deg, Seed: c.Seed,
	}
	AddDisk(s, dc)
	AddBulge(s, nBulge, dc)
	if nDark > 0 && hernquist {
		AddHernquistHalo(s, nDark, haloMass, c.HaloScale, haloMax, v3(c.Position), v3(c.Velocity), c.Seed+1000)
	} else if nDark > 0 {
		cut := c.HaloCut
		if cut == 0 {
			cut = 0.99
		}
		AddHalo(s, nDark, haloMass, c.HaloScale, v3(c.Position), v3(c.Velocity), c.Seed+1000, cut)
	}
}

// addGasCloud adds a box (gasbox) or Plummer sphere (gasclump) of equal-mass gas particles.
func addGasCloud(s *sim.State, c Component, n int, u, turbulence float64) {
	if n <= 0 {
		return
	}
	rng := rand.New(rand.NewPCG(c.Seed, 0x6a5))
	center, vel := v3(c.Position), v3(c.Velocity)
	// Density fluctuations: plane waves with wavelengths of 15–60 % of the box width, applied by
	// rejection sampling.
	type mode struct{ kx, ky, kz, phase float64 }
	var modes []mode
	if c.Perturb > 0 {
		for range 8 {
			k := 2 * math.Pi / ((0.15 + 0.45*rng.Float64()) * math.Max(c.Size[0], 1e-9))
			d := randomDirection(rng)
			modes = append(modes, mode{k * d.X, k * d.Y, k * d.Z, 2 * math.Pi * rng.Float64()})
		}
	}
	density := func(p vec.Vec3) float64 {
		f := 1.0
		for _, md := range modes {
			f += c.Perturb / float64(len(modes)) * 2 * math.Sin(md.kx*p.X+md.ky*p.Y+md.kz*p.Z+md.phase)
		}
		return math.Max(0, f)
	}
	maxDensity := 1 + 2*c.Perturb
	for range n {
		var p vec.Vec3
		for {
			if c.Type == "gasclump" {
				r := c.Scale / math.Sqrt(math.Pow(rng.Float64()*0.99+1e-6, -2.0/3.0)-1)
				p = randomDirection(rng).Scale(r)
			} else {
				p = vec.Vec3{X: (rng.Float64() - 0.5) * c.Size[0], Y: (rng.Float64() - 0.5) * c.Size[1], Z: (rng.Float64() - 0.5) * c.Size[2]}
			}
			if rng.Float64()*maxDensity < density(p) {
				break
			}
		}
		v := vec.Vec3{X: rng.NormFloat64(), Y: rng.NormFloat64(), Z: rng.NormFloat64()}.Scale(turbulence)
		s.Add(sim.Body{Pos: p.Add(center), Vel: v.Add(vel), Mass: c.Mass / float64(n), Kind: sim.Gas, U: u})
	}
}

// addOrbits places a central hole and stars on the given (plus n random background) Keplerian
// orbits around it.
func addOrbits(s *sim.State, c Component, n int) {
	center, vel := v3(c.Position), v3(c.Velocity)
	if c.Mass > 0 {
		s.Add(sim.Body{Pos: center, Vel: vel, Mass: c.Mass, Kind: sim.BlackHole})
	}
	scale := c.AScale
	if scale == 0 {
		scale = 1
	}
	add := func(o Orbit, mass float64) {
		p, v := keplerState(c.Mass, o.A*scale, o.E, o.I*deg, o.Node*deg, o.Peri*deg, o.Anomaly*deg)
		s.Add(sim.Body{Pos: p.Add(center), Vel: v.Add(vel), Mass: mass})
	}
	for _, o := range c.Orbits {
		add(o, c.StarMass)
	}
	backMass := c.BackMass
	if backMass == 0 {
		backMass = c.StarMass
	}
	rng := rand.New(rand.NewPCG(c.Seed, 0x0b17))
	lo, hi := math.Log(c.BackA[0]), math.Log(c.BackA[1])
	for range min(c.Background, max(0, n-len(c.Orbits))) {
		add(Orbit{
			A:       math.Exp(lo+(hi-lo)*rng.Float64()) / scale,
			E:       math.Sqrt(rng.Float64()) * 0.95, // thermal distribution, capped
			I:       math.Acos(2*rng.Float64()-1) / deg,
			Node:    360 * rng.Float64(),
			Peri:    360 * rng.Float64(),
			Anomaly: 360 * rng.Float64(),
		}, backMass)
	}
}

// keplerState converts Keplerian elements around mass m into position and velocity.
func keplerState(m, a, e, inc, node, peri, meanAnomaly float64) (vec.Vec3, vec.Vec3) {
	E := meanAnomaly
	for range 50 { // Newton iteration for Kepler's equation E − e sin E = M
		E -= (E - e*math.Sin(E) - meanAnomaly) / (1 - e*math.Cos(E))
	}
	sinE, cosE := math.Sincos(E)
	nMotion := math.Sqrt(m / (a * a * a))
	b := math.Sqrt(1 - e*e)
	f := nMotion * a / (1 - e*cosE)
	p := vec.Vec3{X: a * (cosE - e), Y: a * b * sinE}
	v := vec.Vec3{X: -f * sinE, Y: f * b * cosE}
	rot := func(x vec.Vec3) vec.Vec3 { return x.RotateZ(peri).RotateX(inc).RotateZ(node) }
	return rot(p), rot(v)
}

// Builtin returns the scenes embedded in the binary, ordered by file name.
func Builtin() ([]*Spec, error) {
	names, err := fs.Glob(builtinFS, "scenes/*.json")
	if err != nil {
		return nil, err
	}
	slices.Sort(names)
	var specs []*Spec
	for _, name := range names {
		data, err := builtinFS.ReadFile(name)
		if err != nil {
			return nil, err
		}
		sp, err := ParseSpec(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path.Base(name), err)
		}
		specs = append(specs, sp)
	}
	return specs, nil
}

//go:embed scenes/*.json
var builtinFS embed.FS

// templateGalaxy is the galaxy every placed galaxy is modeled on: the Milky Way-like disk of the
// built-in spiral galaxy scene (disk, bulge, gas, Hernquist halo, central hole).
func templateGalaxy() (Component, *Spec) {
	for _, p := range Presets {
		for _, c := range p.Spec.Components {
			if c.Type == "galaxy" && p.Spec.Name == "Spiral galaxy" {
				return c, p.Spec
			}
		}
	}
	panic("scene: no template galaxy") // built-in scenes, covered by tests
}

// TemplateGalaxyMsun is the total mass of the template galaxy in solar masses.
func TemplateGalaxyMsun() float64 {
	c, t := templateGalaxy()
	mu, _ := t.MassUnit()
	return c.Mass * mu
}

// AddTemplateGalaxy adds n bodies of a galaxy like the template with total mass msun (solar
// masses), built in the units of scene sp at the origin at rest, with its disk in the xy plane
// spinning around +z. Sizes scale with M^(1/3) (same mean density). It returns false for scenes
// without physical units.
func (sp *Spec) AddTemplateGalaxy(s *sim.State, n int, msun float64, seed uint64) bool {
	c, t := templateGalaxy()
	tMass, _ := t.MassUnit()
	_, tLen, _ := t.Units()
	mUnit, okM := sp.MassUnit()
	_, lUnit, okL := sp.Units()
	if !okM || !okL {
		return false
	}
	m := msun / (c.Mass * tMass)     // mass factor
	l := math.Cbrt(m) * tLen / lUnit // length factor: template units → scene units
	c.Mass = msun / mUnit
	c.HoleMass *= m * tMass / mUnit
	for _, f := range []*float64{&c.DiskScale, &c.DiskMin, &c.DiskMax, &c.Thickness, &c.HaloScale, &c.HaloMax,
		&c.BulgeScale, &c.GasScale, &c.GasMax} {
		*f *= l
	}
	c.Position, c.Velocity, c.Inclination, c.Orientation, c.Seed = [3]float64{}, [3]float64{}, 0, 0, seed
	p := sp.Params
	p.Softening = math.Max(p.Softening, 0.02*c.DiskScale)
	addGalaxy(s, c, n, p, sp.specificEnergy(c.GasK))
	return true
}
