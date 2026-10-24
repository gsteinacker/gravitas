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
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"galaxy/internal/learn"
	"galaxy/internal/render"
	"galaxy/internal/save"
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

// cStep is the factor by which '<' and '>' change the speed of light; cMin/cMax bound it.
const (
	cStep = 1.25
	cMin  = 1.0
	cMax  = 1000.0
)

// scaleC changes the speed of light of the running simulation. Stored proper velocities (and
// thus momenta) stay; coordinate velocities follow from the new c.
func (g *Game) scaleC(f float64) {
	g.runner.Do(func(m *sim.Simulation) {
		if m.Params.C > 0 {
			m.Params.C = math.Max(cMin, math.Min(cMax, m.Params.C*f))
			m.Epoch++ // kinetic energy depends on c
		}
	})
}

// launchScale converts an Alt-drag distance in world units into a launch velocity.
// Dragging across a typical galaxy radius (~2) gives roughly its orbital speed (~1).
const launchScale = 0.5

// launchVelocity is the velocity of a black hole launched by dragging from screen point
// (x0, y0) to (x1, y1). Both points are taken in the camera's focal plane.
func launchVelocity(cam *render.Camera, x0, y0, x1, y1 float64) vec.Vec3 {
	return cam.Unproject(x1, y1).Sub(cam.Unproject(x0, y0)).Scale(launchScale)
}

type Game struct {
	// ShotPath, if set, saves a screenshot ShotAfter after the first frame and quits. Wall-clock
	// time, not frames: at low frame rates a frame count would take arbitrarily long.
	ShotPath  string
	ShotAfter time.Duration
	started   time.Time
	frame     int
	shotErr   error
	shotDone  bool

	runner   *sim.Runner
	cam      render.Camera
	pipeline *render.Pipeline
	fx       render.Effects
	n        int
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
	c                  float64
	dark               int
	gas, starsFormed   int
	darkOff            bool
	scene              *scene.Spec
	message            atomic.Pointer[note]

	// Learning mode (explain.go).
	explain     bool
	level       learn.Level
	topics      []string
	topicIdx    int
	hinted      map[string]bool
	substeps    int
	gasSubsteps int
	gravity     sim.GravityMode
}

// SetEffects replaces the active render effects.
func (g *Game) SetEffects(fx render.Effects) { g.fx = fx }

// SaveDir is where F5 stores and F9 looks for saved states.
const SaveDir = "saves"

// Save captures the running simulation on its goroutine and writes it in the background.
func (g *Game) Save(path string) {
	sp, cam := g.scene, g.saveCamera()
	g.runner.Do(func(m *sim.Simulation) {
		f := save.Capture(m, sp, cam)
		go func() {
			if err := save.Write(path, f); err != nil {
				g.notify("save failed: " + err.Error())
				return
			}
			g.notify("saved " + path)
		}()
	})
}

// Load replaces the running simulation by a saved state, including camera and scene texts.
func (g *Game) Load(path string) error {
	f, err := save.Read(path)
	if err != nil {
		return err
	}
	g.restore(f)
	g.notify("loaded " + path)
	return nil
}

func (g *Game) restore(f *save.File) {
	sp := f.Scene
	g.applyScene(&sp)
	g.cam.Target = vec.Vec3{X: f.Camera.Target[0], Y: f.Camera.Target[1], Z: f.Camera.Target[2]}
	g.cam.Yaw, g.cam.Pitch, g.cam.Distance = f.Camera.Yaw, f.Camera.Pitch, f.Camera.Distance
	g.runner.Do(f.Restore)
}

func (g *Game) saveCamera() save.Camera {
	c := g.cam
	return save.Camera{Target: [3]float64{c.Target.X, c.Target.Y, c.Target.Z}, Yaw: c.Yaw, Pitch: c.Pitch, Distance: c.Distance}
}

// notify shows a short message in the HUD; safe to call from any goroutine.
func (g *Game) notify(msg string) {
	g.message.Store(&note{text: msg, until: time.Now().Add(4 * time.Second)})
}

