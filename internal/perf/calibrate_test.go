package perf

import (
	"testing"
	"time"

	"galaxy/internal/prefs"
	"galaxy/internal/scene"
)

func TestStarsForFitsTheBudget(t *testing.T) {
	// Roughly an M4 Pro with GPU: 100k stars + halo take ~5 ms per force evaluation.
	c := prefs.Calibration{SecPerNL: 5e-3 / nlogn(150_000), BodiesPerStar: 1.5}
	n := StarsFor(c, StepBudget)
	cost := stepOverhead * c.SecPerNL * nlogn(c.BodiesPerStar*float64(n))
	if cost > StepBudget.Seconds() {
		t.Errorf("%d stars cost %.1f ms, budget %v", n, cost*1e3, StepBudget)
	}
	if n < 100_000 {
		t.Errorf("fast machine gets only %d stars", n)
	}
	// Half the budget gives fewer stars, a slower machine too.
	if m := StarsFor(c, StepBudget/2); m >= n {
		t.Errorf("half budget: %d stars, full: %d", m, n)
	}
	slow := c
	slow.SecPerNL *= 10
	if m := StarsFor(slow, StepBudget); m >= n/5 {
		t.Errorf("10× slower machine: %d stars vs %d", m, n)
	}
}

func TestStarsForClamps(t *testing.T) {
	if n := StarsFor(prefs.Calibration{SecPerNL: 1e-15, BodiesPerStar: 1}, time.Second); n != MaxStars {
		t.Errorf("very fast: %d", n)
	}
	if n := StarsFor(prefs.Calibration{SecPerNL: 1e-3, BodiesPerStar: 1}, time.Millisecond); n != MinStars {
		t.Errorf("very slow: %d", n)
	}
	if n := StarsFor(prefs.Calibration{}, StepBudget); n != 0 {
		t.Errorf("uncalibrated: %d", n)
	}
}

func TestRoundDown(t *testing.T) {
	for in, want := range map[int]int{87_341: 87_000, 312_999: 310_000, 5_432: 5_400, 999: MinStars} {
		if got := roundDown(in); got != want {
			t.Errorf("roundDown(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestProfilesOrderStars(t *testing.T) {
	c := prefs.Calibration{SecPerNL: 5e-3 / nlogn(150_000), BodiesPerStar: 1.5}
	low, high, maxi := ProfileByID("low").Stars(c, 0), ProfileByID("high").Stars(c, 0), ProfileByID("max").Stars(c, 0)
	if !(low < high && high < maxi) {
		t.Errorf("stars low %d, high %d, max %d", low, high, maxi)
	}
	if ProfileByID("bogus").ID != "" {
		t.Error("unknown profile must fall back to automatic")
	}
	if ProfileByID("").Stars(prefs.Calibration{}, 42) != 42 {
		t.Error("fallback for uncalibrated machines")
	}
}

func TestCalibrateMeasures(t *testing.T) {
	if testing.Short() {
		t.Skip("measures for up to a second")
	}
	t0 := time.Now()
	c := Calibrate(true)
	if d := time.Since(t0); d > 3*time.Second {
		t.Errorf("calibration took %v", d)
	}
	if c.SecPerNL <= 0 || c.BodiesPerStar < 1 {
		t.Errorf("calibration %+v", c)
	}
	n := StarsFor(c, StepBudget)
	t.Logf("calibration %+v → %d stars for 30 steps/s (took %v)", c, n, time.Since(t0))
}

func TestGasScenesGetFewerStars(t *testing.T) {
	c := prefs.Calibration{SecPerNL: 5e-3 / nlogn(150_000), BodiesPerStar: 1, SecPerGasNL: 7 * 5e-3 / nlogn(150_000)}
	var gasScene, plain *scene.Spec
	for _, p := range scene.Presets {
		if f := GasFraction(p.Spec); f > 0.5 {
			gasScene = p.Spec
		} else if f == 0 && plain == nil {
			plain = p.Spec
		}
	}
	if gasScene == nil || plain == nil {
		t.Fatal("need a gas scene and a gas-free scene")
	}
	auto := ProfileByID("")
	g, s := auto.SceneStars(c, gasScene, 0), auto.SceneStars(c, plain, 0)
	if g*2 > s {
		t.Errorf("gas scene %d stars, gas-free %d", g, s)
	}
	m := c.BodiesPerStar * float64(g)
	cost := stepOverhead*c.SecPerNL*nlogn(m) + gasOverhead*c.SecPerGasNL*nlogn(GasFraction(gasScene)*m)
	if cost > StepBudget.Seconds() {
		t.Errorf("gas scene with %d stars costs %.1f ms", g, cost*1e3)
	}
}
