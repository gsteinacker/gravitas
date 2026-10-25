// Package app is the front-end independent core of the interactive app: the actions a user can
// trigger (by key, menu or widget) and the controller that executes them on the simulation. It
// has no user interface, so any front end (the SwiftUI app through internal/engine) can drive it.
package app

import "galaxy/internal/scene"

// Action is one user command. Scene actions are ActionScene+i for built-in scene i.
type Action int

const (
	ActionNone Action = iota
	ActionTogglePause
	ActionStep
	ActionFaster
	ActionSlower
	ActionLightSlower // c ÷ 1.25
	ActionLightFaster // c × 1.25
	ActionCycleGravity
	ActionGravityInstant
	ActionGravityRetarded
	ActionGravityRetardedV
	ActionToggleDarkMatter
	ActionToolBlackHole
	ActionToolCluster
	ActionNextTool
	ActionMassUp
	ActionMassDown
	ActionToggleGlow
	ActionToggleTrails
	ActionToggleLens
	ActionToggleDoppler
	ActionToggleDark
	ActionToggleGas
	ActionToggleDust
	ActionToggleHUD
	ActionToggleHelp
	ActionCenterView
	ActionToggleFullscreen
	ActionToggleExplain
	ActionNextTopic
	ActionPrevTopic
	ActionNextLevel
	ActionPrevLevel
	ActionCloseExplain
	ActionRestart
	ActionQuickSave
	ActionQuickLoad

	// ActionScene must stay last: ActionScene+i loads built-in scene i.
	ActionScene
)

// SceneAction loads built-in scene i (0-based).
func SceneAction(i int) Action { return ActionScene + Action(i) }

// SceneIndex reports which built-in scene a scene action loads.
func (a Action) SceneIndex() (int, bool) {
	i := int(a - ActionScene)
	return i, a >= ActionScene && i < len(scene.Presets)
}

// charActions are the bindings of typed characters. They are characters, not key positions, so
// they work on any keyboard layout. Layout-independent keys (space, arrows, F-keys) are mapped by
// the front end.
var charActions = map[rune]Action{
	'.': ActionStep,
	'+': ActionFaster, '=': ActionFaster, '-': ActionSlower,
	'm': ActionMassUp, 'M': ActionMassUp, ']': ActionMassUp,
	'n': ActionMassDown, 'N': ActionMassDown, '[': ActionMassDown,
	'b': ActionToolBlackHole, 'B': ActionToolBlackHole,
	'k': ActionToolCluster, 'K': ActionToolCluster, '\t': ActionNextTool,
	'<': ActionLightSlower, '>': ActionLightFaster,
	'g': ActionToggleGlow, 'G': ActionToggleGlow,
	't': ActionToggleTrails, 'T': ActionToggleTrails,
	'l': ActionToggleLens, 'L': ActionToggleLens,
	'd': ActionToggleDoppler, 'D': ActionToggleDoppler,
	'p': ActionCycleGravity, 'P': ActionCycleGravity,
	'x': ActionToggleDarkMatter, 'X': ActionToggleDarkMatter,
	'v': ActionToggleDark, 'V': ActionToggleDark,
	'o': ActionToggleGas, 'O': ActionToggleGas,
	'u': ActionToggleDust, 'U': ActionToggleDust,
	'e': ActionToggleExplain, 'E': ActionToggleExplain,
	'r': ActionRestart, 'R': ActionRestart,
	'h': ActionToggleHUD, 'H': ActionToggleHUD,
	'?': ActionToggleHelp,
	'c': ActionCenterView, 'C': ActionCenterView,
	'f': ActionToggleFullscreen, 'F': ActionToggleFullscreen,
}

// ActionForChar returns the action bound to a typed character.
func ActionForChar(r rune) (Action, bool) {
	if r >= '1' && r <= '9' {
		a := SceneAction(int(r - '1'))
		_, ok := a.SceneIndex()
		return a, ok
	}
	a, ok := charActions[r]
	return a, ok
}

// topicFor is the learning topic an action touches; triggering it shows that topic's hint.
var topicFor = map[Action]string{
	ActionToggleDarkMatter: "dark-matter", ActionToggleDark: "dark-matter",
	ActionToggleGas: "gas", ActionToggleDust: "dust",
	ActionLightSlower: "speed-of-light", ActionLightFaster: "speed-of-light",
	ActionCycleGravity: "retarded-gravity", ActionGravityRetarded: "retarded-gravity",
	ActionGravityRetardedV: "retarded-gravity",
	ActionToggleDoppler:    "doppler", ActionToggleLens: "lensing",
}
