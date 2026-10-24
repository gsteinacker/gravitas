// Package ui wires input, camera, rendering and the simulation runner into an Ebitengine game.
package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