type note struct {
	text  string
	until time.Time
}

// defaultEffects are used for scenes that do not name their own.
var defaultEffects = render.Effects{Glow: true, Doppler: true, Lensing: true, Dust: true}

// NewGame creates the game for a runner that already simulates sp (built for n bodies).
func NewGame(runner *sim.Runner, n int, sp *scene.Spec) (*Game, error) {
	pipeline, err := render.NewPipeline()
	if err != nil {
		return nil, err
	}
	g := &Game{
		runner: runner, pipeline: pipeline,
		n: n, speed: len(speedLevels) - 1, bhMass: 0.5,
		showHUD: true, showHelp: true,
	}
	g.applyScene(sp)
	return g, nil
}

// applyScene takes over the camera, effects and HUD texts of a scene.
func (g *Game) applyScene(sp *scene.Spec) {
	g.scene = sp
	g.pipeline.Stars.DustOpacity = sp.DustOpacity()
	g.topics, g.topicIdx = topicList(sp.Topics), 0
	if len(sp.Intro) > 0 {
		g.notify(sp.Name + ": e öffnet eine Einführung")
	}
	g.cam = render.NewCamera()
	if c := sp.Camera; c != nil {
		g.cam.Target = vec.Vec3{X: c.Target[0], Y: c.Target[1], Z: c.Target[2]}
		g.cam.Distance, g.cam.Pitch, g.cam.Yaw = c.Distance, c.Pitch*math.Pi/180, c.Yaw*math.Pi/180
	}
	g.fx = defaultEffects
	if sp.Effects != "" {
		if fx, err := render.ParseEffects(sp.Effects); err == nil {
			g.fx = fx
		}
	}
}

// LoadPreset replaces the running scene by built-in scene i.
func (g *Game) LoadPreset(i int) { g.LoadScene(scene.Presets[i].Spec) }

// LoadScene replaces the running scene.
func (g *Game) LoadScene(sp *scene.Spec) {
	g.applyScene(sp)
	s, p := sp.Build(g.n)
	g.runner.Do(func(m *sim.Simulation) {
		// Propagation mode and dark matter switch are user choices, not part of the scene.
		p.Gravity, p.DarkMatterOff = m.Params.Gravity, m.Params.DarkMatterOff
		m.Reset(s, p)
	})
}

