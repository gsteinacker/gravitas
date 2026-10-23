package render

import (
	"fmt"
	"image"
	"image/color"
	"math/rand/v2"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"galaxy/internal/sim"
)

// Effects selects the optional render passes (M4).
type Effects struct {
	Glow    bool // bloom (T-016)
	Doppler bool // relativistic color shift (T-017)
	Lensing bool // gravitational lens around black holes (T-018)
	Trails  bool // fading star trails (T-019)
	Dark    bool // draw dark matter as a faint violet haze
	Gas     bool // draw gas as a faint glow
}

// ParseEffects reads a comma-separated list such as "glow,lens,doppler,trails".
func ParseEffects(list string) (Effects, error) {
	var fx Effects
	for _, name := range strings.Split(list, ",") {
		switch strings.TrimSpace(name) {
		case "":
		case "glow":
			fx.Glow = true
		case "doppler":
			fx.Doppler = true
		case "lens":
			fx.Lensing = true
		case "trails":
			fx.Trails = true
		case "dark":
			fx.Dark = true
		case "gas":
			fx.Gas = true
		default:
			return fx, fmt.Errorf("unknown effect %q (want glow, doppler, lens, trails, dark, gas)", name)
		}
	}
	return fx, nil
}

const (
	// lensFactor: the Einstein radius of a hole on screen, in horizon radii. Visual choice; the
	// real value depends on the distances of source and observer.
	lensFactor = 5
	// maxLensHoles must match the array size in lensShader.
	maxLensHoles = 16
	// trailFade is the per-frame brightness kept by star trails.
	trailFade = 0.93
	// bloomStrength scales the blurred image added on top.
	bloomStrength = 0.9
	// bloomScale is the downsampling factor of the bloom pass.
	bloomScale = 4
	// backgroundStars per megapixel of the static sky behind the scene.
	backgroundStars = 1500
)

// lensShader displaces every pixel according to the point-lens equation β = θ − θ_E² θ/|θ|²,
// summed over all holes. Pixels inside the Einstein radius show the mirrored secondary image.
const lensShader = `//kage:unit pixels
package main

var Holes [16]vec4
var Count float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	p := dstPos.xy - imageDstOrigin()
	q := p
	for i := 0; i < 16; i++ {
		if float(i) < Count {
			d := p - Holes[i].xy
			q -= d * (Holes[i].z / max(dot(d, d), 1))
		}
	}
	return imageSrc0At(q + imageSrc0Origin())
}
`

// blurShader is one direction of a separable 9-tap Gaussian blur.
const blurShader = `//kage:unit pixels
package main

var Dir vec2

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	w := [...]float{0.227027, 0.1945946, 0.1216216, 0.054054, 0.016216}
	c := imageSrc0At(srcPos) * w[0]
	for i := 1; i < len(w); i++ {
		o := Dir * float(i) * 1.5
		c += (imageSrc0At(srcPos+o) + imageSrc0At(srcPos-o)) * w[i]
	}
	return c
}
`

// Pipeline renders a snapshot with all post-processing passes:
// stars → (trails) → over background → (lens) → screen → (+ bloom) → holes.
type Pipeline struct {
	Stars *StarRenderer

	lens, blur *ebiten.Shader
	w, h       int

	stars, trailA, trailB, frame, lensed *ebiten.Image
	small, blurA, blurB, background      *ebiten.Image
	fadeFloor                            *ebiten.Image
	uniform                              []float32

	// Cache key of the star image: redrawing 1M stars is the most expensive part of a frame, and
	// the simulation publishes far fewer snapshots than the display shows frames (T-030).
	drawnVersion uint64
	drawnView    View
	drawnFx      Effects
	drawnValid   bool
}

func NewPipeline() (*Pipeline, error) {
	lens, err := ebiten.NewShader([]byte(lensShader))
	if err != nil {
		return nil, err
	}
	blur, err := ebiten.NewShader([]byte(blurShader))
	if err != nil {
		return nil, err
	}
	floor := ebiten.NewImage(1, 1)
	floor.Fill(color.RGBA{trailFloor, trailFloor, trailFloor, trailFloor})
	return &Pipeline{
		Stars: NewStarRenderer(), lens: lens, blur: blur, fadeFloor: floor,
		uniform: make([]float32, 4*maxLensHoles),
	}, nil
}

func (p *Pipeline) resize(w, h int) {
	if w == p.w && h == p.h {
		return
	}
	p.drawnValid = false
	for _, img := range []*ebiten.Image{p.stars, p.trailA, p.trailB, p.frame, p.lensed, p.small, p.blurA, p.blurB, p.background} {
		if img != nil {
			img.Deallocate()
		}
	}
	p.w, p.h = w, h
	p.stars, p.trailA, p.trailB = ebiten.NewImage(w, h), ebiten.NewImage(w, h), ebiten.NewImage(w, h)
	p.frame, p.lensed = ebiten.NewImage(w, h), ebiten.NewImage(w, h)
	sw, sh := max(1, w/bloomScale), max(1, h/bloomScale)
	p.small, p.blurA, p.blurB = ebiten.NewImage(sw, sh), ebiten.NewImage(sw, sh), ebiten.NewImage(sw, sh)
	p.background = starfield(w, h)
}

// blendMax keeps the brighter of source and destination per channel.
var blendMax = ebiten.Blend{
	BlendFactorSourceRGB: ebiten.BlendFactorOne, BlendFactorSourceAlpha: ebiten.BlendFactorOne,
	BlendFactorDestinationRGB: ebiten.BlendFactorOne, BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB: ebiten.BlendOperationMax, BlendOperationAlpha: ebiten.BlendOperationMax,
}

