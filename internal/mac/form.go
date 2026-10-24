package mac

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/learn"
	"galaxy/internal/prefs"
	"galaxy/internal/sim"
)

// Native windows built from a form description: the floating inspector and the settings window.
// Like the menu, the controls are described in Go and built by Cocoa; Cocoa polls their values a
// few times per second and reports changes back.

// Window identifies a form window.
type Window int

const (
	WindowInspector Window = iota
	WindowSettings
)

// controlKind says which AppKit control a form row becomes.
type controlKind byte

const (
	ctlSection  controlKind = 'L' // section heading
	ctlSlider   controlKind = 'S' // NSSlider in [min, max] with a value text
	ctlCheck    controlKind = 'C' // checkbox, value 0/1
	ctlSegments controlKind = 'G' // segmented control, value = index
	ctlPopup    controlKind = 'P' // pop-up button, value = index
	ctlButtons  controlKind = 'R' // a row of push buttons, value = index; text renames the first
	ctlValue    controlKind = 'V' // read-only text
)

type control struct {
	kind     controlKind
	id       string
	title    string
	opts     []string
	min, max float64
}

func (c control) encode() string {
	return strings.Join([]string{string(c.kind), c.id, c.title, strings.Join(c.opts, "|"),
		strconv.FormatFloat(c.min, 'g', -1, 64), strconv.FormatFloat(c.max, 'g', -1, 64)}, fieldSep)
}

func encodeForm(cs []control) string {
	lines := make([]string, len(cs))
	for i, c := range cs {
		lines[i] = c.encode()
	}
	return strings.Join(lines, "\n")
}

// Logarithmic sliders show log10 of the value.
var (
	logCMin, logCMax       = math.Log10(app.CMin), math.Log10(app.CMax)
	logMassMin, logMassMax = math.Log10(app.MassMin), math.Log10(app.MassMax)
)

func speedNames() []string {
	names := make([]string, len(app.SpeedLevels))
	for i, s := range app.SpeedLevels {
		names[i] = strconv.Itoa(s)
		if s == 0 {
			names[i] = "max"
		}
	}
	return names
}

func inspectorForm() []control {
	T := i18n.T
	return []control{
		{kind: ctlValue, id: "scene", title: T("Szene", "Scene")},
		{kind: ctlButtons, id: "sim", opts: []string{T("Pause", "Pause"), T("Schritt", "Step"), T("Neustart", "Restart")}},
		{kind: ctlSegments, id: "speed", title: T("Schritte/s", "Steps/s"), opts: speedNames()},
		{kind: ctlValue, id: "time", title: T("Zeit", "Time")},
		{kind: ctlValue, id: "perf", title: T("Leistung", "Performance")},
		{kind: ctlValue, id: "drift", title: T("Energiedrift", "Energy drift")},

		{kind: ctlSection, title: T("Physik", "Physics")},
		{kind: ctlSlider, id: "c", title: T("Lichtgeschw.", "Light speed"), min: logCMin, max: logCMax},
		{kind: ctlSegments, id: "gravity", title: T("Loch-Gravitation", "Hole gravity"),
			opts: []string{T("sofort", "instant"), T("retardiert", "retarded"), T("ret. + v", "ret. + v")}},
		{kind: ctlCheck, id: "dm", title: T("Dunkle Materie wirkt", "Dark matter gravity")},
		{kind: ctlSlider, id: "theta", title: T("Baum-Genauigkeit", "Tree accuracy"), min: app.ThetaMin, max: app.ThetaMax},

		{kind: ctlSection, title: T("Werkzeug (Klick)", "Tool (click)")},
		{kind: ctlSegments, id: "tool", opts: []string{T("Schwarzes Loch", "Black hole"), T("Sternhaufen", "Star cluster")}},
		{kind: ctlSlider, id: "mass", title: T("Masse", "Mass"), min: logMassMin, max: logMassMax},

		{kind: ctlSection, title: T("Darstellung", "View")},
		{kind: ctlCheck, id: "glow", title: T("Leuchten", "Glow")},
		{kind: ctlCheck, id: "trails", title: T("Spuren", "Trails")},
		{kind: ctlCheck, id: "lens", title: T("Gravitationslinse", "Lensing")},
		{kind: ctlCheck, id: "doppler", title: T("Doppler-Farben", "Doppler colors")},
		{kind: ctlCheck, id: "dust", title: T("Staub", "Dust")},
		{kind: ctlCheck, id: "gas", title: T("Gas zeigen", "Show gas")},
		{kind: ctlCheck, id: "dark", title: T("Dunkle Materie zeigen", "Show dark matter")},
		{kind: ctlCheck, id: "hud", title: T("Statusanzeige im Bild", "Status overlay")},
		{kind: ctlCheck, id: "explain", title: T("Lernmodus", "Learning mode")},
	}
}

