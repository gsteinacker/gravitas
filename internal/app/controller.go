package app

import (
	"fmt"
	"math"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"galaxy/internal/i18n"
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
	ToolGasCloud
	ToolGalaxy
	toolCount
)

func (t Tool) String() string {
	switch t {
	case ToolCluster:
		return "star cluster"
	case ToolGasCloud:
		return "gas cloud"
	case ToolGalaxy:
		return "galaxy"
	}
	return "black hole"
}

// Gas clouds (T-063): temperature and turbulence of the next placed cloud. The turbulence is the
// velocity dispersion per component as a fraction of the virial speed √(GM/R): below about 0.6 the
// cloud is bound and collapses (Jeans), above it disperses.
const (
	CloudTempMin, CloudTempMax = 10.0, 1e4
	CloudTurbMax               = 2.0
)

// cloudRadius is the Plummer scale radius of a gas cloud of the given mass (simulation units):
// Larson's relation R ≈ 20 pc · (M/10⁶ M☉)^½ in scenes with units, else 1.5 times a cluster.
func (c *Controller) cloudRadius(mass float64) float64 {
	if mu, pc, ok := c.units(); ok {
		return c.resolved(20 * math.Sqrt(mass*mu/1e6) / pc)
	}
	return 1.5 * clusterRadius(mass)
}

// resolved enlarges a radius below the resolution of the scene to twice the gravitational
// softening: smaller objects would lose their self-gravity to the softening and fall apart.
func (c *Controller) resolved(r float64) float64 {
	if c.Scene == nil {
		return r
	}
	return math.Max(r, 2*c.Scene.Params.Softening)
}

// ResolutionLimited reports whether the next object of the current tool is enlarged to the
// resolution of the scene (its realistic radius is below twice the softening).
func (c *Controller) ResolutionLimited() bool {
	mu, pc, ok := c.units()
	if !ok || c.Scene == nil {
		return false
	}
	eps2 := 2 * c.Scene.Params.Softening
	switch c.Tool {
	case ToolCluster:
		return 3*math.Cbrt(c.Mass*mu/1e5)/pc < eps2
	case ToolGasCloud:
		return 20*math.Sqrt(c.Mass*mu/1e6)/pc < eps2
	}
	return false
}

// clusterRadius is the Plummer scale radius of a star cluster of the given mass: r ≈ 3 pc ·
// (M/10⁵ M☉)^⅓ (globular clusters) in scenes with units, else a scene-free rule.
func (c *Controller) clusterRadius(mass float64) float64 {
	if mu, pc, ok := c.units(); ok {
		return c.resolved(3 * math.Cbrt(mass*mu/1e5) / pc)
	}
	return clusterRadius(mass)
}

// ToolRadius is the radius a cluster or gas cloud of the given mass gets when placed (simulation
// units, at least the resolution of the scene), 0 for other tools.
func (c *Controller) ToolRadius(tool Tool, mass float64) float64 {
	switch tool {
	case ToolCluster:
		return c.clusterRadius(mass)
	case ToolGasCloud:
		return c.cloudRadius(mass)
	}
	return 0
}

// Units returns solar masses and parsec per simulation unit of the scene; ok is false without
// physical units.
func (c *Controller) Units() (msun, pc float64, ok bool) { return c.units() }

// units returns solar masses and parsec per simulation unit of the scene.
func (c *Controller) units() (msun, pc float64, ok bool) {
	if c.Scene == nil {
		return 0, 0, false
	}
	msun, okM := c.Scene.MassUnit()
	_, pc, okL := c.Scene.Units()
	return msun, pc, okM && okL
}

// GalaxyScale reports whether the scene is large enough for placing galaxies (length unit ≥ 100 pc).
func (c *Controller) GalaxyScale() bool {
	_, pc, ok := c.units()
	return ok && pc >= 100
}

