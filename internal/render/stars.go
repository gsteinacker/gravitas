package render

import (
	"image"
	"image/color"
	"math"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"galaxy/internal/sim"
)

// spriteSize is small on purpose: stars are drawn at 3.5–28 px without mipmaps.
const spriteSize = 16

// minHolePixels keeps tiny horizons visible on screen.
const minHolePixels = 3

// StarRenderer draws a snapshot as additive glowing sprites, batched into few draw calls.
type StarRenderer struct {
	sprite  *ebiten.Image
	verts   []ebiten.Vertex
	indices []uint32
	bhs     []Hole
	world   []worldHole

	exposure  float64 // current auto exposure, 0 before the first frame
	emNorm    float64 // current emission normalization
	emLight   [exposureGridW * exposureGridH]float32
	emLightMu sync.Mutex
	// DustOpacity is the V-band optical depth per unit column density (simulation units) of
	// neutral gas, from the scene's physical units; 0 normalizes the dust automatically.
	DustOpacity    float64
	dustNorm       float64 // automatic dust opacity, 0 before the first frame
	dustVerts      []ebiten.Vertex
	dust           dustMap
	dustOn         bool // the dust map is valid for this frame
	adapting       bool
	light          [exposureGridW * exposureGridH]float32
	lightMu        sync.Mutex
	lightW, lightH float64
	sortBuf        []float32
}

type worldHole struct{ x, y, z, rs, m float64 }

func NewStarRenderer() *StarRenderer {
	return &StarRenderer{sprite: glowSprite()}
}

// glowSprite is a white radial falloff: a bright core with a soft halo.
func glowSprite() *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, spriteSize, spriteSize))
	c := float64(spriteSize-1) / 2
	for y := range spriteSize {
		for x := range spriteSize {
			d := math.Hypot(float64(x)-c, float64(y)-c) / c
			a := math.Exp(-d*d*12) + 0.25*math.Exp(-d*d*3)
			if d >= 1 {
				a = 0
			}
			v := uint8(math.Min(1, a) * 255)
			img.SetRGBA(x, y, color.RGBA{v, v, v, v})
		}
	}
	return ebiten.NewImageFromImage(img)
}

// spriteMean is the mean intensity of the glow sprite over its quad.
var spriteMean = func() float64 {
	c := float64(spriteSize-1) / 2
	var sum float64
	for y := range spriteSize {
		for x := range spriteSize {
			d := math.Hypot(float64(x)-c, float64(y)-c) / c
			if d < 1 {
				sum += math.Min(1, math.Exp(-d*d*12)+0.25*math.Exp(-d*d*3))
			}
		}
	}
	return sum / (spriteSize * spriteSize)
}()

// starColor maps stellar mass (in units of the mean mass) to a rough black-body tint.
func starColor(rel float64) (r, g, b float32) {
	t := float32(math.Max(0, math.Min(1, 0.5+0.25*math.Log2(rel))))
	return 1 - 0.35*t, 0.75 + 0.15*t, 0.45 + 0.55*t
}

// Hole is a black hole as projected on screen.
type Hole struct {
	X, Y  float32 // screen position
	R     float32 // drawn horizon radius in pixels (at least minHolePixels)
	PhysR float32 // actual horizon radius in pixels (for lensing; may be far below a pixel)
}

// Holes returns the black holes collected by the last DrawStars call.
func (sr *StarRenderer) Holes() []Hole { return sr.bhs }

