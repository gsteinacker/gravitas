package ui

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/view"
)

// hudLine is the line height of the status text and the key help.
const hudLine = 17

// drawHUD draws the status text (if shown), messages, the learning panel and the key help. All
// text uses the Go font, which has the umlauts of the German texts.
func (g *Game) drawHUD(screen *ebiten.Image) {
	if g.ShowHUD {
		drawLines(screen, g.statusLines(), 10, 8, hudColor)
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
		keys, texts := helpLines()
		y := float64(g.screenH - hudLine*len(keys) - 24)
		drawLines(screen, keys, 10, y, accentColor)
		drawLines(screen, texts, 130, y, hudColor)
	}
}

func drawLines(dst *ebiten.Image, lines []string, x, y float64, c color.Color) {
	for i, l := range lines {
		drawText(dst, l, smallFace, x, y+float64(i*hudLine), c)
	}
}

// statusLines is the status text in the top left corner.
func (g *Game) statusLines() []string {
	T := i18n.T
	speed := "max"
	if l := g.SpeedLimit(); l > 0 {
		speed = fmt.Sprintf("%d/s", l)
	}
	title := g.Scene.Name
	if g.Runner.Paused() {
		title += T("   [PAUSE]", "   [PAUSED]")
	}
	s := g.Sim()
	lines := []string{title}
	if g.Scene.Info != "" {
		lines = append(lines, g.Scene.Info)
	}
	lines = append(lines,
		fmt.Sprintf(T("%d Körper (Schwarze Löcher: %d)   %s", "bodies %d (black holes %d)   %s"), s.Bodies, s.BlackHoles, timeText(s.Time, g.Scene)),
		darkText(s.Dark, s.DarkMatterOff, g.FX.Dark))
	if gas := gasText(s.Gas, s.StarsFormed, g.FX.Gas); gas != "" {
		lines = append(lines, gas)
	}
	lines = append(lines,
		fmt.Sprintf(T("Schritte/s %.0f (Grenze %s)   Teilschritte bei Löchern %d, Gas %d   FPS %.0f",
			"steps/s %.0f (limit %s)   substeps near holes %d, gas %d   FPS %.0f"),
			g.Runner.StepsPerSecond(), speed, s.Substeps, max(1, s.HydroSubsteps), ebiten.ActualFPS()),
		driftText(g.Runner.EnergyDrift()),
		lightText(s.C, g.Mass, s.Gravity),
		effectsText(g.FX),
		"",
		fmt.Sprintf(T("Werkzeug: %s, Masse %.2f   (b/k/Tab, m/n)", "tool: %s, mass %.2f   (b/k/tab, m/n)"), toolName(g.Tool), g.Mass))
	return lines
}

func toolName(t app.Tool) string {
	if t == app.ToolCluster {
		return i18n.T("Sternhaufen", "star cluster")
	}
	return i18n.T("Schwarzes Loch", "black hole")
}

// helpLines is the key help: keys and their meaning, line by line.
func helpLines() (keys, texts []string) {
	T := i18n.T
	for i, p := range scene.Presets {
		keys, texts = append(keys, fmt.Sprint(i+1)), append(texts, p.Name)
	}
	keys, texts = append(keys, ""), append(texts, "")
	for _, kt := range [][2]string{
		{"b k Tab", T("Werkzeug: Schwarzes Loch / Sternhaufen", "tool: black hole / star cluster")},
		{T("Klick", "click"), T("platzieren", "place")},
		{T("Alt+Ziehen", "alt+drag"), T("mit Geschwindigkeit starten", "launch with velocity")},
		{"m n", T("Masse +/-", "mass +/-")},
		{T("Ziehen", "drag"), T("drehen", "rotate")},
		{T("Shift/rechts ziehen", "shift/right drag"), T("verschieben", "pan")},
		{T("Mausrad", "wheel"), T("zoomen", "zoom")},
		{"e", T("Lernmodus (Pfeile: Thema, Zielgruppe)", "learning mode (arrows: topic, audience)")},
		{"x v", T("Dunkle Materie: Gravitation an/aus, zeigen", "dark matter: gravity on/off, show")},
		{"o u", T("Gaskarte, Staubabsorption", "gas map, dust absorption")},
		{"< >", T("Lichtgeschwindigkeit", "speed of light")},
		{"p", T("Loch-Gravitation: sofort / retardiert / retardiert+v", "hole gravity: instant / retarded / retarded+v")},
		{"g t l d", T("Leuchten, Spuren, Linse, Doppler", "glow, trails, lens, doppler")},
		{T("Leertaste", "space"), T("Pause", "pause")},
		{".", T("Einzelschritt", "single step")},
		{"+ -", T("Tempo", "speed")},
		{"r", T("Neustart", "restart")},
		{"c", T("Ansicht zentrieren", "center view")},
		{"F5 F9", T("schnell sichern / letzten Stand laden", "quick save / load latest")},
		{"f", T("Vollbild", "fullscreen")},
		{"h", T("Statusanzeige aus", "hide status")},
		{"? F1", T("diese Hilfe aus", "hide this help")},
	} {
		keys, texts = append(keys, kt[0]), append(texts, kt[1])
	}
	return keys, texts
}