func (g *Game) Update() error {
	if g.shotDone {
		if g.shotErr != nil {
			return g.shotErr
		}
		return ebiten.Termination
	}
	if g.ShotPath != "" {
		return nil // screenshots must be reproducible: ignore stray input on the window
	}
	g.handleExplainKeys()
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
		if id, ok := hintFor[r]; ok {
			g.hint(id)
		}
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeySpace):
		g.runner.SetPaused(!g.runner.Paused())
	case inpututil.IsKeyJustPressed(ebiten.KeyF1):
		g.showHelp = !g.showHelp
	case inpututil.IsKeyJustPressed(ebiten.KeyF5):
		g.Save(save.NewPath(SaveDir))
	case inpututil.IsKeyJustPressed(ebiten.KeyF9):
		if path, err := save.Latest(SaveDir); err != nil {
			g.notify(err.Error())
		} else if err := g.Load(path); err != nil {
			g.notify(err.Error())
		}
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
	case '<':
		g.scaleC(1 / cStep)
	case '>':
		g.scaleC(cStep)
	case 'g', 'G':
		g.fx.Glow = !g.fx.Glow
	case 't', 'T':
		g.fx.Trails = !g.fx.Trails
	case 'l', 'L':
		g.fx.Lensing = !g.fx.Lensing
	case 'd', 'D':
		g.fx.Doppler = !g.fx.Doppler
	case 'p', 'P':
		g.runner.Do(func(m *sim.Simulation) {
			m.Params.Gravity = (m.Params.Gravity + 1) % 3
			m.Epoch++ // energy is not conserved in retarded modes; restart the drift measurement
		})
	case 'x', 'X':
		g.runner.Do(func(m *sim.Simulation) { m.SetDarkMatterOff(!m.Params.DarkMatterOff) })
	case 'v', 'V':
		g.fx.Dark = !g.fx.Dark
	case 'o', 'O':
		g.fx.Gas = !g.fx.Gas
	case 'u', 'U':
		g.fx.Dust = !g.fx.Dust
	case 'e', 'E':
		g.explain = !g.explain
	case 'r', 'R':
		g.LoadScene(g.scene)
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
	if g.hinted == nil {
		g.hinted = map[string]bool{}
	}
	if topic := map[tool]string{toolBlackHole: "black-holes", toolCluster: "gravity"}[g.tool]; !g.hinted[topic] {
		g.hinted[topic] = true // once per run, placing is frequent
		g.hint(topic)
	}
	if g.tool == toolBlackHole {
		g.runner.Do(func(m *sim.Simulation) {
			m.AddBody(sim.Body{Pos: pos, Vel: sim.ProperVelocity(vel, m.Params.C), Mass: mass, Kind: sim.BlackHole})
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
			b := c.Body(i)
			b.Vel = sim.ProperVelocity(b.Vel, m.Params.C)
			m.AddBody(b)
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
		g.pipeline.Draw(screen, &g.cam, sn, g.fx)
		g.bodies, g.blackHoles, g.simTime, g.c, g.substeps, g.gravity = sn.Len(), 0, sn.Time, sn.C, sn.Substeps, sn.Gravity
		g.gasSubsteps = sn.HydroSubsteps
		g.dark, g.darkOff, g.gas, g.starsFormed = 0, sn.DarkMatterOff, 0, sn.StarsFormed
		for _, k := range sn.Kind {
			switch k {
			case sim.BlackHole:
				g.blackHoles++
			case sim.DarkMatter:
				g.dark++
			case sim.Gas:
				g.gas++
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
	if g.started.IsZero() {
		g.started = time.Now()
	}
	if g.ShotPath != "" && !g.shotDone && time.Since(g.started) >= g.ShotAfter {
		g.shotErr, g.shotDone = saveScreenshot(screen, g.ShotPath), true
		// Machine-readable summary for the harness (the HUD text is only in the image).
		fmt.Printf("shot %s: bodies=%d fps=%.1f steps_per_s=%.1f substeps=%d t=%.2f\n",
			g.ShotPath, g.bodies, ebiten.ActualFPS(), g.runner.StepsPerSecond(), g.substeps, g.simTime)
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
	title := g.scene.Name + state
	if g.scene.Info != "" {
		title += "\n" + g.scene.Info
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf(
		"%s\nbodies %d (black holes %d)   %s\n%s\nsteps/s %.0f (limit %s)   substeps near holes %d, gas %d   FPS %.0f\n%s\n%s\n%s\n\ntool: %s, mass %.2f   (b/k/tab, m/n)",
		title, g.bodies, g.blackHoles, timeText(g.simTime, g.scene), darkText(g.dark, g.darkOff, g.fx.Dark)+gasText(g.gas, g.starsFormed, g.fx.Gas),
		g.runner.StepsPerSecond(), speed, g.substeps, max(1, g.gasSubsteps), ebiten.ActualFPS(), driftText(g.runner.EnergyDrift()),
		lightText(g.c, g.bhMass, g.gravity), effectsText(g.fx), g.tool, g.bhMass), 10, 8)
	if n := g.message.Load(); n != nil && time.Now().Before(n.until) {
		w, _ := text.Measure(n.text, bodyFace, 0)
		area := float64(g.screenW)
		if g.explain {
			area -= panelWidth(g.screenW) + 32 // keep the hint left of the open panel
		}
		x := math.Max(12, (area-w)/2)
		drawText(screen, n.text, bodyFace, x, float64(g.screenH)-48, accentColor)
	}
	if g.explain {
		g.drawExplanation(screen)
	}
	if g.showHelp {
		var b strings.Builder
		for i, p := range scene.Presets {
			fmt.Fprintf(&b, "%d  %s\n", i+1, p.Name)
		}
		b.WriteString("\nb k tab     tool: black hole / star cluster\nclick       place\nalt+drag    launch with velocity\nm n         mass +/-\n" +
			"drag        rotate\nshift/right drag  pan\nwheel       zoom\n" +
			"e           EXPLAIN: learning mode (arrows: topic, audience)\nx v         dark matter: gravity on/off, show\no u         gas map, dust absorption\n< >         speed of light\np           hole gravity: instant / retarded / retarded+v\ng t l d     glow, trails, lens, doppler\nspace       pause\n.           single step\n+ -         speed\nr           restart\nc           center view\n" +
			"F5 F9       save / load latest (saves/)\nf           fullscreen\nh           hide HUD\n? F1        hide this help")
		ebitenutil.DebugPrintAt(screen, b.String(), 10, g.screenH-16*strings.Count(b.String(), "\n")-28)
	}
}

// effectsText lists the render effects with their keys; active ones are marked with *.
func effectsText(fx render.Effects) string {
	mark := func(on bool, name string) string {
		if on {
			return "*" + name
		}
		return " " + name
	}
	return "effects:" + mark(fx.Glow, "glow(g)") + mark(fx.Trails, "trails(t)") +
		mark(fx.Lensing, "lens(l)") + mark(fx.Doppler, "doppler(d)") + mark(fx.Dust, "dust(u)") + mark(fx.Gas, "gas map(o)")
}

// lightText shows the speed of light, the horizon size of the next placed black hole and how the
// gravity of holes propagates.
func lightText(c, mass float64, gravity sim.GravityMode) string {
	if c <= 0 {
		return "c = infinite (Newtonian)"
	}
	note := ""
	if gravity != sim.GravityInstant {
		note = ", energy not conserved"
	}
	return fmt.Sprintf("c = %.1f  (< >)   r_s of mass %.2f = %.3f   hole gravity: %s (p)%s",
		c, mass, sim.SchwarzschildRadius(mass, c), gravity, note)
}

// timeText shows the simulation time, in physical units if the scene defines them.
func timeText(t float64, sp *scene.Spec) string {
	if sp.YearsPer <= 0 {
		return fmt.Sprintf("t = %.2f (no physical units)", t)
	}
	return fmt.Sprintf("t = %.2f (%s)", t, formatYears(t*sp.YearsPer))
}

// formatYears picks a readable unit: yr, kyr, Myr or Gyr.
func formatYears(y float64) string {
	switch a := math.Abs(y); {
	case a >= 1e9:
		return fmt.Sprintf("%.3g Gyr", y/1e9)
	case a >= 1e6:
		return fmt.Sprintf("%.3g Myr", y/1e6)
	case a >= 1e3:
		return fmt.Sprintf("%.3g kyr", y/1e3)
	default:
		return fmt.Sprintf("%.3g yr", y)
	}
}

// gasText describes the gas of the scene, if any.
func gasText(n, formed int, shown bool) string {
	if n == 0 && formed == 0 {
		return ""
	}
	vis := "hidden"
	if shown {
		vis = "shown"
	}
	return fmt.Sprintf("\ngas %d bodies, %d stars formed, %s (o)", n, formed, vis)
}

// darkText describes the dark matter of the scene, if any.
func darkText(n int, off, shown bool) string {
	if n == 0 {
		return "no dark matter"
	}
	state, vis := "on", "hidden"
	if off {
		state = "OFF (test particles)"
	}
	if shown {
		vis = "shown"
	}
	return fmt.Sprintf("dark matter %d bodies, gravity %s (x), %s (v)", n, state, vis)
}

// driftText formats the energy drift; "~" marks a tree estimate (error ~1e-6 since T-028).
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