// defaultMasses are realistic starting masses per tool for the scale of the scene (simulation
// units). They must also be resolvable: in a galaxy scene (softening ~150 pc) a globular cluster
// (10⁵ M☉, 3 pc) or a giant molecular cloud (10⁶ M☉, 20 pc) would be enlarged to the resolution
// and torn apart by the tides at once. So galaxy scenes start with a Sgr A*-like hole, a massive
// cluster like the nucleus of a dwarf galaxy (3·10⁷ M☉, bound against the tides), a giant gas
// complex (10⁸ M☉, collapses even at 10⁴ K) and an LMC-sized galaxy; parsec scenes with a stellar
// black hole, an open cluster and a small cloud.
func (c *Controller) defaultMasses() [toolCount]float64 {
	mu, pc, ok := c.units()
	if !ok {
		return [toolCount]float64{0.5, 0.5, 0.5, 5}
	}
	msun := [toolCount]float64{10, 1e3, 1e3, 2e10}
	if pc >= 100 {
		msun = [toolCount]float64{4e6, 3e7, 1e8, 2e10}
	}
	var m [toolCount]float64
	for i, v := range msun {
		m[i] = math.Max(MassMin, math.Min(MassMax, v/mu))
	}
	return m
}

// cloudParticles is the number of gas particles of a placed cloud for a scene of n bodies.
func cloudParticles(n int) int { return max(400, n/40) }

// Mass bounds and factor of the next placed object.
const (
	MassMin  = 1e-12 // a solar mass in galaxy units is ~5·10⁻¹¹
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
	toolMass [toolCount]float64 // the mass each tool remembers
	// GalaxyTilt is the angle (degrees) between a placed galaxy's disk and the launch plane:
	// 0 spins with the host, 180 against it.
	GalaxyTilt float64
	// PlaceNormal and PlaceRadial orient placed galaxies (set by the launch planner); zero: z, x.
	PlaceNormal, PlaceRadial vec.Vec3
	// HoleSpin is the spin a* of the next black hole, about PlaceNormal (T-071).
	HoleSpin float64
	// CloudK and CloudTurb are the temperature (K) and turbulence (fraction of the virial speed)
	// of the next gas cloud.
	CloudK, CloudTurb float64
	// CloudRot and ClusterRot rotate the next cloud and cluster about PlaceNormal (T-082): extra
	// azimuthal speed in units of the circular speed (β = CloudRot²/3), share of stars turned
	// to orbit in the same sense.
	CloudRot, ClusterRot float64
	ShowHUD              bool
	ShowHelp             bool
	// FPS is the frame rate measured by the front end.
	FPS float64
	// Theta and GPU mirror the Barnes-Hut solver settings (set by main, changed by SetTheta/SetGPU).
	Theta     float64 // in use
	ThetaUser float64 // chosen by the user (flag or inspector); the governor may raise Theta
	GPU       bool
	// RenderScaleNow is the render pixels per window point in use, for display.
	RenderScaleNow float64
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
	// StarsFor, if set, chooses the star count of a scene (performance profile); otherwise N is used.
	StarsFor func(*scene.Spec) int
	// SceneLoaded, if set, is called after a scene or saved state replaced the running one.
	SceneLoaded func(*scene.Spec)

	placements uint64 // seeds successive clusters differently
	message    atomic.Pointer[Note]
	mu         sync.Mutex
	posted     []func(*Controller)
	sim        SimState
	rates      rateMeter
	bar        barMeter
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
	HolesFormed                                int
	Supernovae                                 int     // populations exploded (sim.Simulation.Supernovae)
	GasShare                                   float64 // gas mass / (gas + star mass)
	Infall                                     bool    // gas falls in from the halo
	ScaleFactor                                float64 // comoving mode: a (redshift z = 1/a − 1), else 0
	Time, C                                    float64
	Substeps, HydroSubsteps                    int
	Gravity                                    sim.GravityMode
	DarkMatterOff                              bool
	Bar                                        BarState // bar of the galaxy in galaxy scenes (T-090)
}

