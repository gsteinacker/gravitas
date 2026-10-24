package ui

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"galaxy/internal/sim"
	"galaxy/internal/view"
)

func TestEffectsText(t *testing.T) {
	fx := view.Effects{Glow: true, Trails: true, Lensing: true, Doppler: true}
	if got := effectsText(fx); got != "effects:*glow(g)*trails(t)*lens(l)*doppler(d) dust(u) gas map(o)" {
		t.Errorf("effectsText = %q", got)
	}
}

func TestDriftText(t *testing.T) {
	if got := driftText(1.234e-3, sim.EnergyExact); got != "energy drift +1.2e-03" {
		t.Errorf("exact: %q", got)
	}
	if got := driftText(-4e-4, sim.EnergyEstimated); got != "energy drift ~-4e-04 (tree estimate)" {
		t.Errorf("estimated: %q", got)
	}
	if got := driftText(math.NaN(), sim.EnergyExact); got != "energy drift: measuring..." {
		t.Errorf("unknown: %q", got)
	}
}

func TestFormatYears(t *testing.T) {
	for y, want := range map[float64]string{2.5: "2.5 yr", 47200: "47.2 kyr", 13.2e6: "13.2 Myr", 1.09e9: "1.09 Gyr"} {
		if got := formatYears(y); got != want {
			t.Errorf("formatYears(%v) = %q, want %q", y, got, want)
		}
	}
}

func TestWrapKeepsLinesWithinWidth(t *testing.T) {
	s := "Alles, was Masse hat, zieht sich gegenseitig an: die Erde dich, die Sonne die Erde.\n\nNeuer Absatz mit Umlauten äöü und θ."
	lines := wrap(s, bodyFace, 200)
	if len(lines) < 4 {
		t.Fatalf("expected several lines, got %q", lines)
	}
	for _, l := range lines {
		if w, _ := text.Measure(l, bodyFace, 0); w > 200 && strings.Contains(l, " ") {
			t.Errorf("line %q is %.0f px wide", l, w)
		}
	}
	if !slices.Contains(lines, "") {
		t.Error("paragraph break must produce an empty line")
	}
}