// StarChoices are the star counts offered in the settings; 0 = automatic.
var StarChoices = []int{0, 10_000, 25_000, 50_000, 100_000, 200_000, 500_000, 1_000_000}

// ProfileChoices are the performance profiles (internal/perf); "" = automatic.
var ProfileChoices = []string{"", "low", "medium", "high", "max"}

var gpuChoices = []string{prefs.GPUAuto, prefs.GPUOn, prefs.GPUOff}

func settingsForm() []control {
	T := i18n.T
	stars := make([]string, len(StarChoices))
	for i, n := range StarChoices {
		stars[i] = groupDigits(n)
	}
	stars[0] = T("Automatisch", "Automatic")
	return []control{
		{kind: ctlSection, title: T("Leistung", "Performance")},
		{kind: ctlPopup, id: "profile", title: T("Profil", "Profile"), opts: []string{
			T("Automatisch", "Automatic"), T("Niedrig", "Low"), T("Mittel", "Medium"), T("Hoch", "High"), T("Maximal", "Maximum")}},
		{kind: ctlPopup, id: "stars", title: T("Sterne pro Szene", "Stars per scene"), opts: stars},
		{kind: ctlPopup, id: "gpu", title: T("Grafikkarte (Metal)", "GPU (Metal)"), opts: []string{
			T("Automatisch", "Automatic"), T("Ein", "On"), T("Aus", "Off")}},
		{kind: ctlValue, id: "starsNow", title: T("Aktuell", "Current")},
		{kind: ctlValue, id: "hardware", title: T("Rechner", "Computer")},
		{kind: ctlSection, title: T("Lernmodus", "Learning mode")},
		{kind: ctlPopup, id: "level", title: T("Zielgruppe", "Audience"), opts: []string{
			T("Schüler", "Pupils"), T("Interessierte", "Interested laypeople"), T("Physik", "Physicists")}},
	}
}

// groupDigits writes 50000 as "50 000" (thin grouping works in both languages).
func groupDigits(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + " " + s[i:]
	}
	return s
}