// New creates the controller for a runner that already simulates sp, built for n bodies.
func New(runner *sim.Runner, n int, sp *scene.Spec) *Controller {
	c := &Controller{
		Runner: runner, N: n, Speed: len(SpeedLevels) - 1, Mass: 0.5, CloudK: 100, CloudTurb: 0.3, CloudRot: 0.25, ClusterRot: 0.3,
		ShowHUD: true, ShowHelp: true, SaveDir: DefaultSaveDir(),
	}
	c.applyScene(sp)
	c.Publish()
	return c
}

// DefaultSaveDir is the per-user save directory (~/Library/Application Support/Gravitas/saves on
// macOS), or "saves" if the system has no config directory.
func DefaultSaveDir() string {
	dir := prefs.Dir()
	if dir == "" {
		return "saves"
	}
	return filepath.Join(dir, "saves")
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
		c.SetTool(ToolBlackHole)
	case ActionToolCluster:
		c.SetTool(ToolCluster)
	case ActionToolGasCloud:
		c.SetTool(ToolGasCloud)
	case ActionToolGalaxy:
		c.SetTool(ToolGalaxy)
	case ActionNextTool:
		next := (c.Tool + 1) % toolCount
		if next == ToolGalaxy && !c.GalaxyScale() {
			next = 0
		}
		c.SetTool(next)
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

// SetMass sets the mass of the next placed object, clamped to [MassMin, MassMax]; the tool keeps it.
func (c *Controller) SetMass(m float64) {
	c.Mass = math.Max(MassMin, math.Min(MassMax, m))
	c.toolMass[c.Tool] = c.Mass
}

// SetTool chooses the tool; each tool keeps its own mass.
func (c *Controller) SetTool(t Tool) {
	t = Tool(max(0, min(int(toolCount)-1, int(t))))
	if t == ToolGalaxy && !c.GalaxyScale() && c.Scene != nil {
		c.Notify(i18n.T("Galaxien passen nur in Szenen im Galaxienmaßstab", "galaxies only fit into scenes at galaxy scale"))
		return
	}
	c.toolMass[c.Tool] = c.Mass
	c.Tool = t
	if m := c.toolMass[t]; m > 0 {
		c.Mass = m
	}
}

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
				c.Notify(i18n.T("Sichern fehlgeschlagen: ", "save failed: ") + err.Error())
				return
			}
			c.Notify(i18n.T("Gesichert: ", "saved ") + path)
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
	c.Notify(i18n.T("Geladen: ", "loaded ") + path)
	return nil
}

func (c *Controller) saveCamera() save.Camera {
	k := c.Cam
	return save.Camera{Target: [3]float64{k.Target.X, k.Target.Y, k.Target.Z}, Yaw: k.Yaw, Pitch: k.Pitch, Distance: k.Distance}
}

