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
	Name       string      `json:"name"`
	Info       string      `json:"info,omitempty"`         // shown in the HUD, e.g. the physical units
	YearsPer   float64     `json:"yearsPerUnit,omitempty"` // physical years per simulation time unit
	Params     sim.Params  `json:"params"`
	Camera     *CameraSpec `json:"camera,omitempty"`
	Effects    string      `json:"effects,omitempty"` // render effects to switch on, e.g. "glow,trails"
	BulkRest   bool        `json:"removeBulkMotion,omitempty"`
	Components []Component `json:"components"`
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
	Type     string     `json:"type"` // galaxy | plummer | hole | orbits
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
	HaloScale    float64 `json:"haloScale,omitempty"`    // scale radius of the halo
	HaloProfile  string  `json:"haloProfile,omitempty"`  // "plummer" (default) or "hernquist"
	HaloCut      float64 `json:"haloCut,omitempty"`      // plummer: halo mass fraction kept (0 means 0.99)
	HaloMax      float64 `json:"haloMax,omitempty"`      // hernquist: truncation radius (0 means 10 haloScale)
	DarkFraction float64 `json:"darkFraction,omitempty"` // share of the galaxy's particles in the halo
	HoleMass     float64 `json:"holeMass,omitempty"`
	Inclination  float64 `json:"inclination,omitempty"` // degrees, around x
	Orientation  float64 `json:"orientation,omitempty"` // degrees, around z afterwards

	// plummer: star cluster
	Scale float64 `json:"scale,omitempty"`

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
		case "galaxy", "plummer", "hole", "orbits":
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
	counts := sp.counts(n)
	s := sim.NewState(n + len(sp.Components))
	for i, c := range sp.Components {
		switch c.Type {
		case "galaxy":
			addGalaxy(s, c, counts[i], sp.Params)
		case "plummer":
			AddPlummer(s, counts[i], c.Mass, c.Scale, v3(c.Position), v3(c.Velocity), c.Seed)
		case "hole":
			s.Add(sim.Body{Pos: v3(c.Position), Vel: v3(c.Velocity), Mass: c.Mass, Kind: sim.BlackHole})
		case "orbits":
			addOrbits(s, c, counts[i])
		}
	}
	if sp.BulkRest {
		RemoveBulkMotion(s)
	}
	return s, sp.Params
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

func addGalaxy(s *sim.State, c Component, n int, p sim.Params) {
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
	AddDisk(s, DiskConfig{
		N: n - nDark, StarMass: diskMass, BHMass: c.HoleMass,
		HaloHernquist: hernquist, HaloMax: haloMax,
		ScaleRadius: c.DiskScale, MinRadius: c.DiskMin, MaxRadius: c.DiskMax,
		Thickness: c.Thickness, Softening: p.Softening, Dispersion: c.Dispersion,
		HaloMass: haloMass, HaloScale: c.HaloScale,
		Center: v3(c.Position), Velocity: v3(c.Velocity),
		Inclination: c.Inclination * deg, Orientation: c.Orientation * deg, Seed: c.Seed,
	})
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
