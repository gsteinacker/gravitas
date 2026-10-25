package learn

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

// TestSettingsAreComplete checks every explanation: both languages, all levels, a known topic.
func TestSettingsAreComplete(t *testing.T) {
	if len(SettingIDs()) == 0 {
		t.Fatal("no setting explanations")
	}
	for _, id := range SettingIDs() {
		s := SettingByID(id)
		if s.Title.De == "" || s.Title.En == "" {
			t.Errorf("%s: title missing", id)
		}
		for l := School; l <= Physics; l++ {
			if txt := s.Levels[l.Key()]; txt.De == "" || txt.En == "" {
				t.Errorf("%s: %s text missing in a language", id, l.Key())
			}
		}
		if s.Topic != "" && Get(s.Topic) == nil {
			t.Errorf("%s: unknown topic %q", id, s.Topic)
		}
	}
}

// TestEverySettingInTheAppIsExplained matches the (i) buttons of the SwiftUI app with the data:
// every button has an explanation and every explanation a button.
func TestEverySettingInTheAppIsExplained(t *testing.T) {
	files, _ := filepath.Glob("../../macos/GalaxySwift/Sources/GalaxySwift/*.swift")
	if len(files) == 0 {
		t.Skip("app sources not found")
	}
	re := regexp.MustCompile(`(?:InfoButton\(engine, |\.info\(engine, |effect\(.*, Engine\.[a-z]+, )"([A-Za-z]+)"\)`)
	var used []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(data), -1) {
			used = append(used, m[1])
		}
	}
	for _, id := range used {
		if SettingByID(id) == nil {
			t.Errorf("(i) button %q has no explanation", id)
		}
	}
	for _, id := range SettingIDs() {
		if !slices.Contains(used, id) {
			t.Errorf("explanation %q has no (i) button", id)
		}
	}
}
