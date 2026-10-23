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

	exposure       float64 // current auto exposure, 0 before the first frame
	dustNorm       float64 // current dust normalization, 0 before the first frame
	dustVerts      []ebiten.Vertex
	backVerts      []ebiten.Vertex
	dustMass       [depthGridW * depthGridH]float64
	dustDepth      [depthGridW * depthGridH]float64
	depthW, depthH float64
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
// With doppler set, star colors and brightness follow the relativistic frequency shift.
//
// With back != nil, stars behind the dust (deeper than the mass-weighted dust depth of their screen
// cell, see DrawDust) go to back instead, so only they are attenuated.
func (sr *StarRenderer) DrawStars(dst, back *ebiten.Image, cam *Camera, sn *sim.Snapshot, fx Effects) {
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
	// Dark matter is drawn much fainter than stars, scaled so that the total haze stays similar
	// for any particle count.
	darkAlpha := float32(math.Max(0.02, math.Min(0.25, 0.25*math.Sqrt(5000/math.Max(1, float64(dark))))))
	gasAlpha := float32(math.Max(0.03, math.Min(0.3, 0.3*math.Sqrt(3000/math.Max(1, float64(gasCount))))))
	// Every body owns 4 vertex slots, so the quads can be built in parallel without coordination;
	// holes and culled stars get an empty quad.
	view := cam.View()
	sr.verts = resizeVerts(sr.verts, 4*n)
	split := back != nil
	if split {
		sr.backVerts = resizeVerts(sr.backVerts, 4*n)
	}
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
			if split {
				clear(sr.backVerts[4*i : 4*i+4])
			}
			if sn.Kind[i] == sim.DarkMatter && fx.Dark {
				sr.darkQuad(q, &view, float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i]), focal, darkAlpha, darkColor, cam)
				continue
			}
			if sn.Kind[i] == sim.Gas && fx.Gas {
				sr.darkQuad(q, &view, float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i]), focal, gasAlpha, gasColor, cam)
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
			// Light this star adds at exposure 1, booked to the screen cell of its center.
			if cx, cy := int(sx/sr.lightW), int(sy/sr.lightH); cx >= 0 && cy >= 0 && cx < exposureGridW && cy < exposureGridH {
				light[cy*exposureGridW+cx] += alpha * float32(spriteMean*size*size)
			}
			if split && sr.behindDust(sx, sy, depth) {
				clear(q)
				q = sr.backVerts[4*i : 4*i+4 : 4*i+4]
			}
			setQuad(q, float32(sx), float32(sy), float32(size/2), cr, cg, cb, min(1, alpha*float32(exposure)))
		}
	})
	sr.flush(dst)
	if split {
		sr.verts, sr.backVerts = sr.backVerts, sr.verts
		sr.flush(back)
		sr.verts, sr.backVerts = sr.backVerts, sr.verts
	}
	sr.adaptExposure()
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

// Tints of visualized dark matter and gas, and of young stars.
var (
	darkColor  = [3]float32{0.55, 0.35, 1.0}
	gasColor   = [3]float32{0.45, 0.75, 1.0}
	youngColor = [3]float32{0.6, 0.78, 1.0}
)

// darkQuad writes the faint quad of one dark matter or gas particle, or an empty quad if it is
// off screen.
func (sr *StarRenderer) darkQuad(q []ebiten.Vertex, view *View, x, y, z, focal float64, alpha float32, tint [3]float32, cam *Camera) {
	sx, sy, depth, ok := view.Project(x, y, z)
	size := math.Max(3, math.Min(16, 0.04*focal/depth))
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

// Dust (T-035): gas particles are drawn as column density into a separate image. Their footprint is
// the core of their SPH kernel (diameter 2h, about its full width at half maximum), their weight
// mass / area, so compact dense gas absorbs most. The
// normalization adapts like the exposure: the densest dustPercentile of the covered cells reach
// dustTarget (the image is 8-bit and must not saturate everywhere).
const (
	dustPercentile = 0.9
	dustTarget     = 0.45
)

// depthGridW, depthGridH: resolution of the dust depth map used to sort stars in front of or
// behind the dust.
const depthGridW, depthGridH = 192, 120

// behindDust reports whether a star at screen (sx, sy) and depth is behind the dust there.
func (sr *StarRenderer) behindDust(sx, sy, depth float64) bool {
	cx, cy := int(sx/sr.depthW), int(sy/sr.depthH)
	if cx < 0 || cy < 0 || cx >= depthGridW || cy >= depthGridH {
		return false
	}
	k := cy*depthGridW + cx
	return sr.dustMass[k] > 0 && depth > sr.dustDepth[k]/sr.dustMass[k]
}

// DrawDust renders the dust column density of sn into dst and records the dust depth map. Call it
// before DrawStars.
func (sr *StarRenderer) DrawDust(dst *ebiten.Image, cam *Camera, sn *sim.Snapshot) {
	n := sn.Len()
	view, focal := cam.View(), cam.focal()
	norm := sr.dustNorm
	if norm == 0 {
		norm = 1
	}
	sr.dustVerts = resizeVerts(sr.dustVerts, 0)
	clear(sr.light[:]) // reused as the column density grid
	sr.lightW, sr.lightH = cam.W/exposureGridW, cam.H/exposureGridH
	clear(sr.dustMass[:])
	clear(sr.dustDepth[:])
	sr.depthW, sr.depthH = cam.W/depthGridW, cam.H/depthGridH
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
		column := float64(sn.Mass[i]) / (size * size)
		if cx, cy := int(sx/sr.depthW), int(sy/sr.depthH); cx >= 0 && cy >= 0 && cx < depthGridW && cy < depthGridH {
			sr.dustMass[cy*depthGridW+cx] += float64(sn.Mass[i])
			sr.dustDepth[cy*depthGridW+cx] += float64(sn.Mass[i]) * depth
		}
		if cx, cy := int(sx/sr.lightW), int(sy/sr.lightH); cx >= 0 && cy >= 0 && cx < exposureGridW && cy < exposureGridH {
			sr.light[cy*exposureGridW+cx] += float32(column * spriteMean * size * size)
		}
		a := float32(math.Min(1, norm*column))
		sr.dustVerts = append(sr.dustVerts, make([]ebiten.Vertex, 4)...)
		setQuad(sr.dustVerts[len(sr.dustVerts)-4:], float32(sx), float32(sy), float32(size/2), 1, 1, 1, a)
	}
	verts := sr.verts
	sr.verts = sr.dustVerts
	sr.flush(dst)
	sr.dustVerts, sr.verts = sr.verts, verts
	// Adapt: column per screen cell, densest percentile → dustTarget.
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
	p := float64(lit[int(dustPercentile*float64(len(lit)-1))])
	target := dustTarget / p
	if sr.dustNorm == 0 {
		sr.dustNorm = target
	} else {
		sr.dustNorm += exposureSmoothing * (target - sr.dustNorm)
	}
}