// DrawStars renders all stars of sn as seen by cam into dst and collects the black holes.
// With doppler set, star colors and brightness follow the relativistic frequency shift. With dust
// set (after DrawDust), stars and emission are attenuated by the dust in front of them (dust.go).
func (sr *StarRenderer) DrawStars(dst *ebiten.Image, cam *Camera, sn *sim.Snapshot, fx Effects) {
	doppler := fx.Doppler
	n := sn.Len()
	var starMass float64
	stars, dark, gasCount := 0, 0, 0
	sr.world = sr.world[:0]
	for i := range n {
		switch sn.Kind[i] {
		case sim.Star:
			starMass += float64(sn.Mass[i])
			stars++
		case sim.DarkMatter:
			dark++
		case sim.Gas:
			gasCount++
		case sim.BlackHole:
			sr.world = append(sr.world, worldHole{
				x: float64(sn.X[i]), y: float64(sn.Y[i]), z: float64(sn.Z[i]),
				rs: sim.SchwarzschildRadius(float64(sn.Mass[i]), sn.C), m: float64(sn.Mass[i]),
			})
		}
	}
	meanMass := starMass / math.Max(1, float64(stars))
	exposure := sr.exposure
	if exposure == 0 { // first frame: rough guess from the star count, refined below
		exposure = math.Max(0.02, math.Min(1, math.Sqrt(10000/math.Max(1, float64(stars)))))
	}
	sr.lightW, sr.lightH = cam.W/exposureGridW, cam.H/exposureGridH
	clear(sr.light[:])
	clear(sr.emLight[:])
	camPos := cam.Position()
	doppler = doppler && sn.C > 0

	sr.bhs = sr.bhs[:0]
	focal := cam.focal()
	for _, h := range sr.world {
		sx, sy, depth, ok := cam.Project(h.x, h.y, h.z)
		if !ok {
			continue
		}
		// With a finite c the black disk is the event horizon r_s; without, a symbolic size.
		r := h.rs
		if r == 0 {
			r = 0.06 * math.Sqrt(h.m)
		}
		px := r * focal / depth
		sr.bhs = append(sr.bhs, Hole{X: float32(sx), Y: float32(sy), R: float32(math.Max(minHolePixels, px)), PhysR: float32(px)})
	}
	// Dark matter and the gas map are a haze whose opacity follows from their projected density
	// in this frame (hazeAlpha).
	view := cam.View()
	var darkAlpha, gasAlpha float32
	if fx.Dark {
		darkAlpha = sr.hazeAlpha(sn, sim.DarkMatter, &view, focal, cam)
	}
	if fx.Gas {
		gasAlpha = sr.hazeAlpha(sn, sim.Gas, &view, focal, cam)
	}
	// Every body owns 4 vertex slots, so the quads can be built in parallel without coordination;
	// holes and culled stars get an empty quad.
	sr.verts = resizeVerts(sr.verts, 4*n)
	dusty := fx.Dust && sr.dustOn
	sim.ParallelFor(n, 4096, 0, func(lo, hi int) {
		light := lightPool.Get().(*[exposureGridW * exposureGridH]float32)
		clear(light[:])
		defer func() {
			sr.lightMu.Lock()
			for k, v := range light {
				sr.light[k] += v
			}
			sr.lightMu.Unlock()
			lightPool.Put(light)
		}()
		for i := lo; i < hi; i++ {
			q := sr.verts[4*i : 4*i+4 : 4*i+4]
			if sn.Kind[i] == sim.DarkMatter && fx.Dark {
				sr.darkQuad(q, &view, float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i]), focal, darkAlpha, darkColor, cam)
				continue
			}
			if sn.Kind[i] == sim.Gas && fx.Gas {
				// False-colour gas map: neutral gas blue, partly ionized green, ionized red.
				ion := sn.GasIon[i]
				tint := [3]float32{0.25 + 0.75*ion, 0.45 + 0.5*(1-2*abs32(ion-0.5)), 1 - 0.8*ion}
				sr.darkQuad(q, &view, float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i]), focal, gasAlpha, tint, cam)
				continue
			}
			if sn.Kind[i] == sim.Gas && sn.GasIon[i] > minEmissionIon {
				sr.emissionQuad(q, dusty, i, &view, sn, focal, cam)
				continue
			}
			if sn.Kind[i] != sim.Star {
				clear(q)
				continue
			}
			x, y, z := float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i])
			sx, sy, depth, ok := view.Project(x, y, z)
			if !ok {
				clear(q)
				continue
			}
			rel := float64(sn.Mass[i]) / meanMass
			size := 0.05 * math.Cbrt(rel) * focal / depth
			size = math.Max(3.5, math.Min(28, size))
			if sx < -size || sy < -size || sx > cam.W+size || sy > cam.H+size {
				clear(q)
				continue
			}
			cr, cg, cb := starColor(rel)
			alpha := float32(math.Min(1, 0.55+0.2*rel)) // before exposure
			if age := float64(sn.Age[i]); age >= 0 && age < sn.YoungAge {
				// Young stars from gas: hot, blue and bright, fading to normal over YoungAge.
				f := float32(1 - age/sn.YoungAge)
				cr, cg, cb = cr+(youngColor[0]-cr)*f, cg+(youngColor[1]-cg)*f, cb+(youngColor[2]-cb)*f
				alpha *= 1 + 1.5*f
				size *= 1 + 0.5*float64(f)
			}
			if doppler {
				grav := 1.0
				for _, h := range sr.world {
					dx, dy, dz := x-h.x, y-h.y, z-h.z
					grav *= gravShift(math.Sqrt(dx*dx+dy*dy+dz*dz), h.rs)
				}
				nx, ny, nz := camPos.X-x, camPos.Y-y, camPos.Z-z
				l := math.Sqrt(nx*nx + ny*ny + nz*nz)
				f := frequencyShift(float64(sn.VX[i]), float64(sn.VY[i]), float64(sn.VZ[i]), nx/l, ny/l, nz/l, sn.C, grav)
				var beam float32
				cr, cg, cb, beam = shiftColor(cr, cg, cb, f)
				alpha *= beam
			}
			if dusty {
				tr, tg, tb := transmission(sr.dust.front(sx, sy, depth))
				cr, cg, cb = cr*tr, cg*tg, cb*tb
				alpha *= tg // the exposure sees the attenuated light
			}
			// Light this star adds at exposure 1, booked to the screen cell of its center.
			if cx, cy := int(sx/sr.lightW), int(sy/sr.lightH); cx >= 0 && cy >= 0 && cx < exposureGridW && cy < exposureGridH {
				light[cy*exposureGridW+cx] += alpha * float32(spriteMean*size*size)
			}
			setQuad(q, float32(sx), float32(sy), float32(size/2), cr, cg, cb, min(1, alpha*float32(exposure)))
		}
	})
	sr.flush(dst)
	sr.adaptExposure()
	sr.adaptEmission()
}

