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
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/mac"
	"galaxy/internal/render"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/vec"
	"galaxy/internal/view"
)

// clickThreshold is how far (pixels) the mouse may move for a press to count as a click.
const clickThreshold = 4

// Game is the Ebitengine front end: it turns input into controller actions and draws the
// simulation and the HUD.
type Game struct {
	*app.Controller

	// ShotPath, if set, saves a screenshot ShotAfter after the first frame and quits. Wall-clock
	// time, not frames: at low frame rates a frame count would take arbitrarily long.
	ShotPath  string
	ShotAfter time.Duration
	started   time.Time
	frame     int
	shotErr   error
	shotDone  bool

	pipeline *render.Pipeline

	// NativeMenu installs the macOS menu bar on the first update (not in screenshot mode).
	NativeMenu bool
	// PrefsChanged runs when the settings window changed profile, stars or GPU.
	PrefsChanged func()
	installed    bool
	exportPath   string // save the next frame as PNG here

	dragging         bool
	pressX, pressY   int
	lastX, lastY     int
	movedSincePress  bool
	launching        bool // Alt-drag: place a black hole with an initial velocity
	chars            []rune
	button           ebiten.MouseButton
	screenW, screenH int
}

// SetEffects replaces the active render effects.
func (g *Game) SetEffects(fx view.Effects) { g.FX = fx }

// NewGame creates the game for a runner that already simulates sp (built for n bodies).
func NewGame(runner *sim.Runner, n int, sp *scene.Spec) (*Game, error) {
	pipeline, err := render.NewPipeline()
	if err != nil {
		return nil, err
	}
	g := &Game{pipeline: pipeline}
	g.Controller = app.New(runner, n, sp)
	g.Controller.SceneLoaded = func(sp *scene.Spec) { g.pipeline.Stars.DustOpacity = sp.DustOpacity() }
	g.Controller.ToggleFullscreen = func() { ebiten.SetFullscreen(!ebiten.IsFullscreen()) }
	g.pipeline.Stars.DustOpacity = sp.DustOpacity()
	return g, nil
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
	if g.NativeMenu && !g.installed {
		g.installed = true
		mac.Install(g.Controller, mac.Hooks{ExportImage: func(path string) { g.exportPath = path }, PrefsChanged: g.PrefsChanged})
	}
	g.Drain()
	g.handleKeys()
	g.handleMouse()
	g.Runner.SetTargetSPS(g.SpeedLimit())
	g.FPS = ebiten.ActualFPS()
	g.Publish()
	return nil
}

// handleKeys reacts to typed characters rather than physical key positions, so the bindings
// work on any keyboard layout (ebiten.Key constants follow the US layout, e.g. KeyBracketLeft
// is "Ü" on a German keyboard). Only layout-independent keys are read physically.
func (g *Game) handleKeys() {
	g.chars = ebiten.AppendInputChars(g.chars[:0])
	for _, r := range g.chars {
		if a, ok := app.ActionForChar(r); ok {
			g.Do(a)
		}
	}
	for k, a := range physicalKeys(g.Explain) {
		if inpututil.IsKeyJustPressed(k) {
			g.Do(a)
		}
	}
}

// physicalKeys are the layout-independent bindings; the arrows navigate the open learning panel.
func physicalKeys(explain bool) map[ebiten.Key]app.Action {
	if explain {
		return explainKeys
	}
	return globalKeys
}

var (
	globalKeys = map[ebiten.Key]app.Action{
		ebiten.KeySpace: app.ActionTogglePause, ebiten.KeyF1: app.ActionToggleHelp,
		ebiten.KeyF5: app.ActionQuickSave, ebiten.KeyF9: app.ActionQuickLoad,
	}
	explainKeys = map[ebiten.Key]app.Action{
		ebiten.KeySpace: app.ActionTogglePause, ebiten.KeyF1: app.ActionToggleHelp,
		ebiten.KeyF5: app.ActionQuickSave, ebiten.KeyF9: app.ActionQuickLoad,
		ebiten.KeyArrowRight: app.ActionNextTopic, ebiten.KeyArrowLeft: app.ActionPrevTopic,
		ebiten.KeyArrowDown: app.ActionNextLevel, ebiten.KeyArrowUp: app.ActionPrevLevel,
		ebiten.KeyEscape: app.ActionCloseExplain,
	}
)

