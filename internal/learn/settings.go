package learn

import (
	_ "embed"
	"encoding/json"

	"galaxy/internal/i18n"
)

// Explanations of the settings (T-061): what a control does, what it means and what it costs, for
// the three audiences in German and English. The app shows them behind an (i) next to the control.

// Text is a German and an English text.
type Text struct {
	De string `json:"de"`
	En string `json:"en"`
}

// String is the text in the user's language.
func (t Text) String() string { return i18n.T(t.De, t.En) }

// Setting explains one control.
type Setting struct {
	ID     string          `json:"id"`
	Topic  string          `json:"topic"` // learning topic to continue with, "" none
	Chart  string          `json:"chart"` // visualization, "" none (engine.SettingInfo)
	Title  Text            `json:"title"`
	Levels map[string]Text `json:"levels"`
}

// Text returns the explanation for a level in the user's language.
func (s *Setting) Text(l Level) string { return s.Levels[l.Key()].String() }

//go:embed settings.json
var settingsJSON []byte

var settings = mustLoadSettings()

func mustLoadSettings() map[string]*Setting {
	var list []*Setting
	if err := json.Unmarshal(settingsJSON, &list); err != nil {
		panic("setting explanations: " + err.Error()) // embedded, covered by tests
	}
	m := make(map[string]*Setting, len(list))
	for _, s := range list {
		m[s.ID] = s
	}
	return m
}

// SettingByID returns the explanation of a control, or nil.
func SettingByID(id string) *Setting { return settings[id] }

// SettingIDs returns the ids of all explained controls.
func SettingIDs() []string {
	ids := make([]string, 0, len(settings))
	for id := range settings {
		ids = append(ids, id)
	}
	return ids
}
