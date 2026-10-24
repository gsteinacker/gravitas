package view

import "testing"

func TestParseEffects(t *testing.T) {
	fx, err := ParseEffects("glow, trails")
	if err != nil || fx != (Effects{Glow: true, Trails: true}) {
		t.Errorf("ParseEffects = %+v, %v", fx, err)
	}
	if _, err := ParseEffects("sparkles"); err == nil {
		t.Error("unknown effect must be an error")
	}
	if fx, err := ParseEffects(""); err != nil || fx != (Effects{}) {
		t.Errorf("empty list = %+v, %v", fx, err)
	}
}
