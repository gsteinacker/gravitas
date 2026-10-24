package app

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"galaxy/internal/learn"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/direct"
	"galaxy/internal/vec"
	"galaxy/internal/view"
)

// newTestController runs a small paused scene; n bodies.
func newTestController(t *testing.T, sp *scene.Spec, n int) *Controller {
	t.Helper()
	s, p := sp.Build(n)
	r := sim.NewRunner(sim.New(s, &direct.Solver{}, p))
	r.SetPaused(true)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go r.Run(ctx)
	c := New(r, n, sp)
	c.SaveDir = t.TempDir()
	return c
}

// waitSim polls the runner until cond holds on the latest snapshot.
func waitSim(t *testing.T, c *Controller, what string, cond func(*sim.Snapshot) bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ok := false
		c.Runner.View(func(sn *sim.Snapshot) {
			ok = cond(sn)
			c.Observe(sn)
		})
		if ok {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

func TestCharBindingsWorkWithoutUSLayout(t *testing.T) {
	// Characters as typed on a German keyboard: dedicated "+" and "-" keys, "m"/"n" for mass.
	c := &Controller{Speed: 2, Mass: 1}
	for _, r := range "++-mmn" {
		a, ok := ActionForChar(r)
		if !ok {
			t.Fatalf("%q is not bound", r)
		}
		c.Do(a)
	}
	if c.Speed != 3 {
		t.Errorf("speed level = %d, want 3", c.Speed)
	}
	if math.Abs(c.Mass-1.5) > 1e-12 {
		t.Errorf("mass = %v, want 1.5", c.Mass)
	}
	if _, ok := ActionForChar('ü'); ok {
		t.Error("ü must not be bound")
	}
	if a, ok := ActionForChar('3'); !ok || a != SceneAction(2) {
		t.Errorf("'3' = %v, %v; want scene 2", a, ok)
	}
}

// TestLocalActions checks every action that only changes controller state.
func TestLocalActions(t *testing.T) {
	cases := []struct {
		a     Action
		check func(*Controller) bool
	}{
		{ActionFaster, func(c *Controller) bool { return c.Speed == 3 }},
		{ActionSlower, func(c *Controller) bool { return c.Speed == 1 }},
		{ActionMassUp, func(c *Controller) bool { return math.Abs(c.Mass-1.5) < 1e-12 }},
		{ActionMassDown, func(c *Controller) bool { return math.Abs(c.Mass-1/1.5) < 1e-12 }},
		{ActionToolCluster, func(c *Controller) bool { return c.Tool == ToolCluster }},
		{ActionNextTool, func(c *Controller) bool { return c.Tool == ToolCluster }},
		{ActionToggleGlow, func(c *Controller) bool { return c.FX == view.Effects{Glow: true} }},
		{ActionToggleTrails, func(c *Controller) bool { return c.FX == view.Effects{Trails: true} }},
		{ActionToggleLens, func(c *Controller) bool { return c.FX == view.Effects{Lensing: true} }},
		{ActionToggleDoppler, func(c *Controller) bool { return c.FX == view.Effects{Doppler: true} }},
		{ActionToggleDark, func(c *Controller) bool { return c.FX == view.Effects{Dark: true} }},
		{ActionToggleGas, func(c *Controller) bool { return c.FX == view.Effects{Gas: true} }},
		{ActionToggleDust, func(c *Controller) bool { return c.FX == view.Effects{Dust: true} }},
		{ActionToggleHUD, func(c *Controller) bool { return c.ShowHUD }},
		{ActionToggleHelp, func(c *Controller) bool { return c.ShowHelp }},
		{ActionToggleExplain, func(c *Controller) bool { return c.Explain }},
		{ActionNextTopic, func(c *Controller) bool { return c.TopicIdx == 1 }},
		{ActionPrevTopic, func(c *Controller) bool { return c.TopicIdx == 2 }},
		{ActionNextLevel, func(c *Controller) bool { return c.Level == learn.Lay }},
		{ActionPrevLevel, func(c *Controller) bool { return c.Level == learn.Physics }},
		{ActionCenterView, func(c *Controller) bool { return c.Cam.Target == vec.Vec3{} }},
	}
	for _, tc := range cases {
		c := &Controller{Speed: 2, Mass: 1, Topics: []string{"", "a", "b"}, Cam: view.Camera{Target: vec.Vec3{X: 1}}}
		c.Do(tc.a)
		if !tc.check(c) {
			t.Errorf("action %d: unexpected state %+v", tc.a, c)
		}
	}
	c := &Controller{Explain: true}
	c.Do(ActionCloseExplain)
	if c.Explain {
		t.Error("close must close the panel")
	}
	full := false
	c.ToggleFullscreen = func() { full = !full }
	c.Do(ActionToggleFullscreen)
	if !full {
		t.Error("fullscreen hook not called")
	}
}

func TestSimulationActions(t *testing.T) {
	sp := scene.Presets[0].Spec
	c := newTestController(t, sp, 300)
	waitSim(t, c, "first snapshot", func(sn *sim.Snapshot) bool { return sn.Len() > 0 })

	c.Do(ActionTogglePause)
	if c.Runner.Paused() {
		t.Error("toggle must resume")
	}
	c.Do(ActionStep)
	if !c.Runner.Paused() {
		t.Error("single step must pause")
	}

	c0 := c.Sim().C
	if c0 > 0 {
		c.Do(ActionLightFaster)
		waitSim(t, c, "c × 1.25", func(sn *sim.Snapshot) bool { return math.Abs(sn.C-c0*CStep) < 1e-9 })
		c.Do(ActionLightSlower)
		waitSim(t, c, "c back", func(sn *sim.Snapshot) bool { return math.Abs(sn.C-c0) < 1e-9 })
	}

	c.Do(ActionGravityRetarded)
	waitSim(t, c, "retarded", func(sn *sim.Snapshot) bool { return sn.Gravity == sim.GravityRetarded })
	c.Do(ActionCycleGravity)
	waitSim(t, c, "retarded+v", func(sn *sim.Snapshot) bool { return sn.Gravity == sim.GravityRetardedExtrapolated })
	c.Do(ActionGravityInstant)
	waitSim(t, c, "instant", func(sn *sim.Snapshot) bool { return sn.Gravity == sim.GravityInstant })

	c.Do(ActionToggleDarkMatter)
	waitSim(t, c, "dark matter off", func(sn *sim.Snapshot) bool { return sn.DarkMatterOff })

	other := scene.Presets[1].Spec
	c.Do(SceneAction(1))
	if c.Scene != other {
		t.Errorf("scene = %q, want %q", c.Scene.Name, other.Name)
	}
	c.Publish()
	if s := c.State(); s.Scene != 1 || s.SceneName != other.Name {
		t.Errorf("published scene %d %q", s.Scene, s.SceneName)
	}
	c.Do(ActionRestart)
	if c.Scene != other {
		t.Error("restart must keep the scene")
	}
}

func TestPlaceClusterAddsBoundGroupAtCursor(t *testing.T) {
	s := sim.NewState(1)
	s.Add(sim.Body{Mass: 1, Kind: sim.BlackHole})
	r := sim.NewRunner(sim.New(s, &direct.Solver{}, sim.Params{Dt: 1e-3, Softening: 0.05}))
	r.SetPaused(true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	c := New(r, 1000, scene.Presets[0].Spec)
	c.Cam.W, c.Cam.H, c.Cam.Pitch = 1000, 800, math.Pi/2
	c.Tool, c.Mass = ToolCluster, 0.5
	c.Place(700, 400, vec.Vec3{Y: 0.3})
	want := 1 + clusterStars(1000)
	center := c.Cam.Unproject(700, 400)
	waitSim(t, c, "cluster", func(sn *sim.Snapshot) bool {
		if sn.Len() != want {
			return false
		}
		var cx float64
		for i := 1; i < sn.Len(); i++ {
			cx += float64(sn.X[i]*sn.Mass[i]) / c.Mass
		}
		if math.Abs(cx-center.X) > 0.1 {
			t.Errorf("cluster center x = %.3f, want ≈ %.3f", cx, center.X)
		}
		return true
	})
}

func TestLaunchVelocityFollowsDragInFocalPlane(t *testing.T) {
	cam := view.NewCamera()
	cam.W, cam.H, cam.Pitch = 1000, 800, math.Pi/2 // top-down: screen right = +x, screen up = +y
	v := LaunchVelocity(&cam, 500, 400, 600, 400)
	if v.X <= 0 || math.Abs(v.Y) > 1e-9 || math.Abs(v.Z) > 1e-9 {
		t.Errorf("drag to the right should launch along +x, got %v", v)
	}
	dist := cam.Unproject(600, 400).Sub(cam.Unproject(500, 400)).Len()
	if math.Abs(v.Len()-LaunchScale*dist) > 1e-12 {
		t.Errorf("speed %v, want %v", v.Len(), LaunchScale*dist)
	}
}

func TestQuickSaveAndLoad(t *testing.T) {
	sp := scene.Presets[2].Spec
	c := newTestController(t, sp, 500)
	c.Cam.Yaw = 1.25
	c.Do(ActionQuickSave)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if m, _ := filepath.Glob(filepath.Join(c.SaveDir, "*.galaxy")); len(m) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("save file was not written")
		}
		time.Sleep(10 * time.Millisecond)
	}
	c.Cam.Yaw = 0
	c.Do(SceneAction(0)) // something else in between
	c.Do(ActionQuickLoad)
	if c.Cam.Yaw != 1.25 || c.Scene.Name != sp.Name {
		t.Errorf("camera %v / scene %q not restored", c.Cam.Yaw, c.Scene.Name)
	}
	waitSim(t, c, "restored bodies", func(sn *sim.Snapshot) bool { return sn.Len() == 500 })
}

func TestQuickLoadWithoutSavesNotifies(t *testing.T) {
	c := newTestController(t, scene.Presets[0].Spec, 200)
	c.SaveDir = filepath.Join(t.TempDir(), "missing")
	c.Do(ActionQuickLoad)
	if _, ok := c.Message(); !ok {
		t.Error("missing saves must be reported")
	}
}

func TestDefaultSaveDirIsPerUser(t *testing.T) {
	dir := DefaultSaveDir()
	if base, err := os.UserConfigDir(); err == nil && !strings.HasPrefix(dir, base) {
		t.Errorf("save dir %q not under %q", dir, base)
	}
}

func TestPostIsSafeFromOtherGoroutines(t *testing.T) {
	c := &Controller{Mass: 1}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				c.PostAction(ActionToggleGlow)
			}
		}()
	}
	wg.Wait()
	c.Drain()
	if c.FX.Glow {
		t.Error("80 toggles must leave glow off")
	}
}

