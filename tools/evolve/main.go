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
	"sort"
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
	out := flag.String("o", "evolved.gravitas", "save file to write")
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
		substeps += m.HydroSubsteps
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

// substeps sums the gas evaluations of all steps, for the mean in report.
var substeps int

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
	fmt.Printf("t=%.3f steps=%d (%.1f/s, %.1f gas evaluations) gas=%d ionized=%d formed=%d sn=%d infall=%.3g hmin=%.4f rhomax=%.0f rmax=%.1f\n",
		m.Time, steps, float64(steps)/el, float64(substeps)/float64(max(steps, 1)), gas, ion, m.StarsFormed, m.Supernovae, m.InfallMass, hmin, rhomax, rmax)
	_ = stars
	h, r50, rz := shape(s)
	fmt.Printf("  baryons: half-mass radius %.3g, rms height %.3g, flattening %.2f (height/radius)\n", r50, h, rz)
	if b := sim.MeasureBar(s); b.Radius > 0 {
		fmt.Printf("  bar: A2 %.3f, phase %.0f°, measured within %.3g\n", b.A2, b.Phase*180/math.Pi, b.Radius)
	}
}

// shape measures the baryons (stars and gas) about their center of mass and angular momentum
// axis: half-mass radius, rms height above the plane and their ratio (small for a disk).
func shape(s *sim.State) (height, r50, flat float64) {
	var m, cx, cy, cz, vx, vy, vz float64
	for i := range s.Len() {
		if k := s.Kind[i]; k == sim.Star || k == sim.Gas {
			w := s.Mass[i]
			m += w
			cx, cy, cz = cx+w*s.PosX[i], cy+w*s.PosY[i], cz+w*s.PosZ[i]
			vx, vy, vz = vx+w*s.VelX[i], vy+w*s.VelY[i], vz+w*s.VelZ[i]
		}
	}
	if m == 0 {
		return 0, 0, 0
	}
	cx, cy, cz, vx, vy, vz = cx/m, cy/m, cz/m, vx/m, vy/m, vz/m
	type b struct{ r, z, m float64 }
	var lx, ly, lz float64
	for i := range s.Len() {
		if k := s.Kind[i]; k == sim.Star || k == sim.Gas {
			x, y, z := s.PosX[i]-cx, s.PosY[i]-cy, s.PosZ[i]-cz
			u, v, w := s.VelX[i]-vx, s.VelY[i]-vy, s.VelZ[i]-vz
			lx, ly, lz = lx+s.Mass[i]*(y*w-z*v), ly+s.Mass[i]*(z*u-x*w), lz+s.Mass[i]*(x*v-y*u)
		}
	}
	l := math.Sqrt(lx*lx + ly*ly + lz*lz)
	if l == 0 {
		return 0, 0, 0
	}
	lx, ly, lz = lx/l, ly/l, lz/l
	var bs []b
	for i := range s.Len() {
		if k := s.Kind[i]; k == sim.Star || k == sim.Gas {
			x, y, z := s.PosX[i]-cx, s.PosY[i]-cy, s.PosZ[i]-cz
			h := x*lx + y*ly + z*lz
			bs = append(bs, b{math.Sqrt(math.Max(0, x*x+y*y+z*z-h*h)), h, s.Mass[i]})
		}
	}
	sort.Slice(bs, func(i, j int) bool { return bs[i].r < bs[j].r })
	var acc, h2, mm float64
	for _, x := range bs {
		acc += x.m
		if acc >= m/2 && r50 == 0 {
			r50 = x.r
		}
		if r50 == 0 || x.r <= 2*r50 { // inner part: within twice the half-mass radius
			h2, mm = h2+x.m*x.z*x.z, mm+x.m
		}
	}
	height = math.Sqrt(h2 / mm)
	return height, r50, height / r50
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
