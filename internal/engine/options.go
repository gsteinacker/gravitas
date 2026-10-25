package engine

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/scene"
	"galaxy/internal/sim"
	"galaxy/internal/sim/bh"
	"galaxy/internal/view"
)

// Options are the start parameters of the app, parsed from its command line (CLAUDE.md, docs).
type Options struct {
	Preset    int    // built-in scene, 1-based
	Stars     int    // 0 = from the performance profile (screenshots: 50000)
	SceneFile string // scene JSON instead of the preset
	LoadFile  string // saved state to start from
	C         float64
	Gravity   string
	FX        string
	Explain   string // level[:topic]
	Lang      string
	Theta     float64
	GPU       bool
	GPUSet    bool    // -gpu given explicitly
	Scale     float64 // render pixels per point, 0 = adaptive (screenshots: 1)
	Adapt     bool    // screenshots: calibrate and govern like the app
	HUD       bool    // status overlay
	Help      bool    // key overview
	Launch    string  // plan a launch at "x,y[,preset[,go]]" (fractions of the view; preset as LaunchPreset; go 1 launches)
	Tool      string  // hole, cluster or cloud
	Mass      float64 // mass of the placed object (simulation units), 0 = default
	Shot      string  // save a screenshot after ShotAfter seconds and quit
	ShotAfter float64
}

// Screenshot reports whether the app only takes a screenshot.
func (o *Options) Screenshot() bool { return o.Shot != "" }

