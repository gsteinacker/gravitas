package render

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"galaxy/internal/sim"
)

const spriteSize = 32

// StarRenderer draws a snapshot as additive glowing sprites, batched into few draw calls.
type StarRenderer struct {
	sprite  *ebiten.Image
	verts   []ebiten.Vertex
	indices []uint32
	bhs     []blackHole
}

type blackHole struct{ x, y, r float32 }

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

// Draw renders all bodies of sn as seen by cam.
func (sr *StarRenderer) Draw(dst *ebiten.Image, cam *Camera, sn *sim.Snapshot) {
	n := sn.Len()
	var starMass float64
	stars := 0
	for i := range n {
		if sn.Kind[i] == sim.Star {
			starMass += float64(sn.Mass[i])
			stars++
		}
	}
	meanMass := starMass / math.Max(1, float64(stars))

	sr.verts, sr.bhs = sr.verts[:0], sr.bhs[:0]
	focal := cam.focal()
	for i := range n {
		sx, sy, depth, ok := cam.Project(float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i]))
		if !ok {
			continue
		}
		if sn.Kind[i] == sim.BlackHole {
			r := float32(math.Max(4, 0.06*math.Sqrt(float64(sn.Mass[i]))*focal/depth))
			sr.bhs = append(sr.bhs, blackHole{float32(sx), float32(sy), r})
			continue
		}
		rel := float64(sn.Mass[i]) / meanMass
		size := 0.05 * math.Cbrt(rel) * focal / depth
		size = math.Max(3.5, math.Min(28, size))
		if sx < -size || sy < -size || sx > cam.W+size || sy > cam.H+size {
			continue
		}
		cr, cg, cb := starColor(rel)
		alpha := float32(math.Min(1, 0.55+0.2*rel))
		sr.appendQuad(float32(sx), float32(sy), float32(size/2), cr, cg, cb, alpha)
	}
	sr.flush(dst)

	for _, bh := range sr.bhs {
		vector.FillCircle(dst, bh.x, bh.y, bh.r*2.2, color.RGBA{60, 25, 5, 0}, true)
		vector.StrokeCircle(dst, bh.x, bh.y, bh.r*1.4, bh.r*0.5, color.RGBA{255, 150, 50, 200}, true)
		vector.FillCircle(dst, bh.x, bh.y, bh.r, color.Black, true)
	}
}

func (sr *StarRenderer) appendQuad(x, y, h, r, g, b, a float32) {
	const s = spriteSize
	sr.verts = append(sr.verts,
		ebiten.Vertex{DstX: x - h, DstY: y - h, SrcX: 0, SrcY: 0, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
		ebiten.Vertex{DstX: x + h, DstY: y - h, SrcX: s, SrcY: 0, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
		ebiten.Vertex{DstX: x - h, DstY: y + h, SrcX: 0, SrcY: s, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
		ebiten.Vertex{DstX: x + h, DstY: y + h, SrcX: s, SrcY: s, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
	)
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
	op := &ebiten.DrawTrianglesOptions{Blend: ebiten.BlendLighter, Filter: ebiten.FilterLinear}
	for start := 0; start < len(sr.verts); start += 4 * quadsPerBatch {
		end := min(len(sr.verts), start+4*quadsPerBatch)
		dst.DrawTriangles32(sr.verts[start:end], sr.indices[:6*(end-start)/4], sr.sprite, op)
	}
}
