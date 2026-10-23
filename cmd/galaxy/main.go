// Command galaxy runs the interactive star system simulation.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"galaxy/internal/render"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
	"galaxy/internal/sim/direct"
	"galaxy/internal/ui"
)

func main() {
	n := flag.Int("n", 50000, "number of stars")
	solverName := flag.String("solver", "bh", "force solver: bh (Barnes-Hut) or direct")
	useGPU := flag.Bool("gpu", true, "run the Barnes-Hut tree walk on the GPU if available (macOS/Metal)")
	theta := flag.Float64("theta", bh.DefaultTheta, "Barnes-Hut opening angle (smaller = more accurate)")
	preset := flag.Int("preset", 1, "built-in start scene (1-based)")
	sceneFile := flag.String("scene", "", "start scene from a JSON file (see docs/scenes.md)")
	loadFile := flag.String("load", "", "start from a saved state (.galaxy, written with F5)")
	shot := flag.String("shot", "", "save a screenshot to this PNG file after -shot-after seconds and quit")
	shotAfter := flag.Float64("shot-after", 4, "seconds before the screenshot is taken")
	speedOfLight := flag.Float64("c", 0, "speed of light (0 = scene default, -1 = Newtonian)")
	gravity := flag.String("gravity", "instant", "propagation of black hole gravity: instant, retarded, retarded+v")
	effects := flag.String("fx", "", "render effects (glow, lens, doppler, trails, dark); default: the scene's or glow,lens,doppler")
	cpuprofile := flag.String("cpuprofile", "", "write a CPU profile of the whole run to this file")
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
	p := max(0, min(len(scene.Presets)-1, *preset-1))

	spec := scene.Presets[p].Spec
	if *sceneFile != "" {
		var err error
		if spec, err = scene.LoadSpec(*sceneFile); err != nil {
			log.Fatal(err)
		}
	}
	s, params := spec.Build(*n)
	mode, err := sim.ParseGravityMode(*gravity)
	if err != nil {
		log.Fatal(err)
	}
	params.Gravity = mode
	switch {
	case *speedOfLight > 0:
		params.C = *speedOfLight
	case *speedOfLight < 0:
		params.C = 0
	}
	if *useGPU {
		if err := bh.GPUAvailable(); err != nil {
			log.Printf("GPU tree walk unavailable, using the CPU: %v", err)
			*useGPU = false
		}
	}
	var solver sim.Solver = &bh.Solver{Theta: *theta, GPU: *useGPU}
	switch *solverName {
	case "bh":
	case "direct":
		solver = &direct.Solver{}
	default:
		log.Fatalf("unknown solver %q", *solverName)
	}
	runner := sim.NewRunner(sim.New(s, solver, params))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)

	ebiten.SetWindowTitle("Galaxy")
	ebiten.SetWindowSize(1440, 900)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	game, err := ui.NewGame(runner, *n, spec)
	if err != nil {
		log.Fatal(err)
	}
	if *loadFile != "" {
		if err := game.Load(*loadFile); err != nil {
			log.Fatal(err)
		}
	}
	if *effects != "" {
		fx, err := render.ParseEffects(*effects)
		if err != nil {
			log.Fatal(err)
		}
		game.SetEffects(fx)
	}
	game.ShotPath, game.ShotAfter = *shot, time.Duration(*shotAfter*float64(time.Second))
	if err := ebiten.RunGame(game); err != nil {
		pprof.StopCPUProfile()
		log.Fatal(err)
	}
}
