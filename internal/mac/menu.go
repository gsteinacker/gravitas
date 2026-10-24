// Package mac is the native macOS shell around the app: menu bar, open and save panels, alerts
// and (later) the inspector. The Cocoa part needs cgo on darwin (ADR-0009); elsewhere every
// function is a no-op, so callers need no build tags.
package mac

import (
	"fmt"
	"strings"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
)

// Command is a menu entry that is not a plain app action, because it needs a dialog or the
// front end. Its tag is commandBase+Command, beyond all actions.
type Command int

const (
	CmdOpenScene Command = iota
	CmdOpenState
	CmdSaveState
	CmdExportImage
	CmdAbout
	CmdInspector
	CmdSettings
)

// commandBase separates command tags from action tags.
const commandBase = 1 << 16

func commandTag(c Command) int { return commandBase + int(c) }

// itemKind says how a menu entry is built.
type itemKind byte

const (
	kindMenu      itemKind = 'M' // begins a (sub)menu: title
	kindEnd       itemKind = 'E' // ends the current (sub)menu
	kindItem      itemKind = 'I' // title, key, modifiers, tag
	kindSystem    itemKind = 'A' // title, key, modifiers, selector sent to the first responder
	kindSeparator itemKind = 'S'
	kindWindows   itemKind = 'W' // marks the current menu as the window menu
	kindServices  itemKind = 'V' // a Services submenu
	kindHelp      itemKind = 'H' // marks the current menu as the help menu
)

// entry is one line of the menu description passed to Cocoa.
type entry struct {
	kind  itemKind
	title string
	key   string // key equivalent, "" for none
	mods  string // c = command, s = shift, o = option, t = control; "" = none
	tag   int    // item tag (action or command)
	sel   string // selector for kindSystem
}

// The fields of a line are separated by the ASCII unit separator, lines by newline.
const fieldSep = "\x1f"

func (e entry) encode() string {
	return strings.Join([]string{string(e.kind), e.title, e.key, e.mods, fmt.Sprint(e.tag), e.sel}, fieldSep)
}

func encode(es []entry) string {
	lines := make([]string, len(es))
	for i, e := range es {
		lines[i] = e.encode()
	}
	return strings.Join(lines, "\n")
}

type menuBuilder struct{ es []entry }

func (b *menuBuilder) menu(title string) { b.es = append(b.es, entry{kind: kindMenu, title: title}) }
func (b *menuBuilder) end()              { b.es = append(b.es, entry{kind: kindEnd}) }
func (b *menuBuilder) sep()              { b.es = append(b.es, entry{kind: kindSeparator}) }
func (b *menuBuilder) mark(k itemKind)   { b.es = append(b.es, entry{kind: k}) }
func (b *menuBuilder) action(t string, a app.Action, key, mods string) {
	b.es = append(b.es, entry{kind: kindItem, title: t, key: key, mods: mods, tag: int(a)})
}
func (b *menuBuilder) command(t string, c Command, key, mods string) {
	b.es = append(b.es, entry{kind: kindItem, title: t, key: key, mods: mods, tag: commandTag(c)})
}
func (b *menuBuilder) system(t, sel, key, mods string) {
	b.es = append(b.es, entry{kind: kindSystem, title: t, key: key, mods: mods, sel: sel})
}

