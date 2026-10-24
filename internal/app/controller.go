package app

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"galaxy/internal/learn"
	"galaxy/internal/prefs"
	"galaxy/internal/save"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/vec"
	"galaxy/internal/view"
)

// SpeedLevels are the selectable simulation step rates; 0 means unlimited.
var SpeedLevels = []int{10, 30, 60, 120, 240, 0}

// Tool is what a click or Alt-drag places.
type Tool int

const (
	ToolBlackHole Tool = iota
	ToolCluster
)

func (t Tool) String() string {
	if t == ToolCluster {
		return "star cluster"
	}
	return "black hole"
}

// Mass bounds and factor of the next placed object.
const (
	MassMin  = 0.01
	MassMax  = 20.0
	massStep = 1.5
)

// The speed of light changes by CStep per action, within [CMin, CMax].
const (
	CStep = 1.25
	CMin  = 1.0
	CMax  = 1000.0
)

// clusterStars is the number of stars in a placed cluster for a scene of n stars.
func clusterStars(n int) int { return max(300, n/40) }

// clusterRadius is the Plummer scale radius of a cluster of the given mass: denser for light
// clusters, so they stay visibly compact.
func clusterRadius(mass float64) float64 { return 0.2 * math.Cbrt(mass/0.5) }

// DefaultEffects are used for scenes that do not name their own.
var DefaultEffects = view.Effects{Glow: true, Doppler: true, Lensing: true, Dust: true}

// Controller owns the interactive state and executes actions on the simulation. Its methods must
// be called from one goroutine (the front end's update loop); other goroutines hand work over
// with Post and read the published State.
type Controller struct {
	Runner   *sim.Runner
	Scene    *scene.Spec
	N        int // stars of a built scene
	Cam      view.Camera
	FX       view.Effects
	Speed    int // index into SpeedLevels
	Mass     float64
	Tool     Tool
	ShowHUD  bool
	ShowHelp bool
	// FPS is the frame rate measured by the front end.
	FPS float64
	// Theta and GPU mirror the Barnes-Hut solver settings (set by main, changed by SetTheta/SetGPU).
	Theta float64
	GPU   bool
	// Prefs are the persistent settings, saved to PrefsPath; nil disables persistence.
	Prefs     *prefs.Prefs
	PrefsPath string

	// Learning mode.
	Explain  bool
	Level    learn.Level
	Topics   []string
	TopicIdx int
	hinted   map[string]bool

	// SaveDir is where quick save stores and quick load looks for saved states.
	SaveDir string
	// ToggleFullscreen is provided by the front end; nil disables the action.
	ToggleFullscreen func()
	// SceneLoaded, if set, is called after a scene or saved state replaced the running one.
	SceneLoaded func(*scene.Spec)

	placements uint64 // seeds successive clusters differently
	message    atomic.Pointer[Note]
	mu         sync.Mutex
	posted     []func(*Controller)
	sim        SimState
	state      atomic.Pointer[State]
}

// Note is a short message for the user.
type Note struct {
	Text  string
	Until time.Time
}

// SimState is what the controller last saw of the simulation (from a snapshot).
type SimState struct {
	Bodies, BlackHoles, Dark, Gas, StarsFormed int
	Time, C                                    float64
	Substeps, HydroSubsteps                    int
	Gravity                                    sim.GravityMode
	DarkMatterOff                              bool
}

// New creates the controller for a runner that already simulates sp, built for n bodies.
func New(runner *sim.Runner, n int, sp *scene.Spec) *Controller {
	c := &Controller{
		Runner: runner, N: n, Speed: len(SpeedLevels) - 1, Mass: 0.5,
		ShowHUD: true, ShowHelp: true, SaveDir: DefaultSaveDir(),
	}
	c.applyScene(sp)
	c.Publish()
	return c
}

// DefaultSaveDir is the per-user save directory (~/Library/Application Support/Galaxy/saves on
// macOS), or "saves" if the system has no config directory.
func DefaultSaveDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "saves"
	}
	return filepath.Join(dir, "Galaxy", "saves")
}

