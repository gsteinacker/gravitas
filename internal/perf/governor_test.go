package perf

import (
	"math"
	"testing"
)

// run feeds the same inputs for n ticks and returns the settings after the last.
func run(g *Governor, in Inputs, n int) Settings {
	var s Settings
	for range n {
		s, _ = g.Tick(in)
	}
	return s
}

var smooth = Inputs{FPS: 60, StepsPerSec: 31, TargetSPS: 30}

func TestGovernorStartsAtTheBestScale(t *testing.T) {
	for _, tc := range []struct{ max, device, want float64 }{
		{2, 2, 2}, {2, 1, 1}, {1.5, 2, 1.5}, {1, 2, 1}, {2, 3, 2},
	} {
		if s := NewGovernor(tc.max, tc.device, 0.9).Current(); s.RenderScale != tc.want || s.Theta != 0.9 || s.SPSCap != 0 {
			t.Errorf("max %v device %v: %+v, want scale %v", tc.max, tc.device, s, tc.want)
		}
	}
}

func TestSmoothRunChangesNothing(t *testing.T) {
	g := NewGovernor(2, 2, 0.9)
	for i := range 100 {
		if _, changed := g.Tick(smooth); changed {
			t.Fatalf("tick %d changed the settings: %+v", i, g.Current())
		}
	}
}

func TestSlowFramesLowerTheScaleThenCapSteps(t *testing.T) {
	g := NewGovernor(2, 2, 0.9)
	slow := smooth
	slow.FPS = 30
	if s := run(g, slow, slowTicks-1); s.RenderScale != 2 {
		t.Errorf("degraded before %d slow ticks: %+v", slowTicks, s)
	}
	if s := run(g, slow, 1); s.RenderScale != 1.5 {
		t.Errorf("after %d slow ticks: %+v", slowTicks, s)
	}
	// Keep it slow: down to the lowest scale, then the step cap.
	s := run(g, slow, 200)
	if s.RenderScale != 0.5 || s.SPSCap != 30 {
		t.Errorf("persistently slow frames: %+v", s)
	}
	if s.Theta != 0.9 {
		t.Errorf("slow frames must not touch θ: %v", s.Theta)
	}
}

func TestRecoveryIsSlowerThanDegradation(t *testing.T) {
	g := NewGovernor(2, 2, 0.9)
	slow := smooth
	slow.FPS = 30
	run(g, slow, slowTicks)
	if g.Current().RenderScale != 1.5 {
		t.Fatal("did not degrade")
	}
	if s := run(g, smooth, cooldownTicks+fastTicks-1); s.RenderScale != 1.5 {
		t.Errorf("recovered too fast: %+v", s)
	}
	if s := run(g, smooth, 1); s.RenderScale != 2 {
		t.Errorf("did not recover: %+v", s)
	}
}

// TestNoFlickerAtTheBorder: frame rates between the thresholds keep the level.
func TestNoFlickerAtTheBorder(t *testing.T) {
	g := NewGovernor(2, 2, 0.9)
	changes := 0
	for i := range 200 {
		in := smooth
		in.FPS = []float64{52, 55, 47, 56, 53}[i%5] // one slow tick now and then
		if _, c := g.Tick(in); c {
			changes++
		}
	}
	if changes != 0 {
		t.Errorf("%d changes at frame rates around the threshold", changes)
	}
}

func TestSlowStepsRaiseThetaWithinBounds(t *testing.T) {
	g := NewGovernor(2, 2, 0.9)
	slowSim := smooth
	slowSim.StepsPerSec = 12
	s := run(g, slowSim, 200)
	if math.Abs(s.Theta-ThetaMax) > 1e-9 {
		t.Errorf("θ = %v, want %v", s.Theta, ThetaMax)
	}
	if s.RenderScale != 2 {
		t.Errorf("slow steps must not cost render quality: %+v", s)
	}
	fastSim := smooth
	fastSim.StepsPerSec = 100
	if s := run(g, fastSim, 200); math.Abs(s.Theta-0.9) > 1e-9 {
		t.Errorf("θ did not return to the user's value: %v", s.Theta)
	}
	// A user limit below the target is not a slow simulation.
	limited := smooth
	limited.UserLimit, limited.StepsPerSec = 10, 10
	if s := run(g, limited, 200); math.Abs(s.Theta-0.9) > 1e-9 {
		t.Errorf("user limit raised θ to %v", s.Theta)
	}
	paused := smooth
	paused.StepsPerSec = 0
	if s := run(g, paused, 200); math.Abs(s.Theta-0.9) > 1e-9 {
		t.Errorf("pause raised θ to %v", s.Theta)
	}
}

func TestHeatAndLowPowerDegradeAtOnce(t *testing.T) {
	g := NewGovernor(2, 2, 0.9)
	hot := smooth
	hot.Thermal = ThermalSerious
	s, changed := g.Tick(hot)
	if !changed || s.RenderScale != 1 || s.SPSCap != 30 {
		t.Errorf("serious thermal state: %+v", s)
	}
	if s := run(g, hot, 100); s.RenderScale > 1 {
		t.Errorf("recovered while hot: %+v", s)
	}
	critical := smooth
	critical.Thermal = ThermalCritical
	if s, _ := g.Tick(critical); s.RenderScale != 0.5 {
		t.Errorf("critical: %+v", s)
	}
	g2 := NewGovernor(2, 2, 0.9)
	low := smooth
	low.LowPower = true
	if s, _ := g2.Tick(low); s.RenderScale != 1 || s.SPSCap != 30 {
		t.Errorf("low power: %+v", s)
	}
	if s := run(g2, smooth, 100); s.RenderScale != 2 || s.SPSCap != 0 {
		t.Errorf("no recovery after low power: %+v", s)
	}
}