// effectsText lists the render effects with their keys; active ones are marked with *.
func effectsText(fx view.Effects) string {
	T := i18n.T
	mark := func(on bool, name string) string {
		if on {
			return " *" + name
		}
		return "  " + name
	}
	return T("Effekte:", "effects:") + mark(fx.Glow, T("Leuchten(g)", "glow(g)")) + mark(fx.Trails, T("Spuren(t)", "trails(t)")) +
		mark(fx.Lensing, T("Linse(l)", "lens(l)")) + mark(fx.Doppler, "doppler(d)") +
		mark(fx.Dust, T("Staub(u)", "dust(u)")) + mark(fx.Gas, T("Gaskarte(o)", "gas map(o)"))
}

// lightText shows the speed of light, the horizon size of the next placed black hole and how the
// gravity of holes propagates.
func lightText(c, mass float64, gravity sim.GravityMode) string {
	T := i18n.T
	if c <= 0 {
		return T("c = unendlich (Newton)", "c = infinite (Newtonian)")
	}
	note := ""
	if gravity != sim.GravityInstant {
		note = T(", Energie nicht erhalten", ", energy not conserved")
	}
	return fmt.Sprintf(T("c = %.1f  (< >)   r_s bei Masse %.2f = %.3f   Loch-Gravitation: %s (p)%s",
		"c = %.1f  (< >)   r_s of mass %.2f = %.3f   hole gravity: %s (p)%s"),
		c, mass, sim.SchwarzschildRadius(mass, c), gravityName(gravity), note)
}

func gravityName(g sim.GravityMode) string {
	switch g {
	case sim.GravityRetarded:
		return i18n.T("retardiert", "retarded")
	case sim.GravityRetardedExtrapolated:
		return i18n.T("retardiert+v", "retarded+v")
	}
	return i18n.T("sofort", "instant")
}

// timeText shows the simulation time, in physical units if the scene defines them.
func timeText(t float64, sp *scene.Spec) string {
	if sp.YearsPer <= 0 {
		return fmt.Sprintf(i18n.T("t = %.2f (ohne physikalische Einheiten)", "t = %.2f (no physical units)"), t)
	}
	return fmt.Sprintf("t = %.2f (%s)", t, app.FormatYears(t*sp.YearsPer))
}

// gasText describes the gas of the scene, if any.
func gasText(n, formed int, shown bool) string {
	T := i18n.T
	if n == 0 && formed == 0 {
		return ""
	}
	vis := T("verborgen", "hidden")
	if shown {
		vis = T("sichtbar", "shown")
	}
	return fmt.Sprintf(T("Gas %d Teilchen, %d Sterne entstanden, %s (o)", "gas %d bodies, %d stars formed, %s (o)"), n, formed, vis)
}

// darkText describes the dark matter of the scene, if any.
func darkText(n int, off, shown bool) string {
	T := i18n.T
	if n == 0 {
		return T("keine Dunkle Materie", "no dark matter")
	}
	state, vis := T("an", "on"), T("verborgen", "hidden")
	if off {
		state = T("AUS (Testteilchen)", "OFF (test particles)")
	}
	if shown {
		vis = T("sichtbar", "shown")
	}
	return fmt.Sprintf(T("Dunkle Materie %d Teilchen, Gravitation %s (x), %s (v)", "dark matter %d bodies, gravity %s (x), %s (v)"), n, state, vis)
}

// driftText formats the energy drift; "~" marks a tree estimate (error ~1e-6 since T-028).
func driftText(d float64, q sim.EnergyQuality) string {
	T := i18n.T
	switch {
	case q == sim.EnergyUnknown || math.IsNaN(d):
		return T("Energiedrift: wird gemessen …", "energy drift: measuring...")
	case q == sim.EnergyEstimated:
		return fmt.Sprintf(T("Energiedrift ~%+.0e (Baumschätzung)", "energy drift ~%+.0e (tree estimate)"), d)
	default:
		return fmt.Sprintf(T("Energiedrift %+.1e", "energy drift %+.1e"), d)
	}
}

// statusTexts returns every text the HUD can show, for the glyph test.
func statusTexts() []string {
	var all []string
	keys, texts := helpLines()
	all = append(all, keys...)
	all = append(all, texts...)
	for _, on := range []bool{false, true} {
		all = append(all, effectsText(view.Effects{Glow: on, Trails: on, Lensing: on, Doppler: on, Dust: on, Gas: on}),
			gasText(1, 1, on), darkText(1, on, on), lightText(1, 1, sim.GravityRetarded), toolName(app.ToolCluster),
			toolName(app.ToolBlackHole))
	}
	all = append(all, darkText(0, false, false), lightText(0, 1, 0), driftText(math.NaN(), 0),
		driftText(1, sim.EnergyEstimated), driftText(1, sim.EnergyExact))
	return all
}