// Post hands work to the update loop; safe to call from any goroutine.
func (c *Controller) Post(f func(*Controller)) {
	c.mu.Lock()
	c.posted = append(c.posted, f)
	c.mu.Unlock()
}

// PostAction is Post for a plain action.
func (c *Controller) PostAction(a Action) { c.Post(func(c *Controller) { c.Do(a) }) }

// Drain runs the posted work; call it once per update.
func (c *Controller) Drain() {
	c.mu.Lock()
	work := c.posted
	c.posted = nil
	c.mu.Unlock()
	for _, f := range work {
		f(c)
	}
}

// Do executes one action.
func (c *Controller) Do(a Action) {
	if i, ok := a.SceneIndex(); ok {
		c.LoadScene(scene.Presets[i].Spec)
		return
	}
	switch a {
	case ActionTogglePause:
		c.Runner.SetPaused(!c.Runner.Paused())
	case ActionStep:
		c.Runner.SetPaused(true)
		c.Runner.Do(func(m *sim.Simulation) { m.Step() })
	case ActionFaster:
		c.Speed = min(len(SpeedLevels)-1, c.Speed+1)
	case ActionSlower:
		c.Speed = max(0, c.Speed-1)
	case ActionLightSlower:
		c.ScaleC(1 / CStep)
	case ActionLightFaster:
		c.ScaleC(CStep)
	case ActionCycleGravity:
		c.SetGravity((c.sim.Gravity + 1) % 3)
	case ActionGravityInstant:
		c.SetGravity(sim.GravityInstant)
	case ActionGravityRetarded:
		c.SetGravity(sim.GravityRetarded)
	case ActionGravityRetardedV:
		c.SetGravity(sim.GravityRetardedExtrapolated)
	case ActionToggleDarkMatter:
		off := !c.sim.DarkMatterOff
		c.sim.DarkMatterOff = off
		c.Runner.Do(func(m *sim.Simulation) { m.SetDarkMatterOff(off) })
	case ActionToolBlackHole:
		c.Tool = ToolBlackHole
	case ActionToolCluster:
		c.Tool = ToolCluster
	case ActionNextTool:
		c.Tool = (c.Tool + 1) % 2
	case ActionMassUp:
		c.SetMass(c.Mass * massStep)
	case ActionMassDown:
		c.SetMass(c.Mass / massStep)
	case ActionToggleGlow:
		c.FX.Glow = !c.FX.Glow
	case ActionToggleTrails:
		c.FX.Trails = !c.FX.Trails
	case ActionToggleLens:
		c.FX.Lensing = !c.FX.Lensing
	case ActionToggleDoppler:
		c.FX.Doppler = !c.FX.Doppler
	case ActionToggleDark:
		c.FX.Dark = !c.FX.Dark
	case ActionToggleGas:
		c.FX.Gas = !c.FX.Gas
	case ActionToggleDust:
		c.FX.Dust = !c.FX.Dust
	case ActionToggleHUD:
		c.ShowHUD = !c.ShowHUD
	case ActionToggleHelp:
		c.ShowHelp = !c.ShowHelp
	case ActionCenterView:
		c.Cam.Target = vec.Vec3{}
	case ActionToggleFullscreen:
		if c.ToggleFullscreen != nil {
			c.ToggleFullscreen()
		}
	case ActionToggleExplain:
		c.Explain = !c.Explain
	case ActionCloseExplain:
		c.Explain = false
	case ActionNextTopic:
		c.TopicIdx = (c.TopicIdx + 1) % len(c.Topics)
	case ActionPrevTopic:
		c.TopicIdx = (c.TopicIdx + len(c.Topics) - 1) % len(c.Topics)
	case ActionNextLevel:
		c.Level = c.Level.Next()
	case ActionPrevLevel:
		c.Level = c.Level.Prev()
	case ActionRestart:
		c.LoadScene(c.Scene)
	case ActionQuickSave:
		c.Save(save.NewPath(c.SaveDir))
	case ActionQuickLoad:
		if path, err := save.Latest(c.SaveDir); err != nil {
			c.Notify(err.Error())
		} else if err := c.Load(path); err != nil {
			c.Notify(err.Error())
		}
	}
	if id, ok := topicFor[a]; ok {
		c.hint(id)
	}
}

