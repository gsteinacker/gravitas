package engine

import (
	"math"
	"slices"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

// Launch plans the start of a placed object (T-064): a position on a defined plane plus height,
// and a velocity relative to the galaxy it belongs to, given as speed, azimuth and elevation.
//
//   - The plane goes through the attractor (the black hole nearest to the click on screen, or the
//     center of mass) and is either its disk plane (normal along the angular momentum of the stars
//     around it) or perpendicular to the view.
//   - Azimuth 0° is the prograde circular direction (along the rotation of the disk), 90° radially
//     outward, 180° retrograde, -90° inward; elevation tilts out of the plane.
//   - Speeds are relative to the attractor's velocity. The circular and escape speeds at the
//     position (from the frozen field of the current snapshot) are offered as references.
type Launch struct {
	Active     bool
	Plane      LaunchPlane
	Height     float64 // along the plane normal, length units
	Azimuth    float64 // radians
	Elevation  float64 // radians
	Speed      float64 // velocity units, relative to the attractor
	Center     vec.Vec3
	CenterVel  vec.Vec3
	Normal     vec.Vec3
	Foot       vec.Vec3 // the clicked point on the plane
	Circular   float64  // circular speed at the position
	Escape     float64  // escape speed from the whole scene at the position
	Radius     float64  // distance from the attractor
	Impact     float64  // impact parameter: closest approach to the center on a straight line
	field      []vec.Vec3
	fieldMass  []float64
	soft2      float64
	path       []vec.Vec3 // preview, world positions
	diskNormal vec.Vec3
}

// LaunchPlane chooses the plane objects are placed on.
type LaunchPlane int

const (
	PlaneDisk   LaunchPlane = iota // the attractor's disk plane
	PlaneCamera                    // through the attractor, perpendicular to the view
)

// Launch presets.
const (
	PresetRest = iota
	PresetCircular
	PresetEscape
	PresetCenter
)

// maxFieldPoints bounds the bodies of the frozen field (black holes always count fully).
const maxFieldPoints = 3000

// previewSteps is the number of leapfrog steps of the trajectory preview.
const previewSteps = 600

// BeginLaunch starts planning a launch at screen point (x, y) in render pixels, keeping the
// direction, speed and plane of a previous plan.
func (e *Engine) BeginLaunch(x, y float64) {
	l := &e.launch
	keep := l.Active || l.Speed != 0 || l.Azimuth != 0
	e.freezeField(x, y)
	if !keep {
		l.Height, l.Azimuth, l.Elevation, l.Speed = 0, 0, 0, 0
	}
	l.Active = true
	e.Hint("orbital-speeds")
	e.aimAt(x, y)
	e.planLaunch()
}

// freezeField copies a thinned field of the latest snapshot and finds the attractor of (x, y).
func (e *Engine) freezeField(x, y float64) {
	l := &e.launch
	l.field, l.fieldMass = l.field[:0], l.fieldMass[:0]
	l.soft2 = e.Scene.Params.Softening * e.Scene.Params.Softening
	var holes []int
	e.Runner.View(func(sn *sim.Snapshot) {
		n := sn.Len()
		step := max(1, n/maxFieldPoints)
		var com, comV vec.Vec3
		var total float64
		for i := range n {
			m := float64(sn.Mass[i])
			p := vec.Vec3{X: float64(sn.X[i]), Y: float64(sn.Y[i]), Z: float64(sn.Z[i])}
			com = com.Add(p.Scale(m))
			comV = comV.Add(vec.Vec3{X: float64(sn.VX[i]), Y: float64(sn.VY[i]), Z: float64(sn.VZ[i])}.Scale(m))
			total += m
			switch {
			case sn.Kind[i] == sim.BlackHole:
				holes = append(holes, i)
				l.field, l.fieldMass = append(l.field, p), append(l.fieldMass, m)
			case i%step == 0:
				l.field, l.fieldMass = append(l.field, p), append(l.fieldMass, m*float64(step))
			}
		}
		l.Center, l.CenterVel = com.Scale(1/total), comV.Scale(1/total)
		// The attractor: the hole nearest to the click on screen.
		vw := e.Cam.View()
		best := math.Inf(1)
		for _, i := range holes {
			sx, sy, _, ok := vw.Project(float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i]))
			if d := math.Hypot(sx-x, sy-y); ok && d < best {
				best = d
				l.Center = vec.Vec3{X: float64(sn.X[i]), Y: float64(sn.Y[i]), Z: float64(sn.Z[i])}
				l.CenterVel = vec.Vec3{X: float64(sn.VX[i]), Y: float64(sn.VY[i]), Z: float64(sn.VZ[i])}
			}
		}
		l.diskNormal = diskNormal(sn, l.Center, l.CenterVel)
	})
}

