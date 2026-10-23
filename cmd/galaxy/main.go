// Command galaxy runs the interactive star system simulation.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/hajimehoshi/ebiten/v2"

	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
	"galaxy/internal/sim/direct"
	"galaxy/internal/ui"
)

func main() {
	n := flag.Int("n", 50000, "number of stars")
	solverName := flag.String("solver", "bh", "force solver: bh (Barnes-Hut) or direct")
	theta := flag.Float64("theta", bh.DefaultTheta, "Barnes-Hut opening angle (smaller = more accurate)")
	preset := flag.Int("preset", 1, "start scene (1-based)")
	shot := flag.String("shot", "", "save a screenshot to this PNG file after -shot-after seconds and quit")
	shotAfter := flag.Float64("shot-after", 4, "seconds before the screenshot is taken")
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

	s, params := scene.Presets[p].Build(*n)
	var solver sim.Solver = &bh.Solver{Theta: *theta}
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
	game := ui.NewGame(runner, *n, p)
	game.ShotPath, game.ShotAfter = *shot, int(*shotAfter*60)
	if err := ebiten.RunGame(game); err != nil {
		pprof.StopCPUProfile()
		log.Fatal(err)
	}
}