// Auto exposure (T-033): additive stars saturate an 8-bit image, and saturated pixels hide the
// density contrast of spiral arms. The exposure is chosen so that the brightest exposurePercentile
// of the lit screen cells just reach white. It is measured while drawing and applied from the next
// frame on, smoothed so it does not flicker.
const (
	exposureGridW, exposureGridH = 96, 60
	exposurePercentile           = 0.9
	exposureTarget               = 0.9 // brightness of that percentile
	exposureSmoothing            = 0.35
)

var lightPool = sync.Pool{New: func() any { return new([exposureGridW * exposureGridH]float32) }}

func (sr *StarRenderer) adaptExposure() {
	lit := sr.sortBuf[:0]
	area := float32(sr.lightW * sr.lightH)
	for _, v := range sr.light {
		if v > 0 {
			lit = append(lit, v/area)
		}
	}
	sr.sortBuf = lit
	if len(lit) == 0 {
		return
	}
	slices.Sort(lit)
	p := float64(lit[int(exposurePercentile*float64(len(lit)-1))])
	target := math.Max(1e-4, math.Min(1, exposureTarget/p))
	if sr.exposure == 0 {
		sr.exposure = target
	} else {
		sr.exposure += exposureSmoothing * (target - sr.exposure)
	}
	sr.adapting = math.Abs(target-sr.exposure) > 0.03*sr.exposure
}

// Adapting reports whether the exposure is still converging, so the star image needs redrawing.
func (sr *StarRenderer) Adapting() bool { return sr.adapting }