// LoadScene replaces the running scene.
func (c *Controller) LoadScene(sp *scene.Spec) {
	c.applyScene(sp)
	if c.StarsFor != nil {
		c.N = c.StarsFor(sp)
	}
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
	c.toolMass = c.defaultMasses()
	if c.Tool == ToolGalaxy && !c.GalaxyScale() {
		c.Tool = ToolBlackHole
	}
	c.Mass = c.toolMass[c.Tool]
	c.Topics, c.TopicIdx = topicList(sp.Topics), 0
	if len(sp.Intro) > 0 {
		c.Notify(sp.Name + i18n.T(": e öffnet eine Einführung", ": e opens an introduction"))
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
	if sp.Paused { // a static scene: shown, not stepped (Space starts it)
		c.Runner.SetPaused(true)
	}
	if c.SceneLoaded != nil {
		c.SceneLoaded(sp)
	}
}

// Place adds the current tool's object at screen position (x, y) in the camera's focal plane with
// velocity vel.
func (c *Controller) Place(x, y float64, vel vec.Vec3) { c.PlaceAt(c.Cam.Unproject(x, y), vel) }

// PlaceAt adds the current tool's object at world position pos with coordinate velocity vel.
func (c *Controller) PlaceAt(pos, vel vec.Vec3) {
	mass := c.Mass
	if c.hinted == nil {
		c.hinted = map[string]bool{}
	}
	if topic := map[Tool]string{ToolBlackHole: "black-holes", ToolCluster: "gravity", ToolGasCloud: "gas-clouds", ToolGalaxy: "tides"}[c.Tool]; !c.hinted[topic] {
		c.hinted[topic] = true // once per run, placing is frequent
		c.hint(topic)
	}
	if c.Tool == ToolBlackHole {
		c.Runner.Do(func(m *sim.Simulation) {
			m.AddBody(sim.Body{Pos: pos, Vel: sim.ProperVelocity(vel, m.Params.C), Mass: mass, Kind: sim.BlackHole,
				Spin: scene.SpinVector(c.HoleSpin, c.PlaceNormal, mass, m.Params.C)})
		})
		return
	}
	if c.Tool == ToolGasCloud {
		c.placeCloud(pos, vel)
		return
	}
	if c.Tool == ToolGalaxy {
		c.placeGalaxy(pos, vel)
		return
	}
	// Build the cluster here, so the simulation goroutine only appends.
	c.placements++
	n := clusterStars(c.N)
	s := sim.NewState(n)
	scene.AddPlummer(s, n, mass, c.clusterRadius(mass), pos, vel, 1000+c.placements)
	scene.SpinUpCluster(s, 0, pos, vel, c.PlaceNormal, c.ClusterRot, 1000+c.placements)
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
		HolesFormed: sn.HolesFormed, ScaleFactor: sn.ScaleFactor, Supernovae: sn.Supernovae,
	}
	if s.Supernovae > 0 && c.sim.Supernovae == 0 && s.Time >= c.sim.Time {
		c.Notify(i18n.T("Supernovae: Die massereichen Sterne einer jungen Population explodieren und heizen das Gas auf.",
			"Supernovae: the massive stars of a young population explode and heat the gas."))
		c.focus("supernovae")
	}
	c.rates.add(sn.Time, sn.FormedMass, sn.SNMass, sn.InfallMass, rateWindow(c.Scene))
	s.Infall = sn.Infall
	c.bar.observe(sn, c.Scene)
	s.Bar = c.bar.state(c.Scene)
	var gas, stars float64
	if s.HolesFormed > c.sim.HolesFormed && s.Time >= c.sim.Time { // not after a restart (T-067)
		c.Notify(i18n.T("Ein Schwarzes Loch ist entstanden: die Masse passte in ihren eigenen Schwarzschild-Radius.",
			"A black hole has formed: the mass fit inside its own Schwarzschild radius."))
		c.focus("hole-formation")
	}
	for i, k := range sn.Kind {
		switch k {
		case sim.BlackHole:
			s.BlackHoles++
		case sim.DarkMatter:
			s.Dark++
		case sim.Gas:
			s.Gas++
			gas += float64(sn.Mass[i])
		case sim.Star:
			stars += float64(sn.Mass[i])
		}
	}
	if gas+stars > 0 {
		s.GasShare = gas / (gas + stars)
	}
	c.sim = s
}

// Sim is what the controller last saw of the simulation.
func (c *Controller) Sim() SimState { return c.sim }

// FormatYears picks a readable unit: yr, kyr, Myr or Gyr (German: J., Tsd. J., Mio. J., Mrd. J.).
func FormatYears(y float64) string {
	switch a := math.Abs(y); {
	case a >= 1e9:
		return fmt.Sprintf("%.3g %s", y/1e9, i18n.T("Mrd. J.", "Gyr"))
	case a >= 1e6:
		return fmt.Sprintf("%.3g %s", y/1e6, i18n.T("Mio. J.", "Myr"))
	case a >= 1e3:
		return fmt.Sprintf("%.3g %s", y/1e3, i18n.T("Tsd. J.", "kyr"))
	default:
		return fmt.Sprintf("%.3g %s", y, i18n.T("J.", "yr"))
	}
}