// blendSubtract computes destination − source.
var blendSubtract = ebiten.Blend{
	BlendFactorSourceRGB: ebiten.BlendFactorOne, BlendFactorSourceAlpha: ebiten.BlendFactorOne,
	BlendFactorDestinationRGB: ebiten.BlendFactorOne, BlendFactorDestinationAlpha: ebiten.BlendFactorOne,
	BlendOperationRGB: ebiten.BlendOperationReverseSubtract, BlendOperationAlpha: ebiten.BlendOperationReverseSubtract,
}

// trailFloor is subtracted from the trail every frame (8-bit units).
const trailFloor = 1

// skyColor is the opaque background of the scene.
var skyColor = color.RGBA{2, 2, 8, 255}

// starfield is a static, opaque sky with faint stars, so the lens has something to distort.
// It must be opaque: every later pass relies on the frame covering the whole screen.
func starfield(w, h int) *ebiten.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = skyColor.R, skyColor.G, skyColor.B, skyColor.A
	}
	rng := rand.New(rand.NewPCG(7, 11))
	for range w * h * backgroundStars / 1_000_000 {
		v := uint8(20 + rng.IntN(70))
		tint := uint8(rng.IntN(25))
		img.SetRGBA(rng.IntN(w), rng.IntN(h), color.RGBA{v - tint/2, v - tint/3, v, 255})
	}
	return ebiten.NewImageFromImage(img)
}

// Draw renders sn as seen by cam onto screen.
func (p *Pipeline) Draw(screen *ebiten.Image, cam *Camera, sn *sim.Snapshot, fx Effects) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	p.resize(w, h)

	starFx := Effects{Doppler: fx.Doppler, Dark: fx.Dark, Gas: fx.Gas} // the options that change the star image
	if view := cam.View(); !p.drawnValid || sn.Version != p.drawnVersion || view != p.drawnView || starFx != p.drawnFx || p.Stars.Adapting() {
		p.stars.Clear()
		p.Stars.DrawStars(p.stars, cam, sn, starFx)
		p.drawnValid, p.drawnVersion, p.drawnView, p.drawnFx = true, sn.Version, view, starFx
	}
	stars := p.stars
	if fx.Trails {
		// trailB = max(fade · trailA, stars), then swap: the trail keeps a decaying history but is
		// never brighter than the stars themselves (adding would saturate slow, dense regions).
		op := &ebiten.DrawImageOptions{Blend: ebiten.BlendCopy}
		op.ColorScale.Scale(trailFade, trailFade, trailFade, trailFade)
		p.trailB.DrawImage(p.trailA, op)
		// Multiplicative fading alone never reaches zero in 8 bits (round(7·0.93) = 7) and leaves a
		// grey haze, so a small constant is subtracted as well.
		op = &ebiten.DrawImageOptions{Blend: blendSubtract}
		op.GeoM.Scale(float64(w), float64(h))
		p.trailB.DrawImage(p.fadeFloor, op)
		p.trailB.DrawImage(p.stars, &ebiten.DrawImageOptions{Blend: blendMax})
		p.trailA, p.trailB = p.trailB, p.trailA
		stars = p.trailA
	} else {
		p.trailA.Clear()
	}

	p.frame.DrawImage(p.background, &ebiten.DrawImageOptions{Blend: ebiten.BlendCopy})
	p.frame.DrawImage(stars, &ebiten.DrawImageOptions{Blend: ebiten.BlendLighter})

	out := p.frame
	if holes := p.Stars.Holes(); fx.Lensing && len(holes) > 0 {
		count := min(len(holes), maxLensHoles)
		for i := range count {
			// The physical radius: the drawn one is clamped to stay visible, which would make tiny
			// holes (e.g. at the real c of the Local Group) lens like huge ones.
			e := lensFactor * holes[i].PhysR
			p.uniform[4*i], p.uniform[4*i+1], p.uniform[4*i+2] = holes[i].X, holes[i].Y, e*e
		}
		p.lensed.DrawRectShader(w, h, p.lens, &ebiten.DrawRectShaderOptions{
			Blend:    ebiten.BlendCopy,
			Images:   [4]*ebiten.Image{p.frame},
			Uniforms: map[string]any{"Holes": p.uniform, "Count": float32(count)},
		})
		out = p.lensed
	}
	screen.DrawImage(out, &ebiten.DrawImageOptions{Blend: ebiten.BlendCopy})

	if fx.Glow {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendCopy}
		op.GeoM.Scale(1.0/bloomScale, 1.0/bloomScale)
		p.small.DrawImage(out, op)
		sw, sh := p.small.Bounds().Dx(), p.small.Bounds().Dy()
		p.blurA.DrawRectShader(sw, sh, p.blur, &ebiten.DrawRectShaderOptions{
			Blend: ebiten.BlendCopy, Images: [4]*ebiten.Image{p.small}, Uniforms: map[string]any{"Dir": []float32{1, 0}},
		})
		p.blurB.DrawRectShader(sw, sh, p.blur, &ebiten.DrawRectShaderOptions{
			Blend: ebiten.BlendCopy, Images: [4]*ebiten.Image{p.blurA}, Uniforms: map[string]any{"Dir": []float32{0, 1}},
		})
		op = &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, Blend: ebiten.BlendLighter}
		op.GeoM.Scale(bloomScale, bloomScale)
		op.ColorScale.Scale(bloomStrength, bloomStrength, bloomStrength, bloomStrength)
		screen.DrawImage(p.blurB, op)
	}

	p.Stars.DrawHoles(screen)
}
