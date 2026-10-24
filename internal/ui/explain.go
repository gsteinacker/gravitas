package ui

import (
	"bytes"
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"galaxy/internal/i18n"
)

// Learning mode (T-039): e opens a panel with the scene introduction and explanations of the
// concepts, ←/→ switch the topic, ↑/↓ the audience. The state lives in app.Controller.

var (
	regularSource = mustFace(goregular.TTF)
	boldSource    = mustFace(gobold.TTF)
	bodyFace      = &text.GoTextFace{Source: regularSource, Size: 16}
	smallFace     = &text.GoTextFace{Source: regularSource, Size: 13}
	titleFace     = &text.GoTextFace{Source: boldSource, Size: 22}
)

func mustFace(ttf []byte) *text.GoTextFaceSource {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(ttf))
	if err != nil {
		panic(err) // embedded font
	}
	return s
}

var (
	panelColor  = color.RGBA{6, 9, 22, 225}
	accentColor = color.RGBA{255, 170, 70, 255}
	dimColor    = color.RGBA{150, 160, 185, 255}
	hudColor    = color.RGBA{215, 220, 235, 255}
)

// panelWidth is the width of the explanation panel for a screen width.
func panelWidth(screenW int) float64 { return min(600, float64(screenW)*0.45) }

func (g *Game) drawExplanation(screen *ebiten.Image) {
	w := panelWidth(g.screenW)
	x0, y0 := float64(g.screenW)-w-16, 16.0
	h := float64(g.screenH) - 32
	vector.FillRect(screen, float32(x0), float32(y0), float32(w), float32(h), panelColor, false)
	pad := 22.0
	x, y := x0+pad, y0+pad
	title, body := g.Explanation()
	drawText(screen, title, titleFace, x, y, color.White)
	y += 34
	drawText(screen, i18n.T("Zielgruppe: ", "Audience: ")+g.Level.Name()+"   ("+strconv.Itoa(g.TopicIdx+1)+"/"+strconv.Itoa(len(g.Topics))+")", smallFace, x, y, accentColor)
	y += 30
	for _, line := range wrap(body, bodyFace, w-2*pad) {
		if y > y0+h-60 {
			drawText(screen, "…", bodyFace, x, y, color.White)
			break
		}
		drawText(screen, line, bodyFace, x, y, color.White)
		y += 23
	}
	drawText(screen, i18n.T("←/→ Thema   ↑/↓ Zielgruppe   e oder Esc schließen", "←/→ topic   ↑/↓ audience   e or Esc to close"), smallFace, x, y0+h-pad-14, dimColor)
}

func drawText(dst *ebiten.Image, s string, face *text.GoTextFace, x, y float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
}

// wrap breaks s into lines no wider than width; "\n" starts a new paragraph (with an empty line
// for "\n\n").
func wrap(s string, face text.Face, width float64) []string {
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			if w, _ := text.Measure(line+" "+word, face, 0); w > width {
				lines = append(lines, line)
				line = word
			} else {
				line += " " + word
			}
		}
		lines = append(lines, line)
	}
	return lines
}
