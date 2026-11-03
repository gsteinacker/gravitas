package scene

import "testing"

// Every built-in scene is in the overview exactly once, in a known group, with a summary; the
// digit keys 1–9 go to the first nine scenes, none twice (T-078).
func TestCatalogListsEverySceneOnce(t *testing.T) {
	seen := map[int]bool{}
	keys := map[string]int{}
	for _, e := range Catalog() {
		if seen[e.Index] {
			t.Errorf("scene %q listed twice", e.Name)
		}
		seen[e.Index] = true
		if !knownGroup(e.Group) {
			t.Errorf("scene %q has unknown group %q", e.Name, e.Group)
		}
		if e.Summary == "" {
			t.Errorf("scene %q has no summary", e.Name)
		}
		if e.Key != "" {
			if j, ok := keys[e.Key]; ok {
				t.Errorf("key %s for scenes %d and %d", e.Key, j, e.Index)
			}
			keys[e.Key] = e.Index
			if want := string(rune('1' + e.Index)); e.Key != want {
				t.Errorf("scene %d has key %s, want %s", e.Index, e.Key, want)
			}
		}
	}
	if len(seen) != len(Presets) {
		t.Errorf("overview lists %d of %d scenes", len(seen), len(Presets))
	}
}