// SpeedLimit is the selected step rate, 0 = unlimited.
func (c *Controller) SpeedLimit() int { return SpeedLevels[c.Speed] }

// SetMass sets the mass of the next placed object, clamped to [MassMin, MassMax].
func (c *Controller) SetMass(m float64) { c.Mass = math.Max(MassMin, math.Min(MassMax, m)) }

// SetGravity changes how the gravity of black holes propagates.
func (c *Controller) SetGravity(g sim.GravityMode) {
	c.sim.Gravity = g
	c.Runner.Do(func(m *sim.Simulation) {
		m.Params.Gravity = g
		m.Epoch++ // energy is not conserved in retarded modes; restart the drift measurement
	})
}

// ScaleC changes the speed of light of the running simulation. Stored proper velocities (and
// thus momenta) stay; coordinate velocities follow from the new c.
func (c *Controller) ScaleC(f float64) {
	if c.sim.C > 0 {
		c.sim.C = math.Max(CMin, math.Min(CMax, c.sim.C*f))
	}
	c.Runner.Do(func(m *sim.Simulation) {
		if m.Params.C > 0 {
			m.Params.C = math.Max(CMin, math.Min(CMax, m.Params.C*f))
			m.Epoch++ // kinetic energy depends on c
		}
	})
}

// Notify shows a short message; safe to call from any goroutine.
func (c *Controller) Notify(msg string) {
	c.message.Store(&Note{Text: msg, Until: time.Now().Add(4 * time.Second)})
}

// Message returns the current message, if it has not expired.
func (c *Controller) Message() (string, bool) {
	if n := c.message.Load(); n != nil && time.Now().Before(n.Until) {
		return n.Text, true
	}
	return "", false
}

// Save captures the running simulation on its goroutine and writes it in the background.
func (c *Controller) Save(path string) {
	sp, cam := c.Scene, c.saveCamera()
	c.Runner.Do(func(m *sim.Simulation) {
		f := save.Capture(m, sp, cam)
		go func() {
			if err := save.Write(path, f); err != nil {
				c.Notify("save failed: " + err.Error())
				return
			}
			c.Notify("saved " + path)
		}()
	})
}

// Load replaces the running simulation by a saved state, including camera and scene texts.
func (c *Controller) Load(path string) error {
	f, err := save.Read(path)
	if err != nil {
		return err
	}
	sp := f.Scene
	c.applyScene(&sp)
	c.Cam.Target = vec.Vec3{X: f.Camera.Target[0], Y: f.Camera.Target[1], Z: f.Camera.Target[2]}
	c.Cam.Yaw, c.Cam.Pitch, c.Cam.Distance = f.Camera.Yaw, f.Camera.Pitch, f.Camera.Distance
	c.Runner.Do(f.Restore)
	c.Notify("loaded " + path)
	return nil
}

func (c *Controller) saveCamera() save.Camera {
	k := c.Cam
	return save.Camera{Target: [3]float64{k.Target.X, k.Target.Y, k.Target.Z}, Yaw: k.Yaw, Pitch: k.Pitch, Distance: k.Distance}
}

// LoadScene replaces the running scene.
func (c *Controller) LoadScene(sp *scene.Spec) {
	c.applyScene(sp)
	s, p := sp.Build(c.N)
	c.Runner.Do(func(m *sim.Simulation) {
		// Propagation mode and dark matter switch are user choices, not part of the scene.
		p.Gravity, p.DarkMatterOff = m.Params.Gravity, m.Params.DarkMatterOff
		m.Reset(s, p)
	})
}