// DrawHoles draws the black holes collected by the last DrawStars call.
func (sr *StarRenderer) DrawHoles(dst *ebiten.Image) {
	for _, bh := range sr.bhs {
		// Additive glow out to the shadow edge (~2.6 r_s), the ISCO at 3 r_s, the photon sphere at
		// 1.5 r_s, and the horizon. The glow uses a premultiplied color with zero alpha, so it adds light.
		ring := float32(math.Max(1.5, float64(bh.R)*0.25))
		vector.FillCircle(dst, bh.X, bh.Y, bh.R*2.6, color.RGBA{18, 7, 2, 0}, true)
		if bh.R > minHolePixels {
			// Innermost stable circular orbit of the Paczyński-Wiita potential: stars inside plunge.
			vector.StrokeCircle(dst, bh.X, bh.Y, bh.R*3, 1, color.RGBA{90, 60, 40, 120}, true)
		}
		vector.StrokeCircle(dst, bh.X, bh.Y, bh.R*1.5, ring, color.RGBA{255, 150, 50, 220}, true)
		vector.FillCircle(dst, bh.X, bh.Y, bh.R, color.Black, true)
	}
}

// Emission of photoionized gas (T-037): Hα, brightness ∝ x² · m · ρ for ionized fraction x
// (recombinations ∝ n_e n_p) over the particle's footprint, normalized like the star exposure.
// The normalization puts the median of the emitting area at emissionTarget, so that the extended
// H II region is visible and dense ionization fronts are brighter by their emission measure;
// per sprite the brightness saturates softly instead of clipping.
const (
	minEmissionIon     = 0.2
	emissionTarget     = 0.25
	emissionPercentile = 0.5
)

var emissionColor = [3]float32{1.0, 0.16, 0.22} // Hα 656 nm (with a little Hβ)

func (sr *StarRenderer) emissionQuad(q []ebiten.Vertex, dusty bool, i int, view *View, sn *sim.Snapshot, focal float64, cam *Camera) {
	sx, sy, depth, ok := view.Project(float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i]))
	size := math.Max(3, math.Min(120, 3*float64(sn.GasH[i])*focal/depth))
	if !ok || sx < -size || sy < -size || sx > cam.W+size || sy > cam.H+size {
		clear(q)
		return
	}
	ion := float64(sn.GasIon[i])
	brightness := ion * ion * float64(sn.Mass[i]) * float64(sn.GasRho[i]) / (size * size) // n_e n_p ∝ (x ρ)²
	if dusty {
		brightness *= math.Exp(-sr.dust.front(sx, sy, depth) * dustK[0]) // Hα is red
	}
	if cx, cy := int(sx/sr.lightW), int(sy/sr.lightH); cx >= 0 && cy >= 0 && cx < exposureGridW && cy < exposureGridH {
		sr.emLightMu.Lock()
		sr.emLight[cy*exposureGridW+cx] += float32(brightness * spriteMean * size * size)
		sr.emLightMu.Unlock()
	}
	norm := sr.emNorm
	if norm == 0 {
		norm = 1
	}
	setQuad(q, float32(sx), float32(sy), float32(size/2), emissionColor[0], emissionColor[1], emissionColor[2], float32(-math.Expm1(-norm*brightness)))
}

