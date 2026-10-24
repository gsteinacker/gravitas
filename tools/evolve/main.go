// Command evolve runs a scene headless for a given simulated time and writes a save file that
// the app can open with -load (for inspecting long-term evolution, e.g. of the Horsehead scene).
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"runtime/pprof"
	"time"

	"galaxy/internal/save"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
	"galaxy/internal/sph"
)

func main() {
	n := flag.Int("n", 60000, "number of bodies")
	preset := flag.Int("preset", 9, "built-in scene (1-based)")
	until := flag.Float64("t", 0.5, "simulation time to reach")
	every := flag.Float64("every", 0, "also report progress every this much simulation time")
	out := flag.String("o", "evolved.galaxy", "save file to write")
	in := flag.String("in", "", "continue from this save file instead of building the scene")
	yaw := flag.Float64("yaw", math.NaN(), "camera yaw in degrees (default: the scene's)")
	pitch := flag.Float64("pitch", math.NaN(), "camera pitch in degrees (default: the scene's)")
	dist := flag.Float64("dist", math.NaN(), "camera distance (default: the scene's)")
	cpuprofile := flag.String("cpuprofile", "", "write a CPU profile to this file")
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	spec := scene.Presets[max(0, min(len(scene.Presets)-1, *preset-1))].Spec
	s, params := spec.Build(*n)
	m := sim.New(s, &bh.Solver{GPU: bh.GPUAvailable() == nil}, params)
	m.Hydro = &sph.Solver{}
	if *in != "" {
		f, err := save.Read(*in)
		if err != nil {
			log.Fatal(err)
		}
		f.Restore(m)
		spec = &f.Scene
	}
	start, next := time.Now(), *every
	steps := 0
	for m.Time < *until {
		m.Step()
		steps++
		if i := firstBad(m.State); i >= 0 {
			s := m.State
			fmt.Printf("step %d t=%.5f: body %d kind %d pos (%g %g %g) vel (%g %g %g) acc (%g %g %g) u %g h %g rho %g ion %g m %g\n",
				steps, m.Time, i, s.Kind[i], s.PosX[i], s.PosY[i], s.PosZ[i], s.VelX[i], s.VelY[i], s.VelZ[i], s.AccX[i], s.AccY[i], s.AccZ[i], s.U[i], s.H[i], s.Rho[i], s.Ion[i], s.Mass[i])
			for h := range s.Len() {
				if s.Kind[h] == sim.BlackHole {
					dx, dy, dz := s.PosX[h]-s.PosX[i], s.PosY[h]-s.PosY[i], s.PosZ[h]-s.PosZ[i]
					fmt.Printf("  hole %d m %g at distance %g, speed %g\n", h, s.Mass[h], math.Sqrt(dx*dx+dy*dy+dz*dz),
						math.Sqrt(s.VelX[h]*s.VelX[h]+s.VelY[h]*s.VelY[h]+s.VelZ[h]*s.VelZ[h]))
				}
			}
			os.Exit(1)
		}
		if *every > 0 && m.Time >= next {
			next += *every
			report(m, steps, start)
		}
	}
	report(m, steps, start)
	cam := save.Camera{Distance: 10, Pitch: 0.4}
	if c := spec.Camera; c != nil {
		cam = save.Camera{Target: c.Target, Distance: c.Distance, Pitch: c.Pitch * math.Pi / 180, Yaw: c.Yaw * math.Pi / 180}
	}
	if !math.IsNaN(*yaw) {
		cam.Yaw = *yaw * math.Pi / 180
	}
	if !math.IsNaN(*pitch) {
		cam.Pitch = *pitch * math.Pi / 180
	}
	if !math.IsNaN(*dist) {
		cam.Distance = *dist
	}
	f := save.Capture(m, spec, cam)
	if err := save.Write(*out, f); err != nil {
		log.Fatal(err)
	}
}

func report(m *sim.Simulation, steps int, start time.Time) {
	s := m.State
	var gas, ion, stars int
	hmin, rhomax, rmax := math.Inf(1), 0.0, 0.0
	for i := range s.Len() {
		switch {
		case s.Kind[i] == sim.Gas:
			gas++
			hmin, rhomax = min(hmin, s.H[i]), max(rhomax, s.Rho[i])
			rmax = max(rmax, math.Abs(s.PosX[i]), math.Abs(s.PosY[i]), math.Abs(s.PosZ[i]))
			if s.Ion[i] > 0.5 {
				ion++
			}
		case s.Kind[i] == sim.Star && !(s.Born[i] < -1e300):
			stars++
		}
	}
	el := time.Since(start).Seconds()
	fmt.Printf("t=%.3f steps=%d (%.1f/s) gas=%d ionized=%d formed=%d hmin=%.4f rhomax=%.0f rmax=%.1f\n",
		m.Time, steps, float64(steps)/el, gas, ion, m.StarsFormed, hmin, rhomax, rmax)
	_ = stars
}

// firstBad returns the first body with a non-finite or absurd state, or -1.
func firstBad(s *sim.State) int {
	bad := func(v float64) bool { return math.IsNaN(v) || math.Abs(v) > 1e6 }
	for i := range s.Len() {
		if bad(s.PosX[i]) || bad(s.VelX[i]) || bad(s.U[i]) || bad(s.H[i]) || bad(s.AccX[i]) {
			return i
		}
	}
	return -1
}
