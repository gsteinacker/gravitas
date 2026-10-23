package ui

import (
	"bytes"
	"image/color"
	"slices"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"galaxy/internal/learn"
)

// Learning mode (T-039): e opens a panel with the scene introduction and explanations of the
// concepts, ←/→ switch the topic, ↑/↓ the audience. Toggling a feature shows a one-line hint and
// focuses its topic.

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

// introTopic is the pseudo topic for the scene introduction (first entry of the list).
const introTopic = ""

// topicList is the scene introduction, the scene's topics, then all others.
func topicList(sceneTopics []string) []string {
	list := append([]string{introTopic}, sceneTopics...)
	for _, id := range learn.IDs() {
		if !slices.Contains(list, id) {
			list = append(list, id)
		}
	}
	return list
}

// hintFor maps typed characters to the topic they touch.
var hintFor = map[rune]string{
	'x': "dark-matter", 'X': "dark-matter", 'v': "dark-matter", 'V': "dark-matter",
	'o': "gas", 'O': "gas", 'u': "dust", 'U': "dust",
	'<': "speed-of-light", '>': "speed-of-light", 'p': "retarded-gravity", 'P': "retarded-gravity",
	'd': "doppler", 'D': "doppler", 'l': "lensing", 'L': "lensing",
}

// hint shows the one-line hint of a topic and makes it the one e opens.
func (g *Game) hint(id string) {
	t := learn.Get(id)
	if t == nil {
		return
	}
	g.focus(id)
	g.notify(t.Short + "   (e: mehr)")
}

func (g *Game) focus(id string) {
	if k := slices.Index(g.topics, id); k >= 0 {
		g.topicIdx = k
	}
}

// OpenExplanation opens the panel for an audience ("school", "lay", "physics") and topic ("" for
// the scene introduction); used by the -explain flag and for screenshots.
func (g *Game) OpenExplanation(level, topic string) {
	for l := learn.School; l <= learn.Physics; l++ {
		if l.Key() == level {
			g.level = l
		}
	}
	g.focus(topic)
	g.explain = true
}

// handleExplainKeys handles the panel navigation; it reports whether the panel is open.
func (g *Game) handleExplainKeys() bool {
	if !g.explain {
		return false
	}
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowRight):
		g.topicIdx = (g.topicIdx + 1) % len(g.topics)
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft):
		g.topicIdx = (g.topicIdx + len(g.topics) - 1) % len(g.topics)
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowDown):
		g.level = g.level.Next()
	case inpututil.IsKeyJustPressed(ebiten.KeyArrowUp):
		g.level = g.level.Prev()
	case inpututil.IsKeyJustPressed(ebiten.KeyEscape):
		g.explain = false
	}
	return true
}

// currentExplanation returns title and text of the focused entry.
func (g *Game) currentExplanation() (title, body string) {
	id := g.topics[g.topicIdx]
	if id == introTopic {
		return g.scene.Name, g.scene.Intro[g.level.Key()]
	}
	t := learn.Get(id)
	body = t.Text(g.level)
	if t.Keys != "" {
		body += "\n\nTasten: " + t.Keys
	}
	return t.Title, body
}

var (
	panelColor  = color.RGBA{6, 9, 22, 225}
	accentColor = color.RGBA{255, 170, 70, 255}
	dimColor    = color.RGBA{150, 160, 185, 255}
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
	title, body := g.currentExplanation()
	drawText(screen, title, titleFace, x, y, color.White)
	y += 34
	drawText(screen, "Zielgruppe: "+g.level.Name()+"   ("+strconv.Itoa(g.topicIdx+1)+"/"+strconv.Itoa(len(g.topics))+")", smallFace, x, y, accentColor)
	y += 30
	for _, line := range wrap(body, bodyFace, w-2*pad) {
		if y > y0+h-60 {
			drawText(screen, "…", bodyFace, x, y, color.White)
			break
		}
		drawText(screen, line, bodyFace, x, y, color.White)
		y += 23
	}
	drawText(screen, "←/→ Thema   ↑/↓ Zielgruppe   e oder Esc schließen", smallFace, x, y0+h-pad-14, dimColor)
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