func (g *Game) handleMouse() {
	if _, wy := ebiten.Wheel(); wy != 0 {
		g.Cam.Zoom(math.Pow(0.9, wy))
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
			g.Cam.Rotate(dx, dy)
		} else {
			g.Cam.Pan(dx, dy)
		}
	}
	g.lastX, g.lastY = x, y
	if inpututil.IsMouseButtonJustReleased(g.button) {
		g.dragging = false
		switch {
		case g.launching:
			v := app.LaunchVelocity(&g.Cam, float64(g.pressX), float64(g.pressY), float64(x), float64(y))
			g.Place(float64(g.pressX), float64(g.pressY), v)
		case !g.movedSincePress && g.button == ebiten.MouseButtonLeft:
			g.Place(float64(x), float64(y), vec.Vec3{})
		}
		g.launching = false
	}
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
	v := app.LaunchVelocity(&g.Cam, float64(g.pressX), float64(g.pressY), float64(x), float64(y))
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("v = %.2f", v.Len()), x+12, y+8)
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{2, 2, 8, 255})
	g.Cam.W, g.Cam.H = float64(g.screenW), float64(g.screenH)
	g.Runner.View(func(sn *sim.Snapshot) {
		g.pipeline.Draw(screen, &g.Cam, sn, g.FX)
		g.Observe(sn)
	})
	if g.dragging && g.launching {
		g.drawLaunchPreview(screen)
	}
	if g.exportPath != "" {
		// Before the HUD, so the image shows only the simulation.
		path := g.exportPath
		g.exportPath = ""
		if err := saveScreenshot(screen, path); err != nil {
			mac.Alert(i18n.T("Bild kann nicht gesichert werden", "Cannot save the image"), err.Error())
		} else {
			g.Notify(i18n.T("Bild gesichert: ", "Image saved: ") + path)
		}
	}
	g.drawHUD(screen)
	g.frame++
	if g.started.IsZero() {
		g.started = time.Now()
	}
	if g.ShotPath != "" && !g.shotDone && time.Since(g.started) >= g.ShotAfter {
		g.shotErr, g.shotDone = saveScreenshot(screen, g.ShotPath), true
		// Machine-readable summary for the harness (the HUD text is only in the image).
		s := g.Sim()
		fmt.Printf("shot %s: bodies=%d fps=%.1f steps_per_s=%.1f substeps=%d t=%.2f\n",
			g.ShotPath, s.Bodies, ebiten.ActualFPS(), g.Runner.StepsPerSecond(), s.Substeps, s.Time)
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

// drawHUD draws the status text (if shown), messages, the learning panel and the key help.
func (g *Game) drawHUD(screen *ebiten.Image) {
	speed := "max"
	if l := g.SpeedLimit(); l > 0 {
		speed = fmt.Sprintf("%d/s", l)
	}
	state := ""
	if g.Runner.Paused() {
		state = "  [PAUSED]"
	}
	title := g.Scene.Name + state
	if g.Scene.Info != "" {
		title += "\n" + g.Scene.Info
	}
	s := g.Sim()
	if g.ShowHUD {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf(
			"%s\nbodies %d (black holes %d)   %s\n%s\nsteps/s %.0f (limit %s)   substeps near holes %d, gas %d   FPS %.0f\n%s\n%s\n%s\n\ntool: %s, mass %.2f   (b/k/tab, m/n)",
			title, s.Bodies, s.BlackHoles, timeText(s.Time, g.Scene), darkText(s.Dark, s.DarkMatterOff, g.FX.Dark)+gasText(s.Gas, s.StarsFormed, g.FX.Gas),
			g.Runner.StepsPerSecond(), speed, s.Substeps, max(1, s.HydroSubsteps), ebiten.ActualFPS(), driftText(g.Runner.EnergyDrift()),
			lightText(s.C, g.Mass, s.Gravity), effectsText(g.FX), g.Tool, g.Mass), 10, 8)
	}
	if msg, ok := g.Message(); ok {
		w, _ := text.Measure(msg, bodyFace, 0)
		area := float64(g.screenW)
		if g.Explain {
			area -= panelWidth(g.screenW) + 32 // keep the hint left of the open panel
		}
		x := math.Max(12, (area-w)/2)
		drawText(screen, msg, bodyFace, x, float64(g.screenH)-48, accentColor)
	}
	if g.Explain {
		g.drawExplanation(screen)
	}
	if g.ShowHelp {
		var b strings.Builder
		for i, p := range scene.Presets {
			fmt.Fprintf(&b, "%d  %s\n", i+1, p.Name)
		}
		b.WriteString("\nb k tab     tool: black hole / star cluster\nclick       place\nalt+drag    launch with velocity\nm n         mass +/-\n" +
			"drag        rotate\nshift/right drag  pan\nwheel       zoom\n" +
			"e           EXPLAIN: learning mode (arrows: topic, audience)\nx v         dark matter: gravity on/off, show\no u         gas map, dust absorption\n< >         speed of light\np           hole gravity: instant / retarded / retarded+v\ng t l d     glow, trails, lens, doppler\nspace       pause\n.           single step\n+ -         speed\nr           restart\nc           center view\n" +
			"F5 F9       quick save / load latest\nf           fullscreen\nh           hide HUD\n? F1        hide this help")
		ebitenutil.DebugPrintAt(screen, b.String(), 10, g.screenH-16*strings.Count(b.String(), "\n")-28)
	}
}

// effectsText lists the render effects with their keys; active ones are marked with *.
func effectsText(fx view.Effects) string {
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
	return fmt.Sprintf("t = %.2f (%s)", t, app.FormatYears(t*sp.YearsPer))
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