// menuBar describes the whole menu bar. Single-key equivalents without modifiers are the game
// keys: AppKit offers every key-down to the menu first, so a key either runs its menu item or
// reaches the game, never both. Keys that need Shift on some layouts (?, <, >, +) and the
// space bar stay with the game, which reads typed characters.
func menuBar() []entry {
	T := i18n.T
	b := &menuBuilder{}

	b.menu("Galaxy")
	b.command(T("Über Galaxy", "About Galaxy"), CmdAbout, "", "")
	b.sep()
	b.command(T("Einstellungen …", "Settings…"), CmdSettings, ",", "c")
	b.sep()
	b.mark(kindServices)
	b.sep()
	b.system(T("Galaxy ausblenden", "Hide Galaxy"), "hide:", "h", "c")
	b.system(T("Andere ausblenden", "Hide Others"), "hideOtherApplications:", "h", "co")
	b.system(T("Alle einblenden", "Show All"), "unhideAllApplications:", "", "")
	b.sep()
	b.system(T("Galaxy beenden", "Quit Galaxy"), "terminate:", "q", "c")
	b.end()

	b.menu(T("Ablage", "File"))
	b.menu(T("Neu aus Szene", "New from Scene"))
	for i, p := range scene.Presets {
		key := ""
		if i < 9 {
			key = fmt.Sprint(i + 1)
		}
		b.action(p.Name, app.SceneAction(i), key, "")
	}
	b.end()
	b.command(T("Szene öffnen …", "Open Scene…"), CmdOpenScene, "o", "c")
	b.sep()
	b.command(T("Stand öffnen …", "Open State…"), CmdOpenState, "o", "cs")
	b.command(T("Stand sichern …", "Save State…"), CmdSaveState, "s", "c")
	b.action(T("Schnell sichern", "Quick Save"), app.ActionQuickSave, "", "")
	b.action(T("Letzten Stand laden", "Load Latest"), app.ActionQuickLoad, "", "")
	b.sep()
	b.command(T("Bild exportieren …", "Export Image…"), CmdExportImage, "e", "cs")
	b.sep()
	b.system(T("Fenster schließen", "Close Window"), "performClose:", "w", "c")
	b.end()

	b.menu(T("Simulation", "Simulation"))
	b.action(T("Pause (Leertaste)", "Pause (Space)"), app.ActionTogglePause, "", "")
	b.action(T("Einzelschritt", "Single Step"), app.ActionStep, ".", "")
	b.sep()
	b.action(T("Schneller (+)", "Faster (+)"), app.ActionFaster, "", "")
	b.action(T("Langsamer", "Slower"), app.ActionSlower, "-", "")
	b.sep()
	b.action(T("Szene neu starten", "Restart Scene"), app.ActionRestart, "r", "")
	b.end()

	b.menu(T("Physik", "Physics"))
	b.action(T("Licht schneller (>)", "Faster Light (>)"), app.ActionLightFaster, "", "")
	b.action(T("Licht langsamer (<)", "Slower Light (<)"), app.ActionLightSlower, "", "")
	b.sep()
	b.menu(T("Gravitation der Löcher", "Black Hole Gravity"))
	b.action(T("Sofort (Newton)", "Instant (Newton)"), app.ActionGravityInstant, "", "")
	b.action(T("Retardiert", "Retarded"), app.ActionGravityRetarded, "", "")
	b.action(T("Retardiert + v·Δt", "Retarded + v·Δt"), app.ActionGravityRetardedV, "", "")
	b.sep()
	b.action(T("Nächste", "Next"), app.ActionCycleGravity, "p", "")
	b.end()
	b.action(T("Dunkle Materie wirkt", "Dark Matter Gravity"), app.ActionToggleDarkMatter, "x", "")
	b.end()

	b.menu(T("Werkzeug", "Tool"))
	b.action(T("Schwarzes Loch", "Black Hole"), app.ActionToolBlackHole, "b", "")
	b.action(T("Sternhaufen", "Star Cluster"), app.ActionToolCluster, "k", "")
	b.sep()
	b.action(T("Mehr Masse", "More Mass"), app.ActionMassUp, "m", "")
	b.action(T("Weniger Masse", "Less Mass"), app.ActionMassDown, "n", "")
	b.end()

	b.menu(T("Darstellung", "View"))
	b.action(T("Leuchten", "Glow"), app.ActionToggleGlow, "g", "")
	b.action(T("Spuren", "Trails"), app.ActionToggleTrails, "t", "")
	b.action(T("Gravitationslinse", "Lensing"), app.ActionToggleLens, "l", "")
	b.action(T("Doppler-Farben", "Doppler Colors"), app.ActionToggleDoppler, "d", "")
	b.action(T("Staub", "Dust"), app.ActionToggleDust, "u", "")
	b.action(T("Gas zeigen", "Show Gas"), app.ActionToggleGas, "o", "")
	b.action(T("Dunkle Materie zeigen", "Show Dark Matter"), app.ActionToggleDark, "v", "")
	b.sep()
	b.command(T("Inspektor", "Inspector"), CmdInspector, "i", "co")
	b.action(T("Statusanzeige", "Status Overlay"), app.ActionToggleHUD, "h", "")
	b.action(T("Ansicht zentrieren", "Center View"), app.ActionCenterView, "c", "")
	b.action(T("Vollbild", "Full Screen"), app.ActionToggleFullscreen, "f", "ct")
	b.end()

	b.menu(T("Fenster", "Window"))
	b.system(T("Im Dock ablegen", "Minimize"), "performMiniaturize:", "m", "c")
	b.system(T("Zoomen", "Zoom"), "performZoom:", "", "")
	b.mark(kindWindows)
	b.end()

	b.menu(T("Hilfe", "Help"))
	b.action(T("Lernmodus", "Learning Mode"), app.ActionToggleExplain, "e", "")
	b.action(T("Tastenübersicht (?)", "Key Overview (?)"), app.ActionToggleHelp, "", "")
	b.mark(kindHelp)
	b.end()
	return b.es
}