// diskNormal is the direction of the angular momentum of the stars nearest to center (the disk the
// attractor belongs to); +z if there is none.
func diskNormal(sn *sim.Snapshot, center, centerVel vec.Vec3) vec.Vec3 {
	type near struct {
		d2 float64
		i  int
	}
	var stars []near
	for i := range sn.Len() {
		if sn.Kind[i] != sim.Star && sn.Kind[i] != sim.Gas {
			continue
		}
		d := vec.Vec3{X: float64(sn.X[i]), Y: float64(sn.Y[i]), Z: float64(sn.Z[i])}.Sub(center)
		stars = append(stars, near{d.Dot(d), i})
	}
	slices.SortFunc(stars, func(a, b near) int { return int(math.Copysign(1, a.d2-b.d2)) })
	var lsum vec.Vec3
	for _, s := range stars[:min(len(stars), 2000)] {
		i := s.i
		r := vec.Vec3{X: float64(sn.X[i]), Y: float64(sn.Y[i]), Z: float64(sn.Z[i])}.Sub(center)
		v := vec.Vec3{X: float64(sn.VX[i]), Y: float64(sn.VY[i]), Z: float64(sn.VZ[i])}.Sub(centerVel)
		lsum = lsum.Add(r.Cross(v).Scale(float64(sn.Mass[i])))
	}
	if lsum.Len() == 0 {
		return vec.Vec3{Z: 1}
	}
	return lsum.Scale(1 / lsum.Len())
}

// aimAt puts the foot of the launch where the view ray through (x, y) meets the plane.
func (e *Engine) aimAt(x, y float64) {
	l := &e.launch
	l.Normal = l.diskNormal
	eye := e.Cam.Position()
	dir := e.Cam.Unproject(x, y).Sub(eye)
	dir = dir.Scale(1 / dir.Len())
	if l.Plane == PlaneCamera || math.Abs(dir.Dot(l.Normal)) < 0.08 { // grazing: use the view plane
		l.Normal = e.Cam.Forward().Scale(-1)
	}
	t := l.Center.Sub(eye).Dot(l.Normal) / dir.Dot(l.Normal)
	if t <= 0 || math.IsInf(t, 0) || math.IsNaN(t) {
		l.Foot = e.Cam.Unproject(x, y)
		return
	}
	l.Foot = eye.Add(dir.Scale(t))
}

// Position is where the object starts.
func (l Launch) Position() vec.Vec3 { return l.Foot.Add(l.Normal.Scale(l.Height)) }

// basis returns the radial (in the plane, outward) and prograde directions at the position.
func (l Launch) basis() (radial, prograde vec.Vec3) {
	r := l.Position().Sub(l.Center)
	r = r.Sub(l.Normal.Scale(r.Dot(l.Normal)))
	if r.Len() < 1e-9 {
		r = l.Normal.Cross(vec.Vec3{X: 1})
		if r.Len() < 1e-6 {
			r = l.Normal.Cross(vec.Vec3{Y: 1})
		}
	}
	radial = r.Scale(1 / r.Len())
	return radial, l.Normal.Cross(radial)
}

// Velocity is the start velocity relative to the attractor.
func (l Launch) Velocity() vec.Vec3 {
	radial, prograde := l.basis()
	inPlane := prograde.Scale(math.Cos(l.Azimuth)).Add(radial.Scale(math.Sin(l.Azimuth)))
	dir := inPlane.Scale(math.Cos(l.Elevation)).Add(l.Normal.Scale(math.Sin(l.Elevation)))
	return dir.Scale(l.Speed)
}

