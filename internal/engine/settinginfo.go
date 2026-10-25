package engine

import (
	"math"

	"galaxy/internal/app"
	"galaxy/internal/i18n"
	"galaxy/internal/learn"
	"galaxy/internal/perf"
	"galaxy/internal/sim"
)

// SettingInfo is the (i) explanation of a control (T-061): text for the chosen audience, the
// learning topic to continue with and a chart computed from the current state.
type SettingInfo struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	Topic      string `json:"topic,omitempty"`
	TopicTitle string `json:"topicTitle,omitempty"`
	Chart      *Chart `json:"chart,omitempty"`
}

// Chart is drawn by the app: lines (with log axes), bars, or a fixed sketch (Kind "retarded").
type Chart struct {
	Kind   string      `json:"kind"` // "line", "bar" or a sketch name
	Title  string      `json:"title"`
	X      string      `json:"x"`
	Y      string      `json:"y"`
	LogX   bool        `json:"logX"`
	LogY   bool        `json:"logY"`
	Series []Series    `json:"series,omitempty"`
	Marker *[2]float64 `json:"marker,omitempty"` // the current setting
	Bars   []Bar       `json:"bars,omitempty"`
}

// Series is one line; Color is "red", "green", "blue", "orange" or "" (default).
type Series struct {
	Name   string       `json:"name"`
	Color  string       `json:"color,omitempty"`
	Points [][2]float64 `json:"points"`
}

// Bar is one bar of a bar chart; Current marks the active choice.
type Bar struct {
	Label   string  `json:"label"`
	Value   float64 `json:"value"`
	Current bool    `json:"current"`
}

// SettingInfo returns the explanation of a control, false for unknown ids.
func (e *Engine) SettingInfo(id string) (SettingInfo, bool) {
	s := learn.SettingByID(id)
	if s == nil {
		return SettingInfo{}, false
	}
	info := SettingInfo{Title: s.Title.String(), Text: s.Text(e.Level), Topic: s.Topic}
	if t := learn.Get(s.Topic); t != nil {
		info.TopicTitle = t.Title
	}
	switch s.Chart {
	case "c":
		info.Chart = e.speedOfLightChart()
	case "theta":
		info.Chart = e.thetaChart()
	case "dust":
		info.Chart = dustChart()
	case "mass":
		info.Chart = e.massChart()
	case "profile":
		info.Chart = e.profileChart()
	case "retarded":
		info.Chart = &Chart{Kind: "retarded", Title: i18n.T("Wohin zieht das Loch?", "Where does the hole pull from?")}
	}
	return info, true
}

// logRange returns n values evenly spaced in log between lo and hi.
func logRange(lo, hi float64, n int) []float64 {
	v := make([]float64, n)
	for i := range v {
		v[i] = lo * math.Pow(hi/lo, float64(i)/float64(n-1))
	}
	return v
}

// lengthUnit is the factor and name for lengths shown in charts (pc with scene units).
func (e *Engine) lengthUnit() (float64, string) {
	if _, pc, ok := e.Units(); ok {
		return pc, "pc"
	}
	return 1, i18n.T("Einheiten", "units")
}

// massUnit is the factor and name for masses shown in charts (M☉ with scene units).
func (e *Engine) massUnit() (float64, string) {
	if msun, _, ok := e.Units(); ok {
		return msun, "M☉"
	}
	return 1, i18n.T("Einheiten", "units")
}

// speedOfLightChart shows the horizon and ISCO radius of the largest hole over c.
func (e *Engine) speedOfLightChart() *Chart {
	m, c := 0.0, 0.0
	e.Runner.View(func(sn *sim.Snapshot) {
		c = sn.C
		for i, k := range sn.Kind {
			if k == sim.BlackHole {
				m = math.Max(m, float64(sn.Mass[i]))
			}
		}
	})
	if m == 0 {
		m = e.Mass
	}
	lf, ln := e.lengthUnit()
	ch := &Chart{Kind: "line", Title: i18n.T("Größe des Lochs über c", "Size of the hole over c"),
		X: "c", Y: i18n.T("Radius", "radius") + " (" + ln + ")", LogX: true, LogY: true}
	horizon, isco := Series{Name: i18n.T("Horizont r_s", "horizon r_s"), Color: "orange"}, Series{Name: "ISCO 3 r_s", Color: "blue"}
	for _, x := range logRange(app.CMin, app.CMax, 40) {
		rs := sim.SchwarzschildRadius(m, x) * lf
		horizon.Points = append(horizon.Points, [2]float64{x, rs})
		isco.Points = append(isco.Points, [2]float64{x, 3 * rs})
	}
	ch.Series = []Series{horizon, isco}
	if c > 0 {
		ch.Marker = &[2]float64{c, sim.SchwarzschildRadius(m, c) * lf}
	}
	return ch
}

