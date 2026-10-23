// Package ui wires input, camera, rendering and the simulation runner into an Ebitengine game.
package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"galaxy/internal/render"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
)

// speedLevels are the selectable simulation step rates; 0 means unlimited.
var speedLevels = []int{10, 30, 60, 120, 240, 0}

// clickThreshold is how far (pixels) the mouse may move for a press to count as a click.
const clickThreshold = 4

type Game struct {
	// ShotPath, if set, saves a screenshot after ShotAfter frames and quits.
	ShotPath  string
	ShotAfter int
	frame     int
	shotErr   error
	shotDone  bool

	runner   *sim.Runner
	cam      render.Camera
	stars    *render.StarRenderer
	n        int
	preset   int
	speed    int
	bhMass   float64
	showHUD  bool
	showHelp bool

	dragging           bool
	pressX, pressY     int
	lastX, lastY       int
	movedSincePress    bool
	button             ebiten.MouseButton
	screenW, screenH   int
	bodies, blackHoles int
	simTime            float64
}

func NewGame(runner *sim.Runner, n, preset int) *Game {
	return &Game{
		runner: runner, cam: render.NewCamera(), stars: render.NewStarRenderer(),
		n: n, preset: preset, speed: len(speedLevels) - 1, bhMass: 0.5,
		showHUD: true, showHelp: true,
	}
}

// LoadPreset replaces the running scene.
func (g *Game) LoadPreset(i int) {
	g.preset = i
	s, p := scene.Presets[i].Build(g.n)
	g.runner.Do(func(m *sim.Simulation) { m.Reset(s, p) })
}

func (g *Game) Update() error {
	if g.shotDone {
		if g.shotErr != nil {
			return g.shotErr
		}
		return ebiten.Termination
	}
	g.handleKeys()
	g.handleMouse()
	return nil
}

func (g *Game) handleKeys() {
	for i := range scene.Presets {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			g.LoadPreset(i)
		}
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		g.runner.SetPaused(!g.runner.Paused())
	case inpututil.IsKeyJustPressed(ebiten.KeyR):
		g.LoadPreset(g.preset)
	case inpututil.IsKeyJustPressed(ebiten.KeyH):
		g.showHUD = !g.showHUD
	case inpututil.IsKeyJustPressed(ebiten.KeyF1), inpututil.IsKeyJustPressed(ebiten.KeySlash):
		g.showHelp = !g.showHelp
	case inpututil.IsKeyJustPressed(ebiten.KeyEqual), inpututil.IsKeyJustPressed(ebiten.KeyKPAdd):
		g.speed = min(len(speedLevels)-1, g.speed+1)
	case inpututil.IsKeyJustPressed(ebiten.KeyMinus), inpututil.IsKeyJustPressed(ebiten.KeyKPSubtract):
		g.speed = max(0, g.speed-1)
	case inpututil.IsKeyJustPressed(ebiten.KeyBracketRight):
		g.bhMass = math.Min(20, g.bhMass*1.5)
	case inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft):
		g.bhMass = math.Max(0.01, g.bhMass/1.5)
	case inpututil.IsKeyJustPressed(ebiten.KeyC):
		g.cam.Target = g.cam.Target.Scale(0)
	case inpututil.IsKeyJustPressed(ebiten.KeyF):
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	g.runner.SetTargetSPS(speedLevels[g.speed])
}

func (g *Game) handleMouse() {
	if _, wy := ebiten.Wheel(); wy != 0 {
		g.cam.Zoom(math.Pow(0.9, wy))
	}
	x, y := ebiten.CursorPosition()
	for _, b := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight} {
		if !g.dragging && inpututil.IsMouseButtonJustPressed(b) {
			g.dragging, g.button = true, b
			g.pressX, g.pressY, g.lastX, g.lastY = x, y, x, y
			g.movedSincePress = false
		}
	}
	if !g.dragging {
		return
	}
	if abs(x-g.pressX) > clickThreshold || abs(y-g.pressY) > clickThreshold {
		g.movedSincePress = true
	}
	dx, dy := float64(x-g.lastX), float64(y-g.lastY)
	if g.movedSincePress {
		if g.button == ebiten.MouseButtonLeft && !ebiten.IsKeyPressed(ebiten.KeyShift) {
			g.cam.Rotate(dx, dy)
		} else {
			g.cam.Pan(dx, dy)
		}
	}
	g.lastX, g.lastY = x, y
	if inpututil.IsMouseButtonJustReleased(g.button) {
		g.dragging = false
		if !g.movedSincePress && g.button == ebiten.MouseButtonLeft {
			g.placeBlackHole(x, y)
		}
	}
}

func (g *Game) placeBlackHole(x, y int) {
	pos := g.cam.Unproject(float64(x), float64(y))
	mass := g.bhMass
	g.runner.Do(func(m *sim.Simulation) {
		m.AddBody(sim.Body{Pos: pos, Mass: mass, Kind: sim.BlackHole})
	})
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{2, 2, 8, 255})
	g.cam.W, g.cam.H = float64(g.screenW), float64(g.screenH)
	g.runner.View(func(sn *sim.Snapshot) {
		g.stars.Draw(screen, &g.cam, sn)
		g.bodies, g.blackHoles, g.simTime = sn.Len(), 0, sn.Time
		for _, k := range sn.Kind {
			if k == sim.BlackHole {
				g.blackHoles++
			}
		}
	})
	if g.showHUD {
		g.drawHUD(screen)
	}
	g.frame++
	if g.ShotPath != "" && !g.shotDone && g.frame >= g.ShotAfter {
		g.shotErr, g.shotDone = saveScreenshot(screen, g.ShotPath), true
	}
}

func saveScreenshot(screen *ebiten.Image, path string) error {
	img := image.NewRGBA(screen.Bounds())
	screen.ReadPixels(img.Pix)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (g *Game) drawHUD(screen *ebiten.Image) {
	speed := "max"
	if s := speedLevels[g.speed]; s > 0 {
		speed = fmt.Sprintf("%d/s", s)
	}
	state := ""
	if g.runner.Paused() {
		state = "  [PAUSED]"
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf(
		"%s%s\nbodies %d (black holes %d)   t = %.2f\nsteps/s %.0f (limit %s)   FPS %.0f\nnext black hole mass %.2f",
		scene.Presets[g.preset].Name, state, g.bodies, g.blackHoles, g.simTime,
		g.runner.StepsPerSecond(), speed, ebiten.ActualFPS(), g.bhMass), 10, 8)
	if g.showHelp {
		var b strings.Builder
		for i, p := range scene.Presets {
			fmt.Fprintf(&b, "%d  %s\n", i+1, p.Name)
		}
		b.WriteString("\nclick       place black hole\n[ ]         black hole mass\n" +
			"drag        rotate\nshift/right drag  pan\nwheel       zoom\n" +
			"space       pause\n+ -         speed\nr           restart\nc           center view\n" +
			"f           fullscreen\nh           hide HUD\n?           hide this help")
		ebitenutil.DebugPrintAt(screen, b.String(), 10, g.screenH-16*strings.Count(b.String(), "\n")-28)
	}
}

func (g *Game) Layout(w, h int) (int, int) {
	g.screenW, g.screenH = w, h
	return w, h
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
