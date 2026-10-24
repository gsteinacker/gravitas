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
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/mac"
	"galaxy/internal/perf"
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
	// ShotInfo, if set, is appended to the screenshot summary line (e.g. the performance profile).
	ShotInfo func() string
	started  time.Time
	frame    int
	shotErr  error
	shotDone bool

	pipeline *render.Pipeline

	// NativeMenu installs the macOS menu bar on the first update (not in screenshot mode).
	NativeMenu bool
	// PrefsChanged runs when the settings window changed profile, stars or GPU.
	PrefsChanged func()
	// Recalibrate measures the machine again (settings window).
	Recalibrate func()
	installed   bool
	exportPath  string // save the next frame as PNG here

	dragging         bool
	pressX, pressY   int
	lastX, lastY     int
	movedSincePress  bool
	launching        bool // Alt-drag: place a black hole with an initial velocity
	chars            []rune
	button           ebiten.MouseButton
	screenW, screenH int // window size in points; the screen image has renderScale times as many pixels

	// RenderScale, if > 0, fixes the render pixels per window point (-scale flag, screenshots);
	// otherwise the governor chooses.
	RenderScale float64
	renderScale float64
	// Adaptive lets a governor adapt quality at run time (T-047), with render scales up to
	// MaxRenderScale (and the display's).
	Adaptive       bool
	MaxRenderScale float64
	governor       *perf.Governor
	// TargetSPS is the step rate the governor aims at (from the performance profile).
	TargetSPS float64
	// Tick, if set, runs after every governor tick (about twice per second).
	Tick   func()
	ticks  int
	spsCap int
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
		g.govern() // with -adapt, measurements see the governor like the app
		return nil // screenshots must be reproducible: ignore stray input on the window
	}
	if g.NativeMenu && !g.installed {
		g.installed = true
		mac.Install(g.Controller, mac.Hooks{ExportImage: func(path string) { g.exportPath = path }, PrefsChanged: g.PrefsChanged, Recalibrate: g.Recalibrate})
	}
	g.Drain()
	g.handleKeys()
	g.handleMouse()
	g.govern()
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
	threshold := int(math.Ceil(clickThreshold * g.Scale()))
	if abs(x-g.pressX) > threshold || abs(y-g.pressY) > threshold {
		g.movedSincePress = true
	}
	dx, dy := float64(x-g.lastX), float64(y-g.lastY)
	if g.movedSincePress && !g.launching {
		if g.button == ebiten.MouseButtonLeft && !ebiten.IsKeyPressed(ebiten.KeyShift) {
			g.Cam.Rotate(dx/g.Scale(), dy/g.Scale()) // the same angle per point at every resolution
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
	k := float32(g.Scale())
	vector.StrokeLine(screen, x0, y0, x1, y1, 2*k, orange, true)
	vector.StrokeCircle(screen, x0, y0, 7*k, 2*k, orange, true)
	if dx, dy := x1-x0, y1-y0; dx*dx+dy*dy > 25*k*k {
		l := float32(math.Hypot(float64(dx), float64(dy)))
		ux, uy := dx/l, dy/l
		for _, side := range []float32{1, -1} {
			vector.StrokeLine(screen, x1, y1, x1-k*(10*ux-6*side*uy), y1-k*(10*uy+6*side*ux), 2*k, orange, true)
		}
	}
	v := app.LaunchVelocity(&g.Cam, float64(g.pressX), float64(g.pressY), float64(x), float64(y))
	drawText(screen, fmt.Sprintf("v = %.2f", v.Len()), smallFace, float64(x)/g.Scale()+12, float64(y)/g.Scale()+8, orange)
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{2, 2, 8, 255})
	b := screen.Bounds()
	g.Cam.W, g.Cam.H = float64(b.Dx()), float64(b.Dy())
	g.pipeline.PixelScale = g.Scale()
	uiScale = g.Scale()
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
		extra := ""
		if g.ShotInfo != nil {
			extra = " " + g.ShotInfo()
		}
		fmt.Printf("shot %s: bodies=%d fps=%.1f steps_per_s=%.1f substeps=%d gas_substeps=%d t=%.2f%s\n",
			g.ShotPath, s.Bodies, ebiten.ActualFPS(), g.Runner.StepsPerSecond(), s.Substeps, s.HydroSubsteps, s.Time, extra)
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

// Layout makes the screen renderScale times the window size, so on a Retina display (scale 2)
// every display pixel is rendered. Overlays are laid out in window points (screenW, screenH).
func (g *Game) Layout(w, h int) (int, int) {
	g.screenW, g.screenH = w, h
	s := g.Scale()
	return int(math.Round(float64(w) * s)), int(math.Round(float64(h) * s))
}

// Scale is the current number of render pixels per window point.
func (g *Game) Scale() float64 {
	switch {
	case g.RenderScale > 0:
		return g.RenderScale
	case g.renderScale > 0:
		return g.renderScale
	}
	return 1
}

// SetProfile changes the step rate the governor aims at and its highest render scale.
func (g *Game) SetProfile(targetSPS, maxRenderScale float64) {
	g.TargetSPS, g.MaxRenderScale = targetSPS, maxRenderScale
	g.governor = nil // restart at the best quality of the new profile
}

// governorTicks is the number of updates (at 60 per second) between governor ticks.
const governorTicks = 30

// govern runs the governor every governorTicks updates and applies its settings.
func (g *Game) govern() {
	if !g.Adaptive {
		g.Runner.SetTargetSPS(g.SpeedLimit())
		return
	}
	if g.governor == nil {
		device := 1.0
		if m := ebiten.Monitor(); m != nil {
			device = m.DeviceScaleFactor()
		}
		g.governor = perf.NewGovernor(g.MaxRenderScale, device, g.ThetaUser)
		g.renderScale = g.governor.Current().RenderScale
	}
	g.ticks++
	if g.ticks%governorTicks == 1 {
		thermal, lowPower := mac.Thermal()
		sps := g.Runner.StepsPerSecond()
		if g.Runner.Paused() {
			sps = 0
		}
		g.governor.ThetaBase = g.ThetaUser
		set, _ := g.governor.Tick(perf.Inputs{
			FPS: ebiten.ActualFPS(), StepsPerSec: sps, TargetSPS: g.TargetSPS, UserLimit: g.SpeedLimit(),
			Thermal: thermal, LowPower: lowPower,
		})
		g.renderScale = set.RenderScale
		if set.Theta != g.Theta {
			g.ApplyTheta(set.Theta)
		}
		g.spsCap = set.SPSCap
		g.RenderScaleNow = g.Scale()
		if g.Tick != nil {
			g.Tick()
		}
	}
	limit := g.SpeedLimit()
	if g.spsCap > 0 && (limit == 0 || g.spsCap < limit) {
		limit = g.spsCap
	}
	g.Runner.SetTargetSPS(limit)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