// Item states reported to AppKit's menu validation.
const (
	stateChecked  = 1
	stateDisabled = 2
)

// itemState is the check mark and availability of a menu item in state s.
func itemState(tag int, s app.State) int {
	on := func(b bool) int {
		if b {
			return stateChecked
		}
		return 0
	}
	if tag >= commandBase {
		return 0
	}
	a := app.Action(tag)
	if i, ok := a.SceneIndex(); ok {
		return on(s.Scene == i)
	}
	switch a {
	case app.ActionTogglePause:
		return on(s.Paused)
	case app.ActionLightFaster:
		if s.C <= 0 || s.C >= app.CMax {
			return stateDisabled
		}
	case app.ActionLightSlower:
		if s.C <= 0 || s.C <= app.CMin {
			return stateDisabled
		}
	case app.ActionFaster:
		if s.Speed >= len(app.SpeedLevels)-1 {
			return stateDisabled
		}
	case app.ActionSlower:
		if s.Speed <= 0 {
			return stateDisabled
		}
	case app.ActionMassUp:
		if s.Mass >= app.MassMax {
			return stateDisabled
		}
	case app.ActionMassDown:
		if s.Mass <= app.MassMin {
			return stateDisabled
		}
	case app.ActionGravityInstant:
		return on(s.Gravity == sim.GravityInstant)
	case app.ActionGravityRetarded:
		return on(s.Gravity == sim.GravityRetarded)
	case app.ActionGravityRetardedV:
		return on(s.Gravity == sim.GravityRetardedExtrapolated)
	case app.ActionToggleDarkMatter:
		if s.Sim.Dark == 0 {
			return stateDisabled
		}
		return on(!s.DarkMatterOff)
	case app.ActionToolBlackHole:
		return on(s.Tool == app.ToolBlackHole)
	case app.ActionToolCluster:
		return on(s.Tool == app.ToolCluster)
	case app.ActionToggleGlow:
		return on(s.FX.Glow)
	case app.ActionToggleTrails:
		return on(s.FX.Trails)
	case app.ActionToggleLens:
		return on(s.FX.Lensing)
	case app.ActionToggleDoppler:
		return on(s.FX.Doppler)
	case app.ActionToggleDust:
		return on(s.FX.Dust)
	case app.ActionToggleGas:
		return on(s.FX.Gas)
	case app.ActionToggleDark:
		return on(s.FX.Dark)
	case app.ActionToggleHUD:
		return on(s.ShowHUD)
	case app.ActionToggleHelp:
		return on(s.ShowHelp)
	case app.ActionToggleExplain:
		return on(s.Explain)
	}
	return 0
}
