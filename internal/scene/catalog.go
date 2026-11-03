package scene

import (
	"strings"

	"galaxy/internal/i18n"
)

// Groups are the scene groups in the order of the scene overview and the menu (T-078).
var Groups = []string{"galaxies", "stars", "gas", "cosmos"}

// GroupTitle is the display title of a scene group.
func GroupTitle(group string) string {
	switch group {
	case "galaxies":
		return i18n.T("Galaxien", "Galaxies")
	case "stars":
		return i18n.T("Sterne und Schwarze Löcher", "Stars and Black Holes")
	case "gas":
		return i18n.T("Gas und Sternentstehung", "Gas and Star Formation")
	case "cosmos":
		return i18n.T("Kosmos", "Cosmos")
	}
	return i18n.T("Weitere", "More")
}

// Entry is a built-in scene in the overview.
type Entry struct {
	Index      int    `json:"index"` // position in Presets, the argument of app.SceneAction
	Name       string `json:"name"`
	Group      string `json:"group"`
	GroupTitle string `json:"groupTitle"`
	Summary    string `json:"summary"` // first sentence of the school introduction
	Key        string `json:"key"`     // "1"–"9" for the first nine scenes, else empty
}

// Catalog lists the built-in scenes by group (in Groups order), then in preset order.
func Catalog() []Entry {
	var out []Entry
	for _, g := range append(Groups, "") {
		for i, p := range Presets {
			if p.Spec.Group != g && !(g == "" && !knownGroup(p.Spec.Group)) {
				continue
			}
			e := Entry{Index: i, Name: p.Name, Group: g, GroupTitle: GroupTitle(g), Summary: firstSentence(p.Spec.Intro["school"])}
			if i < 9 {
				e.Key = string(rune('1' + i))
			}
			out = append(out, e)
		}
	}
	return out
}

func knownGroup(g string) bool {
	for _, k := range Groups {
		if k == g {
			return true
		}
	}
	return false
}

// firstSentence returns s up to its first sentence end (". ", "! " or "? ").
func firstSentence(s string) string {
	end := len(s)
	for _, sep := range []string{". ", "! ", "? "} {
		if i := strings.Index(s, sep); i >= 0 && i+1 < end {
			end = i + 1
		}
	}
	return s[:end]
}
