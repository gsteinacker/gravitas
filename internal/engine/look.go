package engine

import (
	"math"
	"slices"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
	"galaxy/internal/view"
)

// Look is everything about a frame's appearance that is not drawing: the normalizations that make
// the picture adapt to the scene (T-033/T-037) and the black holes on
// screen. The front end draws with it; the rules stay in Go.
type Look struct {
	Exposure  float64 // star brightness factor (auto exposure)
	EmNorm    float64 // normalization of the Hα emission
	DarkAlpha float32 // opacity per dark matter sprite (haze), 0 if hidden
	// WebAlpha is the brightness of a dark matter point when a scene holds nothing else (the
	// cosmic web, T-073/T-074): the particles themselves are drawn, and their sum shows the
	// filaments, halos and voids. 0 otherwise; the haze is off then.
	WebAlpha float32
	GasAlpha float32 // opacity per gas sprite of the gas map, 0 if hidden
	// DustKappa is the dust opacity (V-band optical depth per unit column) for the dust image of
	// the background; 0 without dust. The attenuation of the bodies is in Prepare's attn buffer.
	DustKappa float64
	Holes     []Hole
	// World are all black holes in world space with their Schwarzschild radius (0 without c), for
	// the gravitational redshift of the stars around them.
	World [][4]float32
	// Pop draws star particles as stellar populations and star-forming gas (T-070).
	Pop Population
}

// Hole is a black hole on screen, in render pixels.
type Hole struct {
	X, Y  float32
	R     float32 // drawn horizon radius r₊ (at least minHolePixels)
	PhysR float32 // Schwarzschild radius 2GM/c², for the lens (spin does not change it to first order)
	Rings bool    // large enough to draw the ISCO ring
	// Spin a* (0: Schwarzschild) and the spin axis in screen space (x, y projected, z toward the
	// camera): the shader draws the horizon, photon ring and ISCO of a Kerr hole (T-071) and the
	// accretion disk in the plane perpendicular to it (T-065).
	Spin, AxisX, AxisY, AxisZ float32
	// Disk is the brightness of the accretion disk, 0 (none) to 1 (diskBrightness).
	Disk float32
}

// diskReference is the accretion rate, in hole masses per time unit, at which the disk shines with
// 63 % of its full brightness: a hole swallowing one star particle lights up for a few dozen steps.
const diskReference = 1e-3

// diskBrightness maps the recent accretion rate mdot of a hole of mass m to the brightness of
// its disk (0–1, saturating): the luminosity of a thin disk is ∝ Ṁ (T-065).
func diskBrightness(mdot, m float64) float64 {
	if mdot <= 0 || m <= 0 {
		return 0
	}
	return 1 - math.Exp(-mdot/m/diskReference)
}

// Constants of the auto exposure (T-033), the emission (T-037) and the haze.
const (
	gridW, gridH       = 96, 60
	exposurePercentile = 0.9
	exposureTarget     = 0.9
	smoothing          = 0.35
	emissionTarget     = 0.25
	emissionPercentile = 0.5
	minEmissionIon     = 0.2
	hazePercentile     = 0.9
	hazeTarget         = 0.5
	webTarget          = 0.2 // brightness of the densest 10 % of the cosmic web
	webPixels          = 3   // size of a cosmic web point in pixels
	minHolePixels      = 3
	spriteMean         = 0.10972 // mean of the glow sprite over its 16×16 quad (render.spriteMean)
	// kernelMean is the mean of the gas kernel (1 − d²)³ over its quad, π/16 (Galaxy.metal KERNEL_MEAN).
	kernelMean = math.Pi / 16
	// gasMapRef² is the drawn column of a gas body of mean star mass in pixels² of the gas map.
	gasMapRef = 10
)

// looker keeps the smoothed normalizations and the cache key between frames.
type looker struct {
	look            Look
	version         uint64
	view            view.View
	fx              view.Effects
	adapting        bool
	light, emLight  [gridW * gridH]float32
	darkGrid, gasGr [gridW * gridH]float32
	webGrid         [gridW * gridH]float32
	sortBuf         []float32
	dust            dustMap
	dustNorm        float64   // automatic dust opacity, 0 before the first frame
	attn            []float32 // r, g, b transmission and unused, per body
}

