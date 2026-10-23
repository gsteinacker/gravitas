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
	"github.com/hajimehoshi/ebiten/v2/vector"

	"galaxy/internal/render"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// speedLevels are the selectable simulation step rates; 0 means unlimited.
var speedLevels = []int{10, 30, 60, 120, 240, 0}

// clickThreshold is how far (pixels) the mouse may move for a press to count as a click.
const clickThreshold = 4

// tool is what a click or Alt-drag places.
type tool int

const (
	toolBlackHole tool = iota
	toolCluster
)

func (t tool) String() string {
	if t == toolCluster {
		return "star cluster"
	}
	return "black hole"
}

// clusterStars is the number of stars in a placed cluster for a scene of n stars.
func clusterStars(n int) int { return max(300, n/40) }

// clusterRadius is the Plummer scale radius of a cluster of the given mass: denser for light
// clusters, so they stay visibly compact.
func clusterRadius(mass float64) float64 { return 0.2 * math.Cbrt(mass/0.5) }

// launchScale converts an Alt-drag distance in world units into a launch velocity.
// Dragging across a typical galaxy radius (~2) gives roughly its orbital speed (~1).
const launchScale = 0.5

// launchVelocity is the velocity of a black hole launched by dragging from screen point
// (x0, y0) to (x1, y1). Both points are taken in the camera's focal plane.
func launchVelocity(cam *render.Camera, x0, y0, x1, y1 float64) vec.Vec3 {
	return cam.Unproject(x1, y1).Sub(cam.Unproject(x0, y0)).Scale(launchScale)
}

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
	bhMass   float64 // mass of the next placed object
	showHUD  bool
	showHelp bool

	dragging           bool
	pressX, pressY     int
	lastX, lastY       int
	movedSincePress    bool
	launching          bool // Alt-drag: place a black hole with an initial velocity
	chars              []rune
	tool               tool
	placements         uint64 // seeds successive clusters differently
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

// handleKeys reacts to typed characters rather than physical key positions, so the bindings
// work on any keyboard layout (ebiten.Key constants follow the US layout, e.g. KeyBracketLeft
// is "Ü" on a German keyboard). Only layout-independent keys are read physically.
func (g *Game) handleKeys() {
	g.chars = ebiten.AppendInputChars(g.chars[:0])
	for _, r := range g.chars {
		g.applyChar(r)
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		g.runner.SetPaused(!g.runner.Paused())
	case inpututil.IsKeyJustPressed(ebiten.KeyF1):
		g.showHelp = !g.showHelp
	}
	g.runner.SetTargetSPS(speedLevels[g.speed])
}

// applyChar executes the binding for one typed character.
func (g *Game) applyChar(r rune) {
	switch r {
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		if i := int(r - '1'); i < len(scene.Presets) {
			g.LoadPreset(i)
		}
	case '.':
		g.runner.SetPaused(true)
		g.runner.Do(func(m *sim.Simulation) { m.Step() })
	case '+', '=':
		g.speed = min(len(speedLevels)-1, g.speed+1)
	case '-':
		g.speed = max(0, g.speed-1)
	case 'm', 'M', ']':
		g.bhMass = math.Min(20, g.bhMass*1.5)
	case 'n', 'N', '[':
		g.bhMass = math.Max(0.01, g.bhMass/1.5)
	case 'b', 'B':
		g.tool = toolBlackHole
	case 'k', 'K':
		g.tool = toolCluster
	case '\t':
		g.tool = (g.tool + 1) % 2
	case 'r', 'R':
		g.LoadPreset(g.preset)
	case 'h', 'H':
		g.showHUD = !g.showHUD
	case '?':
		g.showHelp = !g.showHelp
	case 'c', 'C':
		g.cam.Target = g.cam.Target.Scale(0)
	case 'f', 'F':
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
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
			g.launching = b == ebiten.MouseButtonLeft && ebiten.IsKeyPressed(ebiten.KeyAlt)
		}
	}
	if !g.dragging {
		return
	}
	if abs(x-g.pressX) > clickThreshold || abs(y-g.pressY) > clickThreshold {
		g.movedSincePress = true
	}
	dx, dy := float64(x-g.lastX), float64(y-g.lastY)
	if g.movedSincePress && !g.launching {
		if g.button == ebiten.MouseButtonLeft && !ebiten.IsKeyPressed(ebiten.KeyShift) {
			g.cam.Rotate(dx, dy)
		} else {
			g.cam.Pan(dx, dy)
		}
	}
	g.lastX, g.lastY = x, y
	if inpututil.IsMouseButtonJustReleased(g.button) {
		g.dragging = false
		switch {
		case g.launching:
			v := launchVelocity(&g.cam, float64(g.pressX), float64(g.pressY), float64(x), float64(y))
			g.place(g.pressX, g.pressY, v)
		case !g.movedSincePress && g.button == ebiten.MouseButtonLeft:
			g.place(x, y, vec.Vec3{})
		}
		g.launching = false
	}
}