// field returns the acceleration and potential of the frozen field at p.
func (l *Launch) fieldAt(p vec.Vec3) (vec.Vec3, float64) {
	var a vec.Vec3
	var phi float64
	for i, q := range l.field {
		d := q.Sub(p)
		r2 := d.Dot(d) + l.soft2
		inv := 1 / math.Sqrt(r2)
		phi -= l.fieldMass[i] * inv
		a = a.Add(d.Scale(l.fieldMass[i] * inv * inv * inv))
	}
	return a, phi
}

// planLaunch updates the references and the preview after a change.
func (e *Engine) planLaunch() {
	l := &e.launch
	p := l.Position()
	r := p.Sub(l.Center)
	l.Radius = r.Len()
	a, phi := l.fieldAt(p)
	l.Circular = 0
	if l.Radius > 0 {
		inward := -a.Dot(r.Scale(1 / l.Radius))
		l.Circular = math.Sqrt(math.Max(0, inward*l.Radius))
	}
	l.Escape = math.Sqrt(math.Max(0, -2*phi))
	l.Impact = l.Radius
	if v := l.Velocity(); v.Len() > 0 {
		l.Impact = r.Cross(v.Scale(1 / v.Len())).Len()
	}
	l.preview()
}

// preview integrates a test particle in the frozen field, in the attractor's frame, for about one
// and a half circular orbits (leapfrog).
func (l *Launch) preview() {
	l.path = l.path[:0]
	p, v := l.Position(), l.Velocity()
	scale := math.Max(l.Circular, 1e-6)
	horizon := 1.5 * 2 * math.Pi * math.Max(l.Radius, 1e-3) / scale
	if l.Circular == 0 {
		horizon = 4 * math.Max(l.Radius, 1) / math.Max(l.Speed, 1e-3)
	}
	dt := horizon / previewSteps
	a, _ := l.fieldAt(p)
	l.path = append(l.path, p)
	for range previewSteps {
		v = v.Add(a.Scale(dt / 2))
		p = p.Add(v.Scale(dt))
		a, _ = l.fieldAt(p)
		v = v.Add(a.Scale(dt / 2))
		l.path = append(l.path, p)
	}
}

// SetLaunch changes one parameter of the plan: plane (0 disk, 1 camera), height, azimuth and
// elevation (radians), speed (velocity units).
func (e *Engine) SetLaunch(name string, v float64) {
	l := &e.launch
	if !l.Active {
		return
	}
	switch name {
	case "plane":
		l.Plane = LaunchPlane(max(0, min(1, int(v))))
		l.Normal = l.diskNormal
		if l.Plane == PlaneCamera {
			l.Normal = e.Cam.Forward().Scale(-1)
		}
		// Keep the foot's distance from the center but move it into the new plane.
		r := l.Foot.Sub(l.Center)
		l.Foot = l.Center.Add(r.Sub(l.Normal.Scale(r.Dot(l.Normal))))
	case "height":
		l.Height = v
	case "azimuth":
		l.Azimuth = v
	case "elevation":
		l.Elevation = math.Max(-math.Pi/2, math.Min(math.Pi/2, v))
	case "speed":
		l.Speed = math.Max(0, v)
	}
	e.planLaunch()
}

// LaunchPreset sets a standard start: at rest relative to the attractor, on a circular orbit,
// with escape speed radially outward, or falling towards the center.
func (e *Engine) LaunchPreset(preset int) {
	l := &e.launch
	if !l.Active {
		return
	}
	l.Elevation = 0
	switch preset {
	case PresetRest:
		l.Speed, l.Azimuth = 0, 0
	case PresetCircular:
		l.Speed, l.Azimuth = l.Circular, 0
	case PresetEscape:
		l.Speed, l.Azimuth = l.Escape, math.Pi/2
	case PresetCenter:
		l.Speed, l.Azimuth = math.Max(l.Speed, 0.5*l.Circular), -math.Pi/2
	}
	e.planLaunch()
}

