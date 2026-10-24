package perf

import "math"

// Governor keeps the app fluid at run time (T-047). Every tick (about twice per second) it looks
// at the frame and step rates and adjusts, in this order:
//
//  1. the render scale (pixels per window point, Retina = 2), which costs the GPU,
//  2. the Barnes-Hut opening angle θ within [ThetaBase, ThetaMax], which costs the simulation,
//  3. a cap on the steps per second, so the simulation does not starve rendering.
//
// The physics model stays the same: θ stays inside the measured error budget (ADR-0009). Effects
// the user switched on stay on. Changes need several consistent ticks (hysteresis) and are
// followed by a cooldown, so the picture does not flicker between levels.
type Governor struct {
	scales []float64 // allowed render scales, best first
	level  int       // index into scales
	// ThetaBase is the user's opening angle; the governor only adds to it.
	ThetaBase              float64
	thetaAdd               float64
	spsCap                 int
	slowFrames, fastFrames int
	slowSteps, fastSteps   int
	cooldown               int
}

// Thermal states as reported by macOS (NSProcessInfoThermalState).
const (
	ThermalNominal = iota
	ThermalFair
	ThermalSerious
	ThermalCritical
)

// Inputs are the measurements of one tick.
type Inputs struct {
	FPS         float64 // frames per second (display limited, usually ≤ 60)
	StepsPerSec float64 // simulation steps per second, 0 while paused
	TargetSPS   float64 // steps per second the profile aims at
	UserLimit   int     // steps per second the user chose, 0 = unlimited
	Thermal     int     // ThermalNominal … ThermalCritical
	LowPower    bool    // macOS low power mode
}

// Settings are what the governor wants applied.
type Settings struct {
	RenderScale float64
	Theta       float64
	SPSCap      int // 0 = no cap from the governor
}

// Thresholds of the governor.
const (
	slowFPS       = 48 // below: a frame is too slow
	fastFPS       = 57 // above: frames have headroom
	slowTicks     = 3  // consecutive slow ticks before degrading
	fastTicks     = 10 // consecutive fast ticks before improving (slower up than down)
	cooldownTicks = 4  // ticks to wait after a change
	thetaStep     = 0.05
	stepsLow      = 0.8 // steps/s below this fraction of the target are too slow
	stepsHigh     = 1.4 // above: room to make θ more accurate again
)

// ThetaMax is the largest opening angle the governor uses.
const ThetaMax = 1.0

// NewGovernor starts at the best render scale that does not exceed maxScale and deviceScale
// (pixels per point of the display).
func NewGovernor(maxScale, deviceScale, thetaBase float64) *Governor {
	top := math.Max(1, math.Min(maxScale, deviceScale))
	var scales []float64
	for _, s := range []float64{2, 1.5, 1, 0.75, 0.5} {
		if s <= top+1e-9 {
			scales = append(scales, s)
		}
	}
	return &Governor{scales: scales, ThetaBase: thetaBase}
}

// Current returns the settings without measuring.
func (g *Governor) Current() Settings {
	return Settings{RenderScale: g.scales[g.level], Theta: g.theta(), SPSCap: g.spsCap}
}

func (g *Governor) theta() float64 {
	return math.Min(math.Max(ThetaMax, g.ThetaBase), g.ThetaBase+g.thetaAdd)
}

// Level is a short description for status lines, e.g. "scale=1.5 theta=+0.05 cap=30".
func (g *Governor) Level() (scale, thetaAdd float64, cap int) {
	return g.scales[g.level], g.thetaAdd, g.spsCap
}

// Tick takes one measurement and returns the settings to apply; changed reports a difference to
// the previous tick.
func (g *Governor) Tick(in Inputs) (s Settings, changed bool) {
	before := g.Current()
	g.thermal(in)
	if g.cooldown > 0 {
		g.cooldown--
	} else {
		g.frames(in)
		g.steps(in)
	}
	s = g.Current()
	return s, s != before
}

// thermal reacts at once to heat and low power: a hot Mac throttles anyway, and the fans should
// not run because of a simulation.
func (g *Governor) thermal(in Inputs) {
	minLevel := 0
	switch {
	case in.Thermal >= ThermalCritical:
		minLevel = len(g.scales) - 1
	case in.Thermal >= ThermalSerious || in.LowPower:
		minLevel = min(len(g.scales)-1, index(g.scales, 1))
	}
	if g.level < minLevel {
		g.level = minLevel
		g.cooldown = cooldownTicks
	}
	if (in.Thermal >= ThermalSerious || in.LowPower) && in.TargetSPS > 0 {
		g.spsCap = int(in.TargetSPS)
	}
}

func index(scales []float64, v float64) int {
	for i, s := range scales {
		if s <= v+1e-9 {
			return i
		}
	}
	return len(scales) - 1
}

func (g *Governor) frames(in Inputs) {
	switch {
	case in.FPS > 0 && in.FPS < slowFPS:
		g.slowFrames++
		g.fastFrames = 0
	case in.FPS >= fastFPS:
		g.fastFrames++
		g.slowFrames = 0
	default:
		g.slowFrames, g.fastFrames = 0, 0
	}
	if g.slowFrames >= slowTicks {
		g.slowFrames = 0
		switch {
		case g.level < len(g.scales)-1:
			g.level++
		case g.spsCap == 0 && in.TargetSPS > 0 && (in.UserLimit == 0 || in.UserLimit > int(in.TargetSPS)):
			g.spsCap = int(in.TargetSPS) // rendering is at its lowest: leave it more CPU
		default:
			return
		}
		g.cooldown = cooldownTicks
	}
	if g.fastFrames >= fastTicks && in.Thermal < ThermalSerious && !in.LowPower {
		g.fastFrames = 0
		switch {
		case g.spsCap != 0:
			g.spsCap = 0
		case g.level > 0:
			g.level--
		default:
			return
		}
		g.cooldown = cooldownTicks
	}
}

func (g *Governor) steps(in Inputs) {
	if in.StepsPerSec <= 0 || in.TargetSPS <= 0 {
		g.slowSteps, g.fastSteps = 0, 0 // paused
		return
	}
	target := in.TargetSPS
	if in.UserLimit > 0 {
		target = math.Min(target, float64(in.UserLimit))
	}
	switch {
	case in.StepsPerSec < stepsLow*target:
		g.slowSteps++
		g.fastSteps = 0
	case in.StepsPerSec > stepsHigh*target:
		g.fastSteps++
		g.slowSteps = 0
	default:
		g.slowSteps, g.fastSteps = 0, 0
	}
	if g.slowSteps >= slowTicks && g.theta() < ThetaMax-1e-9 {
		g.slowSteps = 0
		g.thetaAdd += thetaStep
		g.cooldown = cooldownTicks
	}
	if g.fastSteps >= fastTicks && g.thetaAdd > 0 {
		g.fastSteps = 0
		g.thetaAdd = math.Max(0, g.thetaAdd-thetaStep)
		g.cooldown = cooldownTicks
	}
}