// Prepare computes the look of the bodies last copied by Frame (bodies holds them, FloatsPerBody
// floats each) for a viewport of w×h render pixels at pixel scale ps. With dust, it writes the
// transmission of every body (r, g, b, 1) to attn (4 floats per body), every call, so the front
// end may pass a different buffer per frame. It recomputes only when the snapshot, the camera,
// the effects or the size changed, or while the exposure still adapts.
func (e *Engine) Prepare(bodies, attn []float32, info FrameInfo, w, h, ps float64, fx view.Effects) Look {
	l := &e.looker
	cam := e.Cam
	cam.W, cam.H = w, h
	vw := cam.View()
	if info.Version == l.version && vw == l.view && fx == l.fx && !l.adapting {
		if fx.Dust {
			copy(attn, l.attn)
		}
		return l.look
	}
	l.version, l.view, l.fx = info.Version, vw, fx
	n := min(info.Bodies, len(bodies)/FloatsPerBody)
	focal := cam.Focal()
	cw, ch := w/gridW, h/gridH
	clear(l.light[:])
	clear(l.emLight[:])
	clear(l.darkGrid[:])
	clear(l.gasGr[:])
	clear(l.webGrid[:])
	others := 0 // bodies other than dark matter and holes
	l.look.Holes = l.look.Holes[:0]
	l.look.World = l.look.World[:0]
	mean := math.Max(info.MeanStarMass, 1e-12)
	l.look.Pop = e.population(info.MeanStarMass)
	starSprite := func(rel, depth float64) (alpha, size float64) {
		return math.Min(1, 0.55+0.2*rel), math.Max(3.5*ps, math.Min(28*ps, 0.05*math.Cbrt(rel)*focal/depth))
	}
	l.look.DustKappa = 0
	if fx.Dust {
		l.look.DustKappa = l.buildDust(bodies, n, &vw, w, h, focal, ps, e.Scene.DustOpacity())
		if cap(l.attn) < 4*n {
			l.attn = make([]float32, 4*n)
		}
		l.attn = l.attn[:4*n]
	}
	hazeSize := func(depth float64) float64 { return math.Max(6*ps, math.Min(48*ps, 0.12*focal/depth)) }
	for i := range n {
		b := bodies[i*FloatsPerBody : i*FloatsPerBody+FloatsPerBody : i*FloatsPerBody+FloatsPerBody]
		kind := sim.Kind(b[7] + 0.5)
		if kind == sim.BlackHole {
			rs := sim.SchwarzschildRadius(float64(b[3]), info.C)
			l.look.World = append(l.look.World, [4]float32{b[0], b[1], b[2], float32(rs)})
		}
		sx, sy, depth, ok := vw.Project(float64(b[0]), float64(b[1]), float64(b[2]))
		if !ok {
			continue
		}
		if kind == sim.BlackHole {
			hl := hole(float64(b[3]), info.C, sx, sy, focal/depth, ps)
			m := float64(b[3])
			j := vec.Vec3{X: float64(b[8]), Y: float64(b[9]), Z: float64(b[10])}
			a := sim.SpinParameter(j, m, info.C)
			n := vec.Vec3{Z: 1} // spin axis, which is also the disk normal: the spin grows from what falls in
			if j.Len2() > 0 {
				n = j.Scale(1 / j.Len())
			}
			// The axis on screen: x, y projected (length sin of the inclination), z toward the camera.
			e := 0.01 * depth
			if ax, ay, d2, ok := vw.Project(float64(b[0])+e*n.X, float64(b[1])+e*n.Y, float64(b[2])+e*n.Z); ok {
				k := e * focal / depth
				hl.AxisX, hl.AxisY, hl.AxisZ = float32((ax-sx)/k), float32((ay-sy)/k), float32((depth-d2)/e)
			}
			hl.Spin = float32(a)
			if info.C > 0 {
				hl.R = float32(math.Max(minHolePixels*ps, sim.KerrHorizon(m, a, info.C)*focal/depth))
				hl.Rings = float64(hl.R) > minHolePixels*ps
				hl.Disk = float32(diskBrightness(float64(b[11]), m))
			}
			l.look.Holes = append(l.look.Holes, hl)
			continue
		}
		if fx.Dust {
			tr, tg, tb := transmission(l.dust.front(sx, sy, depth))
			l.attn[4*i], l.attn[4*i+1], l.attn[4*i+2], l.attn[4*i+3] = tr, tg, tb, 1
		}
		cx, cy := int(sx/cw), int(sy/ch)
		if cx < 0 || cy < 0 || cx >= gridW || cy >= gridH {
			continue
		}
		cell := cy*gridW + cx
		if kind != sim.DarkMatter {
			others++
		}
		switch kind {
		case sim.Star:
			alpha, size := starSprite(float64(b[3])/mean, depth)
			if fx.Dust {
				alpha *= float64(l.attn[4*i+1]) // the exposure sees the attenuated light
			}
			ya, ys := youngBoost(float64(b[8]), info.YoungAge)
			alpha *= ya
			size *= ys
			l.light[cell] += float32(alpha * spriteMean * size * size)
		case sim.Gas:
			if w := l.look.Pop.starFormingLight(float64(b[11]), float64(b[9])); w > 0 {
				alpha, size := starSprite(float64(b[3])/mean, depth)
				if fx.Dust {
					alpha *= float64(l.attn[4*i+2]) // young stars are blue
				}
				l.light[cell] += float32(w * alpha * spriteMean * size * size)
			}
			if fx.Gas {
				// Column density per pixel at the center: a kernel of diameter 2h, or the cell if
				// the body is smaller than it (T-069).
				s := math.Max(2*float64(b[10])*focal/depth, 4*ps)
				ref := gasMapRef * ps
				l.gasGr[cell] += float32(float64(b[3]) / mean * ref * ref / math.Max(cw*ch, kernelMean*s*s))
			}
			if ion := float64(b[9]); ion > minEmissionIon {
				size := math.Max(3*ps, 3*float64(b[10])*focal/depth)
				brightness := ion * ion * float64(b[3]) * float64(b[11]) / (size * size)
				if fx.Dust {
					brightness *= float64(l.attn[4*i]) // Hα is red
				}
				l.emLight[cell] += float32(brightness * kernelMean * size * size)
			}
		case sim.DarkMatter:
			l.webGrid[cell] += float32(spriteMean * webPixels * webPixels * ps * ps / (cw * ch))
			if fx.Dark {
				s := hazeSize(depth)
				l.darkGrid[cell] += float32(spriteMean * s * s / (cw * ch))
			}
		}
	}
	area := float32(cw * ch)
	l.adapting = false
	if p, ok := l.percentile(l.light[:], area, exposurePercentile); ok {
		target := math.Max(1e-4, math.Min(1, exposureTarget/p))
		if l.look.Exposure == 0 {
			l.look.Exposure = target
		} else {
			l.look.Exposure += smoothing * (target - l.look.Exposure)
		}
		l.adapting = math.Abs(target-l.look.Exposure) > 0.03*l.look.Exposure
	}
	if p, ok := l.percentile(l.emLight[:], area, emissionPercentile); ok {
		target := emissionTarget / p
		if l.look.EmNorm == 0 {
			l.look.EmNorm = target
		} else {
			l.look.EmNorm += smoothing * (target - l.look.EmNorm)
		}
	}
	l.look.DarkAlpha, l.look.GasAlpha, l.look.WebAlpha = 0, 0, 0
	if p, ok := l.percentile(l.webGrid[:], 1, hazePercentile); ok && others == 0 && info.Stars == 0 {
		l.look.WebAlpha = float32(webTarget / p) // dark matter is all there is: draw it, not a haze
	} else if fx.Dark {
		l.look.DarkAlpha = l.hazeAlpha(l.darkGrid[:])
	}
	if fx.Gas {
		l.look.GasAlpha = l.hazeAlpha(l.gasGr[:])
	}
	if fx.Dust {
		copy(attn, l.attn)
	}
	return l.look
}

