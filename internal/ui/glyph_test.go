package ui

import (
	"sort"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"

	"galaxy/internal/i18n"
	"galaxy/internal/learn"
	"galaxy/internal/scene"
)

// TestLearningTextsUseOnlyAvailableGlyphs: characters missing from the font render as boxes.
func TestLearningTextsUseOnlyAvailableGlyphs(t *testing.T) {
	f, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	missing := map[rune][]string{}
	check := func(where, s string) {
		for _, r := range s {
			if r == '\n' {
				continue
			}
			if gi, err := f.GlyphIndex(&buf, r); err != nil || gi == 0 {
				missing[r] = append(missing[r], where)
			}
		}
	}
	for _, id := range learn.IDs() {
		tp := learn.Get(id)
		check(id, tp.Title+tp.Short+tp.Keys)
		for _, txt := range tp.Levels {
			check(id, txt)
		}
	}
	for _, p := range scene.Presets {
		for _, txt := range p.Spec.Intro {
			check(p.Name, txt)
		}
	}
	check("ui", "←/→ Thema   ↑/↓ Zielgruppe   e oder Esc schließen …")
	// HUD, help and messages in both languages (T-045).
	defer i18n.Set(i18n.Current())
	for _, l := range []i18n.Lang{i18n.German, i18n.English} {
		i18n.Set(l)
		for _, txt := range statusTexts() {
			check("hud", txt)
		}
		for lv := learn.School; lv <= learn.Physics; lv++ {
			check("level", lv.Name())
		}
	}
	var rs []rune
	for r := range missing {
		rs = append(rs, r)
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i] < rs[j] })
	for _, r := range rs {
		t.Errorf("glyph %q (U+%04X) missing, used in %v", r, r, uniq(missing[r]))
	}
}

func uniq(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range s {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
