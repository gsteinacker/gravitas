package scene

import (
	"math"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// Preset is a named starting scene for roughly n stars.
type Preset struct {
	Name  string
	Build func(n int) (*sim.State, sim.Params)
}

var defaultParams = sim.Params{Dt: 0.004, Softening: 0.06}

var Presets = []Preset{
	{"Spiral galaxy", func(n int) (*sim.State, sim.Params) {
		s := sim.NewState(n + 1)
		AddDisk(s, DiskConfig{
			N: n, StarMass: 1, BHMass: 2, ScaleRadius: 0.8, MinRadius: 0.15, MaxRadius: 4,
			Thickness: 0.03, Softening: defaultParams.Softening, Dispersion: 0.12, Inclination: 0, Seed: 1,
		})
		return s, defaultParams
	}},
	{"Galaxy collision", func(n int) (*sim.State, sim.Params) {
		s := sim.NewState(n + 2)
		AddDisk(s, DiskConfig{
			N: n * 3 / 5, StarMass: 0.6, BHMass: 1.2, ScaleRadius: 0.7, MinRadius: 0.15, MaxRadius: 3,
			Thickness: 0.03, Softening: defaultParams.Softening, Dispersion: 0.12,
			Center: vec.Vec3{X: -4, Y: -1.2}, Velocity: vec.Vec3{X: 0.32},
			Inclination: 0.3, Seed: 2,
		})
		AddDisk(s, DiskConfig{
			N: n - n*3/5, StarMass: 0.4, BHMass: 0.8, ScaleRadius: 0.6, MinRadius: 0.15, MaxRadius: 2.5,
			Thickness: 0.03, Softening: defaultParams.Softening, Dispersion: 0.12,
			Center: vec.Vec3{X: 4, Y: 1.2}, Velocity: vec.Vec3{X: -0.32},
			Inclination: -1.1, Orientation: 0.8, Seed: 3,
		})
		RemoveBulkMotion(s)
		return s, defaultParams
	}},
	{"Star cluster", func(n int) (*sim.State, sim.Params) {
		s := sim.NewState(n)
		AddPlummer(s, n, 1, 1, vec.Vec3{}, vec.Vec3{}, 4)
		RemoveBulkMotion(s)
		return s, defaultParams
	}},
	{"Black hole binary in a cluster", func(n int) (*sim.State, sim.Params) {
		s := sim.NewState(n + 2)
		AddPlummer(s, n, 1, 1.2, vec.Vec3{}, vec.Vec3{}, 5)
		// Two equal black holes on a circular orbit of separation d.
		const m, d = 0.5, 0.8
		v := math.Sqrt(m / (2 * d))
		s.Add(sim.Body{Pos: vec.Vec3{X: -d / 2}, Vel: vec.Vec3{Y: -v}, Mass: m, Kind: sim.BlackHole})
		s.Add(sim.Body{Pos: vec.Vec3{X: d / 2}, Vel: vec.Vec3{Y: v}, Mass: m, Kind: sim.BlackHole})
		RemoveBulkMotion(s)
		return s, defaultParams
	}},
}
