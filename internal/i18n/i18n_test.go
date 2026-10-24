package i18n

import "testing"

func TestFromLocale(t *testing.T) {
	for tag, want := range map[string]Lang{
		"de_DE.UTF-8": German, "de-AT": German, "DE": German,
		"en-US": English, "fr_FR": English, "": English, "C": English,
	} {
		if got := FromLocale(tag); got != want {
			t.Errorf("FromLocale(%q) = %v, want %v", tag, got, want)
		}
	}
}

func TestT(t *testing.T) {
	defer Set(Current())
	Set(German)
	if T("Ablage", "File") != "Ablage" {
		t.Error("German")
	}
	Set(English)
	if T("Ablage", "File") != "File" {
		t.Error("English")
	}
}