// placeCloud adds a gas cloud of the current mass at pos with velocity vel: Plummer profile,
// temperature CloudK, turbulence CloudTurb · √(M/R) and 30 % density fluctuations. Built here, so
// the simulation goroutine only appends.
func (c *Controller) placeCloud(pos, vel vec.Vec3) {
	c.placements++
	n, mass := cloudParticles(c.N), c.Mass
	r := c.cloudRadius(mass)
	u := c.Scene.SpecificEnergy(c.CloudK)
	turb := c.CloudTurb * math.Sqrt(mass/r)
	s := sim.NewState(n)
	scene.AddGasClump(s, n, mass, r, pos, vel, u, turb, 0.3, 2000+c.placements)
	scene.SpinUpCloud(s, 0, pos, c.PlaceNormal, c.CloudRot)
	c.Runner.Do(func(m *sim.Simulation) {
		for i := range s.Len() {
			b := s.Body(i)
			b.Vel = sim.ProperVelocity(b.Vel, m.Params.C)
			m.AddBody(b)
		}
	})
}

// SetCloud sets temperature (K) and turbulence (fraction of the virial speed) of the next cloud.
func (c *Controller) SetCloud(kelvin, turb float64) {
	c.CloudK = math.Max(CloudTempMin, math.Min(CloudTempMax, kelvin))
	c.CloudTurb = math.Max(0, math.Min(CloudTurbMax, turb))
}

// galaxyParticles is the number of bodies of a placed galaxy for a scene of n bodies.
func galaxyParticles(n int) int { return max(2000, n/4) }

// placeGalaxy adds a galaxy like the template (scene.AddTemplateGalaxy) of the current mass at pos
// with velocity vel, its disk in the launch plane tilted by GalaxyTilt around the radial axis.
func (c *Controller) placeGalaxy(pos, vel vec.Vec3) {
	mu, _, ok := c.units()
	if !ok {
		return
	}
	c.placements++
	s := sim.NewState(0)
	if !c.Scene.AddTemplateGalaxy(s, galaxyParticles(c.N), c.Mass*mu, 3000+c.placements) {
		return
	}
	n, r := c.PlaceNormal, c.PlaceRadial
	if n.Len() == 0 {
		n, r = vec.Vec3{Z: 1}, vec.Vec3{X: 1}
	}
	// The disk normal: n tilted by GalaxyTilt around r; then the frame (r, n'×r, n') maps x, y, z,
	// so the template's spin around +z becomes a spin around n'.
	tilt := c.GalaxyTilt * math.Pi / 180
	t := n.Cross(r)
	nz := n.Scale(math.Cos(tilt)).Sub(t.Scale(math.Sin(tilt)))
	ny := nz.Cross(r)
	rot := func(v vec.Vec3) vec.Vec3 { return r.Scale(v.X).Add(ny.Scale(v.Y)).Add(nz.Scale(v.Z)) }
	c.Runner.Do(func(m *sim.Simulation) {
		for i := range s.Len() {
			b := s.Body(i)
			b.Pos = rot(b.Pos).Add(pos)
			b.Vel = sim.ProperVelocity(rot(b.Vel).Add(vel), m.Params.C)
			m.AddBody(b)
		}
	})
}

// CloudForecast is the virial parameter α = 5 (σ² + c_s²) R / (G M) of the next gas cloud with
// its turbulence and temperature (at least the scene's floor temperature, returned as kelvin):
// below ~1 the cloud collapses, above ~2 it disperses.
func (c *Controller) CloudForecast() (alpha, kelvin float64) {
	kelvin = c.CloudK
	u := 0.0
	if c.Scene != nil {
		kelvin = math.Max(kelvin, c.Scene.GasFloorK)
		u = c.Scene.SpecificEnergy(kelvin)
	}
	r := c.cloudRadius(c.Mass)
	return 5*c.CloudTurb*c.CloudTurb + 10.0/3*u*r/c.Mass, kelvin
}