// ParseOptions reads the command line (without the program name). Unknown arguments, e.g. those
// macOS adds when launching an app (-NSDocumentRevisionsDebugMode, -psn_…), are ignored.
func ParseOptions(args []string) (*Options, error) {
	o := &Options{}
	fs := flag.NewFlagSet("galaxy", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.IntVar(&o.Preset, "preset", 1, "built-in start scene (1-based)")
	fs.IntVar(&o.Stars, "n", 0, "number of stars (0 = from the performance profile; screenshots: 50000)")
	fs.StringVar(&o.SceneFile, "scene", "", "start scene from a JSON file (see docs/scenes.md)")
	fs.StringVar(&o.LoadFile, "load", "", "start from a saved state (.galaxy)")
	fs.Float64Var(&o.C, "c", 0, "speed of light (0 = scene default, -1 = Newtonian)")
	fs.StringVar(&o.Gravity, "gravity", "instant", "propagation of black hole gravity: instant, retarded, retarded+v")
	fs.StringVar(&o.FX, "fx", "", "render effects: glow, lens, doppler, trails, dark, gas, dust")
	fs.StringVar(&o.Explain, "explain", "", "open the learning panel: school, lay or physics (optionally level:topic)")
	fs.StringVar(&o.Lang, "lang", "", "user interface language: de or en (default: system)")
	fs.Float64Var(&o.Theta, "theta", bh.DefaultTheta, "Barnes-Hut opening angle (smaller = more accurate)")
	fs.BoolVar(&o.GPU, "gpu", true, "run the tree walk on the GPU if available")
	fs.Float64Var(&o.Scale, "scale", 0, "render pixels per window point, e.g. 2 for Retina (0 = adaptive; screenshots: 1)")
	fs.BoolVar(&o.Adapt, "adapt", false, "screenshots: adapt quality and star count like the app")
	fs.BoolVar(&o.HUD, "hud", false, "show the status overlay")
	fs.BoolVar(&o.Help, "help", false, "show the key overview")
	fs.StringVar(&o.Launch, "launch", "", "plan a launch at x,y (fractions of the view) with preset 0 rest, 1 circular, 2 escape, 3 center, 4 binary; ,1 launches")
	fs.StringVar(&o.Tool, "tool", "", "tool for placing: hole, cluster or cloud")
	fs.Float64Var(&o.Mass, "mass", 0, "mass of placed objects (simulation units)")
	fs.StringVar(&o.Shot, "shot", "", "save a screenshot to this PNG file after -shot-after seconds and quit")
	fs.Float64Var(&o.ShotAfter, "shot-after", 4, "seconds before the screenshot is taken")
	var known []string
	for i := 0; i < len(args); i++ {
		name := strings.TrimLeft(strings.SplitN(args[i], "=", 2)[0], "-")
		f := fs.Lookup(name)
		if !strings.HasPrefix(args[i], "-") || f == nil {
			continue
		}
		known = append(known, args[i])
		_, isBool := f.Value.(interface{ IsBoolFlag() bool })
		if !isBool && !strings.Contains(args[i], "=") && i+1 < len(args) {
			i++
			known = append(known, args[i])
		}
	}
	if err := fs.Parse(known); err != nil {
		return nil, err
	}
	fs.Visit(func(f *flag.Flag) { o.GPUSet = o.GPUSet || f.Name == "gpu" })
	// Paths are relative to the shell's directory; an app may start elsewhere.
	if dir := os.Getenv("PWD"); dir != "" {
		for _, p := range []*string{&o.SceneFile, &o.LoadFile, &o.Shot} {
			if *p != "" && !filepath.IsAbs(*p) {
				*p = filepath.Join(dir, *p)
			}
		}
	}
	if o.Screenshot() && !o.Adapt {
		if o.Stars <= 0 {
			o.Stars = defaultStars
		}
		if o.Scale <= 0 {
			o.Scale = 1
		}
	}
	return o, nil
}

// spec is the start scene.
func (o *Options) spec() (*scene.Spec, error) {
	if o.SceneFile != "" {
		return scene.LoadSpec(o.SceneFile)
	}
	return scene.Presets[max(0, min(len(scene.Presets)-1, o.Preset-1))].Spec, nil
}

// apply sets the options that act on a started engine.
func (o *Options) apply(e *Engine) error {
	if o.Lang != "" {
		i18n.Set(i18n.FromLocale(o.Lang))
	}
	if o.LoadFile != "" {
		if err := e.Load(o.LoadFile); err != nil {
			return err
		}
	}
	if o.FX != "" {
		fx, err := view.ParseEffects(o.FX)
		if err != nil {
			return err
		}
		e.FX = fx
	}
	if o.Explain != "" {
		level, topic, _ := strings.Cut(o.Explain, ":")
		e.OpenExplanation(level, topic)
	}
	e.ShowHUD, e.ShowHelp = o.HUD, o.Help
	switch o.Tool {
	case "":
	case "hole":
		e.Tool = app.ToolBlackHole
	case "cluster":
		e.Tool = app.ToolCluster
	case "cloud":
		e.Tool = app.ToolGasCloud
	default:
		return fmt.Errorf("unknown tool %q (want hole, cluster, cloud)", o.Tool)
	}
	if o.Mass > 0 {
		e.SetMass(o.Mass)
	}
	return nil
}

// params adjusts the scene parameters by -c and -gravity.
func (o *Options) params(p *sim.Params) error {
	mode, err := sim.ParseGravityMode(o.Gravity)
	if err != nil {
		return err
	}
	p.Gravity = mode
	switch {
	case o.C > 0:
		p.C = o.C
	case o.C < 0:
		p.C = 0
	}
	return nil
}

// ShotLine is the machine-readable summary printed with a screenshot (CLAUDE.md).
func (e *Engine) ShotLine(path string, fps float64) string {
	s := e.State()
	profile := e.settings.Profile
	if profile == "" {
		profile = "auto"
	}
	return fmt.Sprintf("shot %s: bodies=%d fps=%.1f steps_per_s=%.1f substeps=%d gas_substeps=%d t=%.2f profile=%s auto_stars=%v gpu=%v scale=%g theta=%.2f",
		path, s.Sim.Bodies, fps, s.StepsPerSec, s.Sim.Substeps, s.Sim.HydroSubsteps, s.Sim.Time, profile,
		e.StarsFor != nil, s.GPU, e.RenderScale(), s.Theta)
}

// pendingLaunch parses -launch: fractions of the view, an optional preset and whether to launch.
func (o *Options) pendingLaunch() (fx, fy float64, preset int, launch, ok bool) {
	if o.Launch == "" {
		return 0, 0, 0, false, false
	}
	preset = -1
	var g int
	n, _ := fmt.Sscanf(o.Launch, "%g,%g,%d,%d", &fx, &fy, &preset, &g)
	return fx, fy, preset, g == 1, n >= 2
}