func TestTopicListStartsWithIntroAndSceneTopics(t *testing.T) {
	list := topicList([]string{"dust", "gas"})
	if list[0] != IntroTopic || list[1] != "dust" || list[2] != "gas" || len(list) != 1+len(learn.IDs()) {
		t.Errorf("unexpected topic list %v", list)
	}
}

func TestHintFocusesTopic(t *testing.T) {
	c := &Controller{Scene: scene.Presets[0].Spec}
	c.Topics = topicList(c.Scene.Topics)
	c.Do(ActionToggleDust)
	if c.Topics[c.TopicIdx] != "dust" {
		t.Errorf("focused %q, want dust", c.Topics[c.TopicIdx])
	}
	if msg, ok := c.Message(); !ok || !strings.Contains(msg, "Staub") {
		t.Errorf("hint not shown: %q", msg)
	}
	title, body := c.Explanation()
	if title != "Staub" || body == "" {
		t.Errorf("explanation %q / %q", title, body)
	}
}

// TestNoEbitengine keeps the core usable by native front ends without Ebitengine (T-049).
func TestNoEbitengine(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", "galaxy/internal/app").Output()
	if err != nil {
		t.Skip("go list unavailable:", err)
	}
	if strings.Contains(string(out), "hajimehoshi/ebiten") {
		t.Error("internal/app must not depend on Ebitengine")
	}
}
