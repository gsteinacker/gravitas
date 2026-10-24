// Command icon turns a screenshot of the app into a macOS icon set: it crops a square around a
// point, places it into the rounded square of macOS app icons and writes all sizes of an
// .iconset directory, which iconutil converts into an .icns file (make icon).
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

// macOS icon grid: the artwork is a rounded square of 824 px in a 1024 px canvas.
const (
	canvas = 1024
	body   = 824
	radius = 185 // corner radius of the body
)

func main() {
	in := flag.String("in", "", "source PNG (screenshot with -hud=false)")
	out := flag.String("out", "Galaxy.iconset", "output .iconset directory")
	cx := flag.Float64("cx", 0.5, "crop center x, fraction of the width")
	cy := flag.Float64("cy", 0.5, "crop center y, fraction of the height")
	size := flag.Float64("size", 0.9, "crop size, fraction of the height")
	flag.Parse()
	if err := run(*in, *out, *cx, *cy, *size); err != nil {
		log.Fatal(err)
	}
}

func run(in, out string, cx, cy, size float64) error {
	f, err := os.Open(in)
	if err != nil {
		return err
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	b := src.Bounds()
	side := int(size * float64(b.Dy()))
	x0 := b.Min.X + int(cx*float64(b.Dx())) - side/2
	y0 := b.Min.Y + int(cy*float64(b.Dy())) - side/2
	crop := image.Rect(x0, y0, x0+side, y0+side).Intersect(b)

	art := image.NewRGBA(image.Rect(0, 0, body, body))
	draw.CatmullRom.Scale(art, art.Bounds(), src, crop, draw.Src, nil)
	icon := image.NewRGBA(image.Rect(0, 0, canvas, canvas))
	off := (canvas - body) / 2
	for y := range body {
		for x := range body {
			a := coverage(float64(x)+0.5, float64(y)+0.5)
			if a == 0 {
				continue
			}
			c := art.RGBAAt(x, y)
			// A faint rim keeps the dark icon visible on dark docks.
			rim := rimLight(float64(x)+0.5, float64(y)+0.5)
			r := math.Min(255, float64(c.R)+rim)
			g := math.Min(255, float64(c.G)+rim)
			bl := math.Min(255, float64(c.B)+rim*1.3)
			icon.SetRGBA(x+off, y+off, color.RGBA{uint8(r * a), uint8(g * a), uint8(bl * a), uint8(255 * a)})
		}
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	for _, s := range []int{16, 32, 128, 256, 512} {
		for _, k := range []int{1, 2} {
			name := fmt.Sprintf("icon_%dx%d.png", s, s)
			if k == 2 {
				name = fmt.Sprintf("icon_%dx%d@2x.png", s, s)
			}
			if err := write(filepath.Join(out, name), icon, s*k); err != nil {
				return err
			}
		}
	}
	return nil
}

// coverage is the anti-aliased opacity of the rounded body at (x, y) in body coordinates.
func coverage(x, y float64) float64 {
	dx := math.Max(0, math.Max(radius-x, x-(body-radius)))
	dy := math.Max(0, math.Max(radius-y, y-(body-radius)))
	d := math.Hypot(dx, dy) - radius // signed distance outside the rounded corner
	return math.Max(0, math.Min(1, 0.5-d))
}

// rimLight brightens a thin band along the edge of the body.
func rimLight(x, y float64) float64 {
	dx := math.Max(0, math.Max(radius-x, x-(body-radius)))
	dy := math.Max(0, math.Max(radius-y, y-(body-radius)))
	inside := radius - math.Hypot(dx, dy) // distance to the rounded edge, inside the corners
	edge := math.Min(math.Min(x, y), math.Min(body-x, body-y))
	if dx > 0 && dy > 0 {
		edge = inside
	}
	return 28 * math.Exp(-edge/3)
}

func write(path string, img *image.RGBA, size int) error {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, dst); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
