package render

import (
	"math"
	"testing"
)

func TestFrequencyShift(t *testing.T) {
	const c = 10
	if f := frequencyShift(0, 0, 0, 0, 0, 1, c, 1); f != 1 {
		t.Errorf("star at rest: f = %v, want 1", f)
	}
	// Approaching at 0.6c: D = √((1+β)/(1−β)) = 2.
	if f := frequencyShift(0, 0, 6, 0, 0, 1, c, 1); math.Abs(f-2) > 1e-12 {
		t.Errorf("approaching at 0.6c: f = %v, want 2", f)
	}
	if f := frequencyShift(0, 0, -6, 0, 0, 1, c, 1); math.Abs(f-0.5) > 1e-12 {
		t.Errorf("receding at 0.6c: f = %v, want 0.5", f)
	}
	// Transverse Doppler: pure time dilation, f = 1/γ = 0.8.
	if f := frequencyShift(6, 0, 0, 0, 0, 1, c, 1); math.Abs(f-0.8) > 1e-12 {
		t.Errorf("transverse at 0.6c: f = %v, want 0.8", f)
	}
	if f := frequencyShift(0, 0, 6, 0, 0, 1, 0, 0.7); f != 0.7 {
		t.Errorf("c = 0 must ignore Doppler: f = %v", f)
	}
}

func TestGravShift(t *testing.T) {
	if g := gravShift(4, 1); math.Abs(g-math.Sqrt(0.75)) > 1e-15 {
		t.Errorf("gravShift(4, 1) = %v", g)
	}
	if g := gravShift(1, 1); g <= 0 {
		t.Errorf("at the horizon the star must stay faintly visible, got %v", g)
	}
	if gravShift(1, 0) != 1 {
		t.Error("no hole, no shift")
	}
}

func TestShiftColorDirection(t *testing.T) {
	r, _, b, beamBlue := shiftColor(0.8, 0.8, 0.8, 1.5)
	if b <= r || beamBlue <= 1 {
		t.Errorf("blueshift should tint blue and brighten: r=%v b=%v beam=%v", r, b, beamBlue)
	}
	r, _, b, beamRed := shiftColor(0.8, 0.8, 0.8, 0.6)
	if r <= b || beamRed >= 1 {
		t.Errorf("redshift should tint red and dim: r=%v b=%v beam=%v", r, b, beamRed)
	}
	if r, g, b, beam := shiftColor(0.3, 0.4, 0.5, 1); r != 0.3 || g != 0.4 || b != 0.5 || beam != 1 {
		t.Errorf("f = 1 must not change the color: %v %v %v %v", r, g, b, beam)
	}
}