// hole projects a black hole: with a finite c the horizon r_s, without a symbolic size.
func hole(m, c, sx, sy, scale, ps float64) Hole {
	r := sim.SchwarzschildRadius(m, c)
	if r == 0 {
		r = 0.06 * math.Sqrt(m)
	}
	px := r * scale
	drawn := math.Max(minHolePixels*ps, px)
	return Hole{X: float32(sx), Y: float32(sy), R: float32(drawn), PhysR: float32(px), Rings: drawn > minHolePixels*ps}
}

// percentile returns the q-quantile of the lit cells per area.
func (l *looker) percentile(cells []float32, area float32, q float64) (float64, bool) {
	lit := l.sortBuf[:0]
	for _, v := range cells {
		if v > 0 {
			lit = append(lit, v/area)
		}
	}
	l.sortBuf = lit
	if len(lit) == 0 {
		return 0, false
	}
	slices.Sort(lit)
	return float64(lit[int(q*float64(len(lit)-1))]), true
}

// hazeAlpha makes the densest hazePercentile of the covered cells reach intensity hazeTarget.
func (l *looker) hazeAlpha(grid []float32) float32 {
	p, ok := l.percentile(grid, 1, hazePercentile)
	if !ok {
		return 0
	}
	return float32(math.Min(1, hazeTarget/p))
}
