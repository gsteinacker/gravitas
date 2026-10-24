package mac

import (
	"strings"
	"testing"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/view"
)

func TestMenuIsBalancedAndKeysAreUnique(t *testing.T) {
	for _, l := range []i18n.Lang{i18n.German, i18n.English} {
		i18n.Set(l)
		depth := 0
		keys := map[string]string{}
		for _, e := range menuBar() {
			switch e.kind {
			case kindMenu:
				depth++
			case kindEnd:
				depth--
			case kindItem, kindSystem:
				if strings.TrimSpace(e.title) == "" {
					t.Errorf("item without title: %+v", e)
				}
				if e.key == "" {
					continue
				}
				id := e.mods + "+" + e.key
				if other, dup := keys[id]; dup {
					t.Errorf("key %q used by %q and %q", id, other, e.title)
				}
				keys[id] = e.title
				if strings.ToLower(e.key) != e.key {
					t.Errorf("%q: key equivalent %q must be lower case (upper case means Shift)", e.title, e.key)
				}
			}
			if depth < 0 {
				t.Fatal("menu closes more than it opens")
			}
			if strings.ContainsAny(e.title, fieldSep+"\n") {
				t.Errorf("title %q contains a separator", e.title)
			}
		}
		if depth != 0 {
			t.Errorf("%d menus left open", depth)
		}
	}
	i18n.Set(i18n.English)
}

// TestEveryActionIsReachable keeps menu and actions in sync: each action is in the menu or is
// panel navigation that only makes sense with keys.
func TestEveryActionIsReachable(t *testing.T) {
	inMenu := map[app.Action]bool{}
	for _, e := range menuBar() {
		if e.kind == kindItem && e.tag < commandBase {
			inMenu[app.Action(e.tag)] = true
		}
	}
	keysOnly := map[app.Action]bool{
		app.ActionNextTool: true, app.ActionNextTopic: true, app.ActionPrevTopic: true,
		app.ActionNextLevel: true, app.ActionPrevLevel: true, app.ActionCloseExplain: true,
	}
	for a := app.ActionNone + 1; a < app.ActionScene; a++ {
		if !inMenu[a] && !keysOnly[a] {
			t.Errorf("action %d is not in the menu", a)
		}
	}
	for i := range scene.Presets {
		if !inMenu[app.SceneAction(i)] {
			t.Errorf("scene %d is not in the menu", i)
		}
	}
}

// TestMenuKeysMatchGameKeys: a menu key equivalent without modifiers must run the same action as
// the typed character, since AppKit hands the key to whichever claims it first.
func TestMenuKeysMatchGameKeys(t *testing.T) {
	for _, e := range menuBar() {
		if e.kind != kindItem || e.mods != "" || e.key == "" {
			continue
		}
		a, ok := app.ActionForChar([]rune(e.key)[0])
		if !ok || int(a) != e.tag {
			t.Errorf("%q: key %q runs action %d in the game, menu tag is %d", e.title, e.key, a, e.tag)
		}
	}
}

func TestEncodeHasOneLinePerEntry(t *testing.T) {
	es := menuBar()
	lines := strings.Split(encode(es), "\n")
	if len(lines) != len(es) {
		t.Fatalf("%d lines for %d entries", len(lines), len(es))
	}
	for _, l := range lines {
		if n := strings.Count(l, fieldSep); n != 5 {
			t.Errorf("line %q has %d separators, want 5", l, n)
		}
	}
}

func TestItemState(t *testing.T) {
	s := app.State{
		Paused: true, Speed: 0, C: app.CMax, Gravity: sim.GravityRetarded, Tool: app.ToolCluster,
		Mass: app.MassMin, FX: view.Effects{Glow: true}, Scene: 2, Sim: app.SimState{Dark: 0},
	}
	cases := map[int]int{
		int(app.ActionTogglePause):      stateChecked,
		int(app.ActionSlower):           stateDisabled,
		int(app.ActionFaster):           0,
		int(app.ActionLightFaster):      stateDisabled,
		int(app.ActionLightSlower):      0,
		int(app.ActionGravityRetarded):  stateChecked,
		int(app.ActionGravityInstant):   0,
		int(app.ActionToolCluster):      stateChecked,
		int(app.ActionToolBlackHole):    0,
		int(app.ActionMassDown):         stateDisabled,
		int(app.ActionToggleGlow):       stateChecked,
		int(app.ActionToggleTrails):     0,
		int(app.ActionToggleDarkMatter): stateDisabled, // scene without dark matter
		int(app.SceneAction(2)):         stateChecked,
		int(app.SceneAction(1)):         0,
		commandTag(CmdSaveState):        0,
	}
	for tag, want := range cases {
		if got := itemState(tag, s); got != want {
			t.Errorf("tag %d: state %d, want %d", tag, got, want)
		}
	}
	s.C = 0 // Newtonian: c cannot change
	if itemState(int(app.ActionLightSlower), s) != stateDisabled {
		t.Error("c actions must be disabled without finite c")
	}
}

func TestFileNameIsSafe(t *testing.T) {
	n := fileName("A/B: C", ".galaxy")
	if strings.ContainsAny(n, "/:") || !strings.HasPrefix(n, "A-B- C ") || !strings.HasSuffix(n, ".galaxy") {
		t.Errorf("file name %q", n)
	}
	if !strings.HasPrefix(fileName("", ".png"), "Galaxy ") {
		t.Error("empty scene name needs a fallback")
	}
}