// adaptEmission adjusts the emission normalization (like adaptExposure).
func (sr *StarRenderer) adaptEmission() {
	lit := sr.sortBuf[:0]
	area := float32(sr.lightW * sr.lightH)
	for _, v := range sr.emLight {
		if v > 0 {
			lit = append(lit, v/area)
		}
	}
	sr.sortBuf = lit
	if len(lit) == 0 {
		return
	}
	slices.Sort(lit)
	p := float64(lit[int(emissionPercentile*float64(len(lit)-1))])
	target := emissionTarget / p
	if sr.emNorm == 0 {
		sr.emNorm = target
	} else {
		sr.emNorm += exposureSmoothing * (target - sr.emNorm)
	}
	if math.Abs(target-sr.emNorm) > 0.03*sr.emNorm {
		sr.adapting = true
	}
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// Tints of visualized dark matter and gas, and of young stars.
var (
	darkColor  = [3]float32{0.55, 0.35, 1.0}
	gasColor   = [3]float32{0.45, 0.75, 1.0}
	youngColor = [3]float32{0.6, 0.78, 1.0}
)

// darkQuad writes the faint quad of one dark matter or gas particle, or an empty quad if it is
// off screen.
// hazeSize is the sprite size of a haze body (dark matter, gas map) at the given depth.
func hazeSize(focal, depth float64) float64 { return math.Max(6, math.Min(48, 0.12*focal/depth)) }

// Haze opacity: the densest hazePercentile of the covered screen cells reach intensity hazeTarget.
const (
	hazePercentile = 0.9
	hazeTarget     = 0.5
)

// hazeAlpha returns the per-sprite opacity of the bodies of kind k for this frame.
func (sr *StarRenderer) hazeAlpha(sn *sim.Snapshot, k sim.Kind, view *View, focal float64, cam *Camera) float32 {
	var grid [exposureGridW * exposureGridH]float32
	cw, ch := cam.W/exposureGridW, cam.H/exposureGridH
	for i := range sn.Len() {
		if sn.Kind[i] != k {
			continue
		}
		sx, sy, depth, ok := view.Project(float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i]))
		if !ok {
			continue
		}
		if cx, cy := int(sx/cw), int(sy/ch); cx >= 0 && cy >= 0 && cx < exposureGridW && cy < exposureGridH {
			size := hazeSize(focal, depth)
			grid[cy*exposureGridW+cx] += float32(spriteMean * size * size / (cw * ch))
		}
	}
	lit := sr.sortBuf[:0]
	for _, v := range grid {
		if v > 0 {
			lit = append(lit, v)
		}
	}
	sr.sortBuf = lit
	if len(lit) == 0 {
		return 0
	}
	slices.Sort(lit)
	p := lit[int(hazePercentile*float64(len(lit)-1))]
	return min(1, hazeTarget/p)
}

func (sr *StarRenderer) darkQuad(q []ebiten.Vertex, view *View, x, y, z, focal float64, alpha float32, tint [3]float32, cam *Camera) {
	sx, sy, depth, ok := view.Project(x, y, z)
	size := hazeSize(focal, depth)
	if !ok || sx < -size || sy < -size || sx > cam.W+size || sy > cam.H+size {
		clear(q)
		return
	}
	setQuad(q, float32(sx), float32(sy), float32(size/2), tint[0], tint[1], tint[2], alpha)
}