// place adds the current tool's object at screen position (x, y) with velocity vel.
func (g *Game) place(x, y int, vel vec.Vec3) {
	pos := g.cam.Unproject(float64(x), float64(y))
	mass := g.bhMass
	if g.tool == toolBlackHole {
		g.runner.Do(func(m *sim.Simulation) {
			m.AddBody(sim.Body{Pos: pos, Vel: vel, Mass: mass, Kind: sim.BlackHole})
		})
		return
	}
	// Build the cluster here, so the simulation goroutine only appends.
	g.placements++
	n := clusterStars(g.n)
	c := sim.NewState(n)
	scene.AddPlummer(c, n, mass, clusterRadius(mass), pos, vel, 1000+g.placements)
	g.runner.Do(func(m *sim.Simulation) {
		for i := range c.Len() {
			m.AddBody(c.Body(i))
		}
	})
}

// drawLaunchPreview shows the pending Alt-drag launch as an arrow with its speed.
func (g *Game) drawLaunchPreview(screen *ebiten.Image) {
	x, y := ebiten.CursorPosition()
	x0, y0, x1, y1 := float32(g.pressX), float32(g.pressY), float32(x), float32(y)
	orange := color.RGBA{255, 150, 50, 255}
	vector.StrokeLine(screen, x0, y0, x1, y1, 2, orange, true)
	vector.StrokeCircle(screen, x0, y0, 7, 2, orange, true)
	if dx, dy := x1-x0, y1-y0; dx*dx+dy*dy > 25 {
		l := float32(math.Hypot(float64(dx), float64(dy)))
		ux, uy := dx/l, dy/l
		for _, side := range []float32{1, -1} {
			vector.StrokeLine(screen, x1, y1, x1-10*ux+6*side*uy, y1-10*uy-6*side*ux, 2, orange, true)
		}
	}
	v := launchVelocity(&g.cam, float64(g.pressX), float64(g.pressY), float64(x), float64(y))
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("v = %.2f", v.Len()), x+12, y+8)
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
	if g.dragging && g.launching {
		g.drawLaunchPreview(screen)
	}
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
		"%s%s\nbodies %d (black holes %d)   t = %.2f\nsteps/s %.0f (limit %s)   FPS %.0f\n%s\n\ntool: %s, mass %.2f   (b/k/tab, m/n)",
		scene.Presets[g.preset].Name, state, g.bodies, g.blackHoles, g.simTime,
		g.runner.StepsPerSecond(), speed, ebiten.ActualFPS(), driftText(g.runner.EnergyDrift()),
		g.tool, g.bhMass), 10, 8)
	if g.showHelp {
		var b strings.Builder
		for i, p := range scene.Presets {
			fmt.Fprintf(&b, "%d  %s\n", i+1, p.Name)
		}
		b.WriteString("\nb k tab     tool: black hole / star cluster\nclick       place\nalt+drag    launch with velocity\nm n         mass +/-\n" +
			"drag        rotate\nshift/right drag  pan\nwheel       zoom\n" +
			"space       pause\n.           single step\n+ -         speed\nr           restart\nc           center view\n" +
			"f           fullscreen\nh           hide HUD\n? F1        hide this help")
		ebitenutil.DebugPrintAt(screen, b.String(), 10, g.screenH-16*strings.Count(b.String(), "\n")-28)
	}
}

// driftText formats the energy drift; "~" marks a tree estimate, which is only good to ±30 %.
func driftText(d float64, q sim.EnergyQuality) string {
	switch {
	case q == sim.EnergyUnknown || math.IsNaN(d):
		return "energy drift: measuring..."
	case q == sim.EnergyEstimated:
		return fmt.Sprintf("energy drift ~%+.0e (tree estimate)", d)
	default:
		return fmt.Sprintf("energy drift %+.1e", d)
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
