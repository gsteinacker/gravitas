package ui

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"galaxy/internal/learn"
	"galaxy/internal/render"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/direct"
	"galaxy/internal/vec"
)

func TestLaunchVelocityFollowsDragInFocalPlane(t *testing.T) {
	cam := render.NewCamera()
	cam.W, cam.H, cam.Pitch = 1000, 800, math.Pi/2 // top-down: screen right = +x, screen up = +y
	v := launchVelocity(&cam, 500, 400, 600, 400)
	if v.X <= 0 || math.Abs(v.Y) > 1e-9 || math.Abs(v.Z) > 1e-9 {
		t.Errorf("drag to the right should launch along +x, got %v", v)
	}
	dist := cam.Unproject(600, 400).Sub(cam.Unproject(500, 400)).Len()
	if math.Abs(v.Len()-launchScale*dist) > 1e-12 {
		t.Errorf("speed %v, want %v", v.Len(), launchScale*dist)
	}
	if v := launchVelocity(&cam, 300, 300, 300, 300); v.Len() != 0 {
		t.Errorf("zero drag should give zero velocity, got %v", v)
	}
}

func TestCharBindingsWorkWithoutUSLayout(t *testing.T) {
	g := &Game{speed: 2, bhMass: 1}
	// Characters as typed on a German keyboard: dedicated "+" and "-" keys, "m"/"n" for mass.
	for _, r := range "++-mmn" {
		g.applyChar(r)
	}
	if g.speed != 3 {
		t.Errorf("speed level = %d, want 3", g.speed)
	}
	if math.Abs(g.bhMass-1.5) > 1e-12 {
		t.Errorf("black hole mass = %v, want 1.5", g.bhMass)
	}
	g.applyChar('?')
	g.applyChar('h')
	if !g.showHelp || !g.showHUD {
		t.Errorf("help %v, HUD %v; want both toggled on", g.showHelp, g.showHUD)
	}
}

func TestToolSelection(t *testing.T) {
	g := &Game{}
	for r, want := range map[rune]tool{'k': toolCluster, 'b': toolBlackHole, 'K': toolCluster} {
		g.applyChar(r)
		if g.tool != want {
			t.Errorf("after %q tool = %v, want %v", r, g.tool, want)
		}
	}
	g.tool = toolBlackHole
	g.applyChar('\t')
	if g.tool != toolCluster {
		t.Errorf("tab should cycle to %v, got %v", toolCluster, g.tool)
	}
}

func TestEffectToggles(t *testing.T) {
	g := &Game{}
	for _, r := range "gtld" {
		g.applyChar(r)
	}
	if !g.fx.Glow || !g.fx.Trails || !g.fx.Lensing || !g.fx.Doppler {
		t.Errorf("all effects should be on after g t l d: %+v", g.fx)
	}
	if got := effectsText(g.fx); got != "effects:*glow(g)*trails(t)*lens(l)*doppler(d) dust(u) gas map(o)" {
		t.Errorf("effectsText = %q", got)
	}
}

func TestDriftText(t *testing.T) {
	if got := driftText(1.234e-3, sim.EnergyExact); got != "energy drift +1.2e-03" {
		t.Errorf("exact: %q", got)
	}
	if got := driftText(-4e-4, sim.EnergyEstimated); got != "energy drift ~-4e-04 (tree estimate)" {
		t.Errorf("estimated: %q", got)
	}
	if got := driftText(math.NaN(), sim.EnergyExact); got != "energy drift: measuring..." {
		t.Errorf("unknown: %q", got)
	}
}

