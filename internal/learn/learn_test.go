package learn

import (
	"testing"

	"galaxy/internal/scene"
)

func TestTopicsAreComplete(t *testing.T) {
	for _, id := range IDs() {
		tp := Get(id)
		if tp == nil {
			t.Fatalf("topic %q listed in Order but missing", id)
		}
		if tp.Title == "" || tp.Short == "" {
			t.Errorf("%s: missing title or short text", id)
		}
		for l := School; l < levels; l++ {
			if len(tp.Text(l)) < 100 {
				t.Errorf("%s: text for %s missing or too short", id, l.Name())
			}
		}
	}
}

func TestScenesReferenceExistingTopicsAndHaveIntros(t *testing.T) {
	for _, p := range scene.Presets {
		for _, id := range p.Spec.Topics {
			if Get(id) == nil {
				t.Errorf("scene %q references unknown topic %q", p.Name, id)
			}
		}
		for l := School; l < levels; l++ {
			if p.Spec.Intro[l.Key()] == "" {
				t.Errorf("scene %q has no intro for %s", p.Name, l.Name())
			}
		}
	}
}

func TestLevelCycle(t *testing.T) {
	if School.Prev() != Physics || Physics.Next() != School || Lay.Next() != Physics {
		t.Error("levels must cycle")
	}
}
