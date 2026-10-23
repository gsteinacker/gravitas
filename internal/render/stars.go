package render

import (
	"image"
	"image/color"
	"math"

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

// starColor maps stellar mass (in units of the mean mass) to a rough black-body tint.
func starColor(rel float64) (r, g, b float32) {
	t := float32(math.Max(0, math.Min(1, 0.5+0.25*math.Log2(rel))))
	return 1 - 0.35*t, 0.75 + 0.15*t, 0.45 + 0.55*t
}

// Hole is a black hole as projected on screen.
type Hole struct {
	X, Y float32 // screen position
	R    float32 // horizon radius in pixels (at least minHolePixels)
}

// Holes returns the black holes collected by the last DrawStars call.
func (sr *StarRenderer) Holes() []Hole { return sr.bhs }

// DrawStars renders all stars of sn as seen by cam into dst and collects the black holes.
// With doppler set, star colors and brightness follow the relativistic frequency shift.
func (sr *StarRenderer) DrawStars(dst *ebiten.Image, cam *Camera, sn *sim.Snapshot, doppler bool) {
	n := sn.Len()
	var starMass float64
	stars := 0
	sr.world = sr.world[:0]
	for i := range n {
		if sn.Kind[i] == sim.Star {
			starMass += float64(sn.Mass[i])
			stars++
		} else {
			sr.world = append(sr.world, worldHole{
				x: float64(sn.X[i]), y: float64(sn.Y[i]), z: float64(sn.Z[i]),
				rs: sim.SchwarzschildRadius(float64(sn.Mass[i]), sn.C), m: float64(sn.Mass[i]),
			})
		}
	}
	meanMass := starMass / math.Max(1, float64(stars))
	// Keep the total brightness roughly constant: dense scenes would otherwise saturate to white.
	exposure := math.Max(0.12, math.Min(1, math.Sqrt(10000/math.Max(1, float64(stars)))))
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
		sr.bhs = append(sr.bhs, Hole{float32(sx), float32(sy), float32(math.Max(minHolePixels, r*focal/depth))})
	}
	// Every body owns 4 vertex slots, so the quads can be built in parallel without coordination;
	// holes and culled stars get an empty quad.
	view := cam.View()
	sr.verts = resizeVerts(sr.verts, 4*n)
	sim.ParallelFor(n, 4096, 0, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			q := sr.verts[4*i : 4*i+4 : 4*i+4]
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
			alpha := float32(exposure * math.Min(1, 0.55+0.2*rel))
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
				alpha = min(1, alpha*beam)
			}
			setQuad(q, float32(sx), float32(sy), float32(size/2), cr, cg, cb, alpha)
		}
	})
	sr.flush(dst)
}

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
