package app

import (
	"math"
	"testing"

	"galaxy/internal/scene"
	"galaxy/internal/sim"
)

func TestRateMeterAveragesOverWindow(t *testing.T) {
	var r rateMeter
	// Steady 2 mass per time unit formed, 1 exploded, sampled every 0.01 for 20 units.
	for i := range 2001 {
		tm := float64(i) * 0.01
		r.add(tm, 2*tm, tm, 0, 5)
	}
	f, e, _ := r.rates()
	if math.Abs(f-2) > 1e-9 || math.Abs(e-1) > 1e-9 {
		t.Errorf("rates %.4g, %.4g, want 2, 1", f, e)
	}
	if n := len(r.samples); n > rateSamples+2 {
		t.Errorf("%d samples kept, want at most about %d", n, rateSamples)
	}
	if span := r.samples[len(r.samples)-1].t - r.samples[0].t; span < 5 || span > 5.1 {
		t.Errorf("window spans %.3g, want 5", span)
	}
	// A burst only counts while it is inside the window.
	r.add(20.5, 2*20.5+10, 20.5, 0, 5)
	if f, _, _ := r.rates(); f < 3 {
		t.Errorf("burst not seen: %.3g", f)
	}
	for i := range 700 {
		tm := 20.5 + float64(i+1)*0.01
		r.add(tm, 2*tm+10, tm, 0, 5)
	}
	if f, _, _ := r.rates(); math.Abs(f-2) > 1e-9 {
		t.Errorf("burst still counted after the window: %.4g", f)
	}
	// A restart (time goes back) starts over.
	r.add(0, 0, 0, 0, 5)
	if f, e, _ := r.rates(); f != 0 || e != 0 || len(r.samples) != 1 {
		t.Errorf("after restart: %.3g, %.3g, %d samples", f, e, len(r.samples))
	}
}

func TestStarRatesInPhysicalUnits(t *testing.T) {
	sp := &scene.Spec{YearsPer: 1e7, KmPerS: 100, Params: sim.Params{Gas: sim.GasParams{SFDensity: 1}}}
	msun, _ := sp.MassUnit()
	var r rateMeter
	r.add(0, 0, 0, 0, 10)
	r.add(1, 1/msun*1e7, 1/msun*1e7, 0.5/msun*1e7, 10) // 1e7 M☉ formed and exploded in 1e7 years
	got := starRates(sp, &r, 0.2, true)
	if !got.Formation || !got.Units || got.GasShare != 0.2 {
		t.Fatalf("%+v", got)
	}
	if math.Abs(got.SFR-1) > 1e-9 {
		t.Errorf("SFR %.4g M☉/yr, want 1", got.SFR)
	}
	if math.Abs(got.SNRate-1) > 1e-9 { // 1 M☉/yr · 0.01 per M☉ · 100 yr
		t.Errorf("SN rate %.4g per century, want 1", got.SNRate)
	}
	if !got.Infall || math.Abs(got.InfallRate-0.5) > 1e-9 {
		t.Errorf("infall %v %.4g M☉/yr, want 0.5", got.Infall, got.InfallRate)
	}
}
