package mac

import (
	"math"
	"strings"
	"testing"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/learn"
	"galaxy/internal/prefs"
	"galaxy/internal/sim"
)

func TestFormIDsAreUniqueAndEncodable(t *testing.T) {
	for _, l := range []i18n.Lang{i18n.German, i18n.English} {
		i18n.Set(l)
		seen := map[string]bool{}
		for _, form := range [][]control{inspectorForm(), settingsForm()} {
			for _, c := range form {
				if c.id == "" {
					if c.kind != ctlSection {
						t.Errorf("control %q without id", c.title)
					}
					continue
				}
				if seen[c.id] {
					t.Errorf("duplicate id %q", c.id)
				}
				seen[c.id] = true
				for _, o := range c.opts {
					if strings.ContainsAny(o, "|"+fieldSep+"\n") {
						t.Errorf("%s: option %q contains a separator", c.id, o)
					}
				}
				if c.kind == ctlSlider && !(c.min < c.max) {
					t.Errorf("%s: empty slider range", c.id)
				}
			}
			if n := strings.Count(encodeForm(form), "\n") + 1; n != len(form) {
				t.Errorf("%d lines for %d controls", n, len(form))
			}
		}
	}
	i18n.Set(i18n.English)
}

// TestFormValuesStayInRange: every control reports a value its widget can show.
func TestFormValuesStayInRange(t *testing.T) {
	s := app.State{C: 10, Mass: 0.5, Theta: 0.9, Speed: 5, Gravity: sim.GravityRetarded, Level: learn.Physics, Stars: 50000}
	p := prefs.Default()
	p.Stars, p.Profile = 100000, "high"
	for _, form := range [][]control{inspectorForm(), settingsForm()} {
		for _, c := range form {
			if c.id == "" {
				continue
			}
			v, _, _ := formValue(c.id, s, p, "M4 Pro")
			switch c.kind {
			case ctlSlider:
				if v < c.min-1e-9 || v > c.max+1e-9 {
					t.Errorf("%s: %v outside [%v, %v]", c.id, v, c.min, c.max)
				}
			case ctlSegments, ctlPopup:
				if int(v) < 0 || int(v) >= len(c.opts) {
					t.Errorf("%s: index %v of %d options", c.id, v, len(c.opts))
				}
			case ctlCheck:
				if v != 0 && v != 1 {
					t.Errorf("%s: checkbox value %v", c.id, v)
				}
			}
		}
	}
	if v, _, _ := formValue("stars", s, p, ""); StarChoices[int(v)] != 100000 {
		t.Errorf("stars popup shows %v", v)
	}
	if _, _, on := formValue("c", app.State{C: 0}, p, ""); on {
		t.Error("c slider must be disabled in a Newtonian run")
	}
}

func TestFormChangesReachController(t *testing.T) {
	c := testController(t)
	c.Prefs = prefs.Default()
	c.PrefsPath = "" // saving runs in the background; nothing to write here
	changed := 0
	h := Hooks{PrefsChanged: func() { changed++ }}

	onFormChange(c, h, "mass", math.Log10(2))
	onFormChange(c, h, "speed", 1)
	onFormChange(c, h, "tool", 1)
	onFormChange(c, h, "glow", 0)
	onFormChange(c, h, "trails", 1)
	onFormChange(c, h, "theta", 0.7)
	onFormChange(c, h, "level", 2)
	if math.Abs(c.Mass-2) > 1e-9 || c.Speed != 1 || c.Tool != app.ToolCluster || c.FX.Glow || !c.FX.Trails ||
		c.Theta != 0.7 || c.Level != learn.Physics {
		t.Errorf("state after changes: mass %v speed %d tool %v fx %+v theta %v level %v", c.Mass, c.Speed, c.Tool, c.FX, c.Theta, c.Level)
	}
	onFormChange(c, h, "trails", 1) // setting the same value must not toggle
	if !c.FX.Trails {
		t.Error("checkbox set to its current value toggled the effect")
	}

	onFormChange(c, h, "stars", 4)
	onFormChange(c, h, "gpu", 2)
	onFormChange(c, h, "profile", 1)
	if c.Prefs.Stars != StarChoices[4] || c.Prefs.GPU != prefs.GPUOff || c.Prefs.Profile != "low" || changed != 3 {
		t.Errorf("prefs %+v, hook calls %d", c.Prefs, changed)
	}
	onFormChange(c, h, "sim", 0)
	if c.Runner.Paused() {
		t.Error("first button resumes the paused test run")
	}
}
