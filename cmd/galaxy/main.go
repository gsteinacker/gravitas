// Command galaxy runs the interactive star system simulation.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/direct"
	"galaxy/internal/ui"
)

func main() {
	n := flag.Int("n", 8000, "number of stars")
	preset := flag.Int("preset", 1, "start scene (1-based)")
	shot := flag.String("shot", "", "save a screenshot to this PNG file after -shot-after seconds and quit")
	shotAfter := flag.Float64("shot-after", 4, "seconds before the screenshot is taken")
	flag.Parse()
	p := max(0, min(len(scene.Presets)-1, *preset-1))

	s, params := scene.Presets[p].Build(*n)
	runner := sim.NewRunner(sim.New(s, &direct.Solver{}, params))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)

	ebiten.SetWindowTitle("Galaxy")
	ebiten.SetWindowSize(1440, 900)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	game := ui.NewGame(runner, *n, p)
	game.ShotPath, game.ShotAfter = *shot, int(*shotAfter*60)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
