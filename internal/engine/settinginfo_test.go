package engine

import (
	"math"
	"testing"

	"galaxy/internal/learn"
)

// TestSettingInfoCharts builds every explanation in a galaxy scene: charts are present where the
// data names one, and their numbers are finite.
func TestSettingInfoCharts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := New(0, 2000)
	defer e.Close()
	for _, id := range learn.SettingIDs() {
		info, ok := e.SettingInfo(id)
		if !ok || info.Title == "" || info.Text == "" {
			t.Errorf("%s: %+v", id, info)
			continue
		}
		if learn.SettingByID(id).Chart != "" && info.Chart == nil {
			t.Errorf("%s: chart missing", id)
		}
		if c := info.Chart; c != nil {
			for _, s := range c.Series {
				for _, p := range s.Points {
					if math.IsNaN(p[1]) || math.IsInf(p[1], 0) || (c.LogY && p[1] <= 0) {
						t.Errorf("%s/%s: point %v", id, s.Name, p)
					}
				}
			}
		}
	}
	if _, ok := e.SettingInfo("nonsense"); ok {
		t.Error("unknown id must fail")
	}
}

// TestHorizonChartFollowsC checks the physics of the c chart: r_s ∝ 1/c².
func TestHorizonChartFollowsC(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	e := New(0, 2000)
	defer e.Close()
	ch := e.speedOfLightChart()
	p := ch.Series[0].Points
	first, last := p[0], p[len(p)-1]
	if got := (first[1] / last[1]) / math.Pow(last[0]/first[0], 2); math.Abs(got-1) > 1e-9 {
		t.Errorf("r_s ratio / c² ratio = %v, want 1", got)
	}
}