// DragLaunch sets direction and speed from a drag on screen from the start point to (x1, y1):
// the direction is the drag in the plane, and a drag as long as the distance to the center gives
// the circular speed.
func (e *Engine) DragLaunch(x1, y1 float64) {
	l := &e.launch
	if !l.Active {
		return
	}
	foot, height := l.Foot, l.Height
	e.aimAt(x1, y1)
	end := l.Foot
	l.Foot, l.Height = foot, height
	d := end.Sub(foot)
	d = d.Sub(l.Normal.Scale(d.Dot(l.Normal)))
	if d.Len() < 1e-9 {
		return
	}
	radial, prograde := l.basis()
	l.Azimuth = math.Atan2(d.Dot(radial), d.Dot(prograde))
	l.Elevation = 0
	ref := l.Circular
	if ref == 0 {
		ref = 1
	}
	l.Speed = d.Len() / math.Max(l.Radius, 1e-6) * ref
	e.planLaunch()
}

// GoLaunch places the current tool's object as planned and ends the plan.
func (e *Engine) GoLaunch() {
	l := &e.launch
	if !l.Active {
		return
	}
	e.PlaceAt(l.Position(), l.CenterVel.Add(l.Velocity()))
	l.Active = false
}

// CancelLaunch ends the plan without placing anything.
func (e *Engine) CancelLaunch() { e.launch.Active = false }

// LaunchPlan returns the current plan.
func (e *Engine) LaunchPlan() Launch { return e.launch }

// Gizmo line colors.
const (
	GizmoGrid = iota
	GizmoRadial
	GizmoDrop
	GizmoArrow
	GizmoPath
)

// Segment is a line on screen, in render pixels.
type Segment struct {
	X0, Y0, X1, Y1 float32
	Color          int
}

// Gizmo projects the plan for the current view: rings of the plane around the attractor, the line
// to the position, the drop to the plane, the velocity arrow and the preview path.
func (e *Engine) Gizmo(dst []Segment) []Segment {
	dst = dst[:0]
	l := &e.launch
	if !l.Active {
		return dst
	}
	vw := e.Cam.View()
	line := func(a, b vec.Vec3, c int) {
		x0, y0, _, ok0 := vw.Project(a.X, a.Y, a.Z)
		x1, y1, _, ok1 := vw.Project(b.X, b.Y, b.Z)
		if ok0 && ok1 {
			dst = append(dst, Segment{float32(x0), float32(y0), float32(x1), float32(y1), c})
		}
	}
	radial, prograde := l.basis()
	r := math.Hypot(l.Foot.Sub(l.Center).Dot(radial), l.Foot.Sub(l.Center).Dot(prograde))
	if r < 1e-6 {
		r = math.Max(l.Radius, 1)
	}
	for _, k := range []float64{0.5, 1, 1.5} {
		const seg = 72
		var prev vec.Vec3
		for i := 0; i <= seg; i++ {
			phi := 2 * math.Pi * float64(i) / seg
			p := l.Center.Add(radial.Scale(k * r * math.Cos(phi))).Add(prograde.Scale(k * r * math.Sin(phi)))
			if i > 0 {
				line(prev, p, GizmoGrid)
			}
			prev = p
		}
	}
	pos := l.Position()
	line(l.Center, l.Foot, GizmoRadial)
	if l.Height != 0 {
		line(l.Foot, pos, GizmoDrop)
	}
	for i := 1; i < len(l.path); i += 2 {
		line(l.path[i-1], l.path[min(i+1, len(l.path)-1)], GizmoPath)
	}
	if l.Speed > 0 {
		ref := math.Max(l.Circular, l.Speed)
		tip := pos.Add(l.Velocity().Scale(0.6 * r / ref))
		line(pos, tip, GizmoArrow)
		// The arrow head in screen space.
		x0, y0, _, ok0 := vw.Project(pos.X, pos.Y, pos.Z)
		x1, y1, _, ok1 := vw.Project(tip.X, tip.Y, tip.Z)
		if ok0 && ok1 {
			dx, dy := x1-x0, y1-y0
			if d := math.Hypot(dx, dy); d > 4 {
				ux, uy := dx/d, dy/d
				for _, side := range []float64{1, -1} {
					dst = append(dst, Segment{float32(x1), float32(y1),
						float32(x1 - 12*ux + 7*side*uy), float32(y1 - 12*uy - 7*side*ux), GizmoArrow})
				}
			}
		}
	}
	return dst
}