// formValue is the value, text and availability of a control in state s.
func formValue(id string, s app.State, p *prefs.Prefs, hardware string) (v float64, text string, enabled bool) {
	T := i18n.T
	b := func(on bool) float64 {
		if on {
			return 1
		}
		return 0
	}
	enabled = true
	switch id {
	case "scene":
		text = s.SceneName
	case "sim":
		text = T("Pause", "Pause")
		if s.Paused {
			text = T("Weiter", "Resume")
		}
	case "speed":
		v = float64(s.Speed)
	case "time":
		text = fmt.Sprintf("t = %.2f", s.Sim.Time)
		if s.YearsPer > 0 {
			text += "  (" + app.FormatYears(s.Sim.Time*s.YearsPer) + ")"
		}
	case "perf":
		text = fmt.Sprintf(T("%.0f Schritte/s · %.0f FPS · %s Körper", "%.0f steps/s · %.0f FPS · %s bodies"),
			s.StepsPerSec, s.FPS, groupDigits(s.Sim.Bodies))
	case "drift":
		switch {
		case s.DriftQuality == sim.EnergyUnknown || math.IsNaN(s.Drift):
			text = T("wird gemessen …", "measuring…")
		case s.DriftQuality == sim.EnergyEstimated:
			text = fmt.Sprintf("~%+.0e", s.Drift)
		default:
			text = fmt.Sprintf("%+.1e", s.Drift)
		}
	case "c":
		enabled = s.C > 0
		if enabled {
			v, text = math.Log10(s.C), fmt.Sprintf("c = %.1f", s.C)
		} else {
			v, text = logCMax, T("unendlich", "infinite")
		}
	case "gravity":
		v = float64(s.Gravity)
	case "dm":
		v, enabled = b(!s.DarkMatterOff), s.Sim.Dark > 0
	case "theta":
		v, text = s.Theta, fmt.Sprintf("θ = %.2f", s.Theta)
	case "tool":
		v = float64(s.Tool)
	case "mass":
		v, text = math.Log10(s.Mass), fmt.Sprintf("%.2f", s.Mass)
	case "glow":
		v = b(s.FX.Glow)
	case "trails":
		v = b(s.FX.Trails)
	case "lens":
		v = b(s.FX.Lensing)
	case "doppler":
		v = b(s.FX.Doppler)
	case "dust":
		v = b(s.FX.Dust)
	case "gas":
		v = b(s.FX.Gas)
	case "dark":
		v = b(s.FX.Dark)
	case "hud":
		v = b(s.ShowHUD)
	case "explain":
		v = b(s.Explain)
	case "profile":
		v = float64(max(0, index(ProfileChoices, p.Profile)))
	case "stars":
		v = float64(max(0, index(StarChoices, p.Stars)))
	case "gpu":
		v = float64(max(0, index(gpuChoices, p.GPU)))
	case "starsNow":
		text = fmt.Sprintf(T("%s Sterne, ab nächster Szene", "%s stars, from the next scene"), groupDigits(s.Stars))
	case "hardware":
		text = hardware
	case "level":
		v = float64(s.Level)
	}
	return v, text, enabled
}

func index[T comparable](list []T, v T) int {
	for i, x := range list {
		if x == v {
			return i
		}
	}
	return -1
}

// onFormChange applies a control change; it runs on the update goroutine.
func onFormChange(c *app.Controller, h Hooks, id string, v float64) {
	i := int(math.Round(v))
	on := v >= 0.5
	toggle := func(cur bool, a app.Action) {
		if cur != on {
			c.Do(a)
		}
	}
	switch id {
	case "sim":
		c.Do([]app.Action{app.ActionTogglePause, app.ActionStep, app.ActionRestart}[max(0, min(2, i))])
	case "speed":
		c.SetSpeed(i)
	case "c":
		c.SetC(math.Pow(10, v))
	case "gravity":
		c.SetGravity(sim.GravityMode(max(0, min(2, i))))
	case "dm":
		c.SetDarkMatterOff(!on)
	case "theta":
		c.SetTheta(v)
	case "tool":
		c.Tool = app.Tool(max(0, min(1, i)))
	case "mass":
		c.SetMass(math.Pow(10, v))
	case "glow":
		toggle(c.FX.Glow, app.ActionToggleGlow)
	case "trails":
		toggle(c.FX.Trails, app.ActionToggleTrails)
	case "lens":
		toggle(c.FX.Lensing, app.ActionToggleLens)
	case "doppler":
		toggle(c.FX.Doppler, app.ActionToggleDoppler)
	case "dust":
		toggle(c.FX.Dust, app.ActionToggleDust)
	case "gas":
		toggle(c.FX.Gas, app.ActionToggleGas)
	case "dark":
		toggle(c.FX.Dark, app.ActionToggleDark)
	case "hud":
		toggle(c.ShowHUD, app.ActionToggleHUD)
	case "explain":
		toggle(c.Explain, app.ActionToggleExplain)
	case "level":
		c.SetLevel(learn.Level(max(0, min(int(learn.Physics), i))))
	case "profile", "stars", "gpu":
		if c.Prefs == nil {
			return
		}
		switch id {
		case "profile":
			c.Prefs.Profile = ProfileChoices[max(0, min(len(ProfileChoices)-1, i))]
		case "stars":
			c.Prefs.Stars = StarChoices[max(0, min(len(StarChoices)-1, i))]
		case "gpu":
			c.Prefs.GPU = gpuChoices[max(0, min(len(gpuChoices)-1, i))]
		}
		c.SavePrefs()
		if h.PrefsChanged != nil {
			h.PrefsChanged()
		}
	}
}