func TestPlaceClusterAddsBoundGroupAtCursor(t *testing.T) {
	s := sim.NewState(1)
	s.Add(sim.Body{Mass: 1, Kind: sim.BlackHole})
	m := sim.New(s, &direct.Solver{}, sim.Params{Dt: 1e-3, Softening: 0.05})
	r := sim.NewRunner(m)
	r.SetPaused(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	g, err := NewGame(r, 1000, scene.Presets[0].Spec)
	if err != nil {
		t.Fatal(err)
	}
	g.cam.W, g.cam.H, g.cam.Pitch = 1000, 800, math.Pi/2
	g.tool, g.bhMass = toolCluster, 0.5
	g.place(700, 400, vec.Vec3{Y: 0.3})
	want := 1 + clusterStars(1000)

	deadline := time.Now().Add(2 * time.Second)
	for {
		var n int
		var cx float64
		r.View(func(sn *sim.Snapshot) {
			n = sn.Len()
			for i := 1; i < n; i++ {
				cx += float64(sn.X[i]*sn.Mass[i]) / g.bhMass
			}
		})
		if n == want {
			center := g.cam.Unproject(700, 400)
			if math.Abs(cx-center.X) > 0.1 {
				t.Errorf("cluster center x = %.3f, want ≈ %.3f", cx, center.X)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("snapshot has %d bodies, want %d", n, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestFormatYears(t *testing.T) {
	for y, want := range map[float64]string{2.5: "2.5 yr", 47200: "47.2 kyr", 13.2e6: "13.2 Myr", 1.09e9: "1.09 Gyr"} {
		if got := formatYears(y); got != want {
			t.Errorf("formatYears(%v) = %q, want %q", y, got, want)
		}
	}
}

func TestSaveAndLoadThroughTheGame(t *testing.T) {
	sp := scene.Presets[2].Spec
	s, p := sp.Build(500)
	r := sim.NewRunner(sim.New(s, &direct.Solver{}, p))
	r.SetPaused(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)
	g, err := NewGame(r, 500, sp)
	if err != nil {
		t.Fatal(err)
	}
	g.cam.Yaw = 1.25
	path := filepath.Join(t.TempDir(), "s.galaxy")
	g.Save(path)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("save file was not written")
		}
		time.Sleep(10 * time.Millisecond)
	}
	g.cam.Yaw = 0
	g.LoadPreset(0) // something else in between
	if err := g.Load(path); err != nil {
		t.Fatal(err)
	}
	if g.cam.Yaw != 1.25 || g.scene.Name != sp.Name {
		t.Errorf("camera %v / scene %q not restored", g.cam.Yaw, g.scene.Name)
	}
	waitLen := func(want int) {
		for time.Now().Before(deadline) {
			n := 0
			r.View(func(sn *sim.Snapshot) { n = sn.Len() })
			if n == want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("simulation has not been restored to %d bodies", want)
	}
	waitLen(500)
}

func TestWrapKeepsLinesWithinWidth(t *testing.T) {
	s := "Alles, was Masse hat, zieht sich gegenseitig an: die Erde dich, die Sonne die Erde.\n\nNeuer Absatz mit Umlauten äöü und θ."
	lines := wrap(s, bodyFace, 200)
	if len(lines) < 4 {
		t.Fatalf("expected several lines, got %q", lines)
	}
	for _, l := range lines {
		if w, _ := text.Measure(l, bodyFace, 0); w > 200 && strings.Contains(l, " ") {
			t.Errorf("line %q is %.0f px wide", l, w)
		}
	}
	if !slices.Contains(lines, "") {
		t.Error("paragraph break must produce an empty line")
	}
}

func TestTopicListStartsWithIntroAndSceneTopics(t *testing.T) {
	list := topicList([]string{"dust", "gas"})
	if list[0] != introTopic || list[1] != "dust" || list[2] != "gas" || len(list) != 1+len(learn.IDs()) {
		t.Errorf("unexpected topic list %v", list)
	}
}

func TestHintFocusesTopic(t *testing.T) {
	g := &Game{scene: scene.Presets[0].Spec}
	g.topics = topicList(g.scene.Topics)
	g.applyChar('u')
	g.hint(hintFor['u'])
	if g.topics[g.topicIdx] != "dust" {
		t.Errorf("focused %q, want dust", g.topics[g.topicIdx])
	}
	if n := g.message.Load(); n == nil || !strings.Contains(n.text, "Staub") {
		t.Errorf("hint not shown: %+v", n)
	}
	title, body := g.currentExplanation()
	if title != "Staub" || body == "" {
		t.Errorf("explanation %q / %q", title, body)
	}
}