// applyScene takes over the camera, effects and texts of a scene.
func (c *Controller) applyScene(sp *scene.Spec) {
	c.Scene = sp
	c.Topics, c.TopicIdx = topicList(sp.Topics), 0
	if len(sp.Intro) > 0 {
		c.Notify(sp.Name + ": e öffnet eine Einführung")
	}
	w, h := c.Cam.W, c.Cam.H
	c.Cam = view.NewCamera()
	c.Cam.W, c.Cam.H = w, h
	if k := sp.Camera; k != nil {
		c.Cam.Target = vec.Vec3{X: k.Target[0], Y: k.Target[1], Z: k.Target[2]}
		c.Cam.Distance, c.Cam.Pitch, c.Cam.Yaw = k.Distance, k.Pitch*math.Pi/180, k.Yaw*math.Pi/180
	}
	c.FX = DefaultEffects
	if sp.Effects != "" {
		if fx, err := view.ParseEffects(sp.Effects); err == nil {
			c.FX = fx
		}
	}
	if c.SceneLoaded != nil {
		c.SceneLoaded(sp)
	}
}

// Place adds the current tool's object at screen position (x, y) with velocity vel.
func (c *Controller) Place(x, y float64, vel vec.Vec3) {
	pos := c.Cam.Unproject(x, y)
	mass := c.Mass
	if c.hinted == nil {
		c.hinted = map[string]bool{}
	}
	if topic := map[Tool]string{ToolBlackHole: "black-holes", ToolCluster: "gravity"}[c.Tool]; !c.hinted[topic] {
		c.hinted[topic] = true // once per run, placing is frequent
		c.hint(topic)
	}
	if c.Tool == ToolBlackHole {
		c.Runner.Do(func(m *sim.Simulation) {
			m.AddBody(sim.Body{Pos: pos, Vel: sim.ProperVelocity(vel, m.Params.C), Mass: mass, Kind: sim.BlackHole})
		})
		return
	}
	// Build the cluster here, so the simulation goroutine only appends.
	c.placements++
	n := clusterStars(c.N)
	s := sim.NewState(n)
	scene.AddPlummer(s, n, mass, clusterRadius(mass), pos, vel, 1000+c.placements)
	c.Runner.Do(func(m *sim.Simulation) {
		for i := range s.Len() {
			b := s.Body(i)
			b.Vel = sim.ProperVelocity(b.Vel, m.Params.C)
			m.AddBody(b)
		}
	})
}

// LaunchScale converts an Alt-drag distance in world units into a launch velocity.
// Dragging across a typical galaxy radius (~2) gives roughly its orbital speed (~1).
const LaunchScale = 0.5

// LaunchVelocity is the velocity of a black hole launched by dragging from screen point
// (x0, y0) to (x1, y1). Both points are taken in the camera's focal plane.
func LaunchVelocity(cam *view.Camera, x0, y0, x1, y1 float64) vec.Vec3 {
	return cam.Unproject(x1, y1).Sub(cam.Unproject(x0, y0)).Scale(LaunchScale)
}

// Observe takes over what the front end saw in the latest snapshot.
func (c *Controller) Observe(sn *sim.Snapshot) {
	s := SimState{
		Bodies: sn.Len(), Time: sn.Time, C: sn.C, Substeps: sn.Substeps, HydroSubsteps: sn.HydroSubsteps,
		Gravity: sn.Gravity, DarkMatterOff: sn.DarkMatterOff, StarsFormed: sn.StarsFormed,
	}
	for _, k := range sn.Kind {
		switch k {
		case sim.BlackHole:
			s.BlackHoles++
		case sim.DarkMatter:
			s.Dark++
		case sim.Gas:
			s.Gas++
		}
	}
	c.sim = s
}

// Sim is what the controller last saw of the simulation.
func (c *Controller) Sim() SimState { return c.sim }

// FormatYears picks a readable unit: yr, kyr, Myr or Gyr.
func FormatYears(y float64) string {
	switch a := math.Abs(y); {
	case a >= 1e9:
		return fmt.Sprintf("%.3g Gyr", y/1e9)
	case a >= 1e6:
		return fmt.Sprintf("%.3g Myr", y/1e6)
	case a >= 1e3:
		return fmt.Sprintf("%.3g kyr", y/1e3)
	default:
		return fmt.Sprintf("%.3g yr", y)
	}
}