// thetaMeasurements are the force errors (median) and times of perf-log T-034 (spiral galaxy with
// halo, 150k, CPU).
var thetaMeasurements = [][3]float64{{0.5, 7.8e-4, 193}, {0.6, 1.5e-3, 126}, {0.7, 2.5e-3, 90}, {0.8, 3.9e-3, 51}, {0.9, 5.7e-3, 33}}

func (e *Engine) thetaChart() *Chart {
	ch := &Chart{Kind: "line", Title: i18n.T("Fehler und Rechenzeit über θ (gemessen)", "Error and time over θ (measured)"),
		X: "θ", Y: "%", LogY: true}
	errS := Series{Name: i18n.T("Kraftfehler (Median)", "force error (median)"), Color: "red"}
	timeS := Series{Name: i18n.T("Rechenzeit (θ 0,5 = 100 %)", "time (θ 0.5 = 100 %)"), Color: "blue"}
	for _, m := range thetaMeasurements {
		errS.Points = append(errS.Points, [2]float64{m[0], 100 * m[1]})
		timeS.Points = append(timeS.Points, [2]float64{m[0], 100 * m[2] / thetaMeasurements[0][2]})
	}
	ch.Series = []Series{errS, timeS}
	ch.Marker = &[2]float64{e.ThetaUser, 100 * interpolate(thetaMeasurements, e.ThetaUser)}
	return ch
}

// interpolate returns the error at theta, linear in the measurements and clamped to their range.
func interpolate(m [][3]float64, theta float64) float64 {
	if theta <= m[0][0] {
		return m[0][1]
	}
	for i := 1; i < len(m); i++ {
		if theta <= m[i][0] {
			f := (theta - m[i-1][0]) / (m[i][0] - m[i-1][0])
			return m[i-1][1] + f*(m[i][1]-m[i-1][1])
		}
	}
	return m[len(m)-1][1]
}

// dustChart shows the transmission of the color channels over the optical depth in V.
func dustChart() *Chart {
	ch := &Chart{Kind: "line", Title: i18n.T("Durchgelassenes Licht je Farbe", "Light let through per color"),
		X: i18n.T("optische Tiefe τ (V)", "optical depth τ (V)"), Y: "%"}
	for k, name := range []string{"red", "green", "blue"} {
		s := Series{Name: [...]string{i18n.T("Rot", "red"), i18n.T("Grün", "green"), i18n.T("Blau", "blue")}[k], Color: name}
		for i := range 31 {
			tau := float64(i) / 10
			s.Points = append(s.Points, [2]float64{tau, 100 * math.Exp(-tau*dustK[k])})
		}
		ch.Series = append(ch.Series, s)
	}
	return ch
}

// massChart shows the radius of the next object over its mass for the current tool.
func (e *Engine) massChart() *Chart {
	mf, mn := e.massUnit()
	lf, ln := e.lengthUnit()
	c := 0.0
	e.Runner.View(func(sn *sim.Snapshot) { c = sn.C })
	radius := func(m float64) float64 { return e.ToolRadius(e.Tool, m) }
	name := i18n.T("Radius", "radius")
	switch e.Tool {
	case app.ToolBlackHole:
		if c <= 0 {
			return nil // no horizon without a speed of light
		}
		radius = func(m float64) float64 { return sim.SchwarzschildRadius(m, c) }
		name = i18n.T("Horizont r_s", "horizon r_s")
	case app.ToolGalaxy:
		return nil
	}
	ch := &Chart{Kind: "line", Title: i18n.T("Größe über Masse", "Size over mass"),
		X: i18n.T("Masse", "mass") + " (" + mn + ")", Y: name + " (" + ln + ")", LogX: true, LogY: true}
	s := Series{Name: name, Color: "orange"}
	for _, m := range logRange(math.Max(app.MassMin, 1e-6*app.MassMax), app.MassMax, 50) {
		s.Points = append(s.Points, [2]float64{m * mf, radius(m) * lf})
	}
	ch.Series = []Series{s}
	ch.Marker = &[2]float64{e.Mass * mf, radius(e.Mass) * lf}
	return ch
}

// profileChart shows the stars each profile chooses on this computer.
func (e *Engine) profileChart() *Chart {
	ch := &Chart{Kind: "bar", Title: i18n.T("Sterne je Profil auf diesem Rechner", "Stars per profile on this computer"),
		Y: i18n.T("Sterne", "stars")}
	for _, p := range perf.Profiles {
		ch.Bars = append(ch.Bars, Bar{Label: p.Name(), Value: float64(p.Stars(e.cal, defaultStars)), Current: p.ID == e.settings.Profile})
	}
	return ch
}