// setQuad writes a sprite quad centered at (x, y) with half size h into the 4 vertices of q.
func setQuad(q []ebiten.Vertex, x, y, h, r, g, b, a float32) {
	const s = spriteSize
	q[0] = ebiten.Vertex{DstX: x - h, DstY: y - h, SrcX: 0, SrcY: 0, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
	q[1] = ebiten.Vertex{DstX: x + h, DstY: y - h, SrcX: s, SrcY: 0, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
	q[2] = ebiten.Vertex{DstX: x - h, DstY: y + h, SrcX: 0, SrcY: s, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
	q[3] = ebiten.Vertex{DstX: x + h, DstY: y + h, SrcX: s, SrcY: s, ColorR: r, ColorG: g, ColorB: b, ColorA: a}
}

func resizeVerts(v []ebiten.Vertex, n int) []ebiten.Vertex {
	if cap(v) < n {
		return make([]ebiten.Vertex, n)
	}
	return v[:n]
}

// flush draws the collected quads in batches that respect ebiten.MaxVertexCount.
func (sr *StarRenderer) flush(dst *ebiten.Image) {
	quadsPerBatch := ebiten.MaxVertexCount / 4
	if want := 6 * min(quadsPerBatch, len(sr.verts)/4); len(sr.indices) < want {
		sr.indices = sr.indices[:0]
		for q := range uint32(want / 6) {
			b := 4 * q
			sr.indices = append(sr.indices, b, b+1, b+2, b+1, b+3, b+2)
		}
	}
	// No mipmaps: Ebitengine would compute a mipmap level per triangle on the render thread,
	// which was half of the frame time at 1M stars (T-030). The small, smooth sprite keeps
	// downscaled sampling free of aliasing.
	op := &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendLighter, Filter: ebiten.FilterLinear, DisableMipmaps: true}
	for start := 0; start < len(sr.verts); start += 4 * quadsPerBatch {
		end := min(len(sr.verts), start+4*quadsPerBatch)
		dst.DrawTriangles32(sr.verts[start:end], sr.indices[:6*(end-start)/4], sr.sprite, op)
	}
}

// Dust (T-035, T-040): DrawDust fills the dust map (dust.go) used to attenuate stars and emission,
// and draws the gas as column density into dst, which attenuates the background behind it (the
// dust shader; τ = alpha · dustStrength). Footprints are the core of the SPH kernel (diameter 2h).
// Without physical units (DustOpacity 0) the opacity adapts: the densest dustPercentile of the
// covered screen reaches optical depth dustAutoTau.
const (
	dustPercentile = 0.9
	dustAutoTau    = 1.5
)

// DrawDust must run before DrawStars.
func (sr *StarRenderer) DrawDust(dst *ebiten.Image, cam *Camera, sn *sim.Snapshot) {
	n := sn.Len()
	view, focal := cam.View(), cam.focal()
	kappa := sr.DustOpacity
	if kappa == 0 {
		kappa = sr.dustNorm
		if kappa == 0 {
			kappa = 1
		}
	}
	sr.dustVerts = resizeVerts(sr.dustVerts, 0)
	sr.dust.reset(cam.W)
	sr.dustOn = true
	for i := range n {
		if sn.Kind[i] != sim.Gas {
			continue
		}
		sx, sy, depth, ok := view.Project(float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i]))
		if !ok {
			continue
		}
		size := math.Max(2, math.Min(80, 2*float64(sn.GasH[i])*focal/depth))
		if sx < -size || sy < -size || sx > cam.W+size || sy > cam.H+size {
			continue
		}
		// τ · footprint in pixels² = κ m (focal/depth)² for the neutral mass (dust survives there).
		tauArea := kappa * float64(sn.Mass[i]) * (1 - float64(sn.GasIon[i])) * (focal / depth) * (focal / depth)
		sr.dust.deposit(sx, sy, size, tauArea, depth)
		a := float32(math.Min(1, tauArea/(spriteMean*size*size)/dustStrength))
		sr.dustVerts = append(sr.dustVerts, make([]ebiten.Vertex, 4)...)
		setQuad(sr.dustVerts[len(sr.dustVerts)-4:], float32(sx), float32(sy), float32(size/2), 1, 1, 1, a)
	}
	verts := sr.verts
	sr.verts = sr.dustVerts
	sr.flush(dst)
	sr.dustVerts, sr.verts = sr.verts, verts
	if sr.DustOpacity > 0 {
		return
	}
	// Automatic opacity: sample the total optical depth on the exposure grid.
	lit := sr.sortBuf[:0]
	for cy := range exposureGridH {
		for cx := range exposureGridW {
			x, y := (float64(cx)+0.5)*cam.W/exposureGridW, (float64(cy)+0.5)*cam.H/exposureGridH
			if tau, _, _ := sr.dust.sample(x, y); tau > 0 {
				lit = append(lit, float32(tau))
			}
		}
	}
	sr.sortBuf = lit
	if len(lit) == 0 {
		return
	}
	slices.Sort(lit)
	p := float64(lit[int(dustPercentile*float64(len(lit)-1))])
	target := kappa * dustAutoTau / p
	if sr.dustNorm == 0 {
		sr.dustNorm = target
	} else {
		sr.dustNorm += exposureSmoothing * (target - sr.dustNorm)
	}
}
