package engine

import (
	"math"

	"galaxy/internal/vec"
)

// The launch pad (T-066): a visual control around the planned object, after the maneuver nodes of
// Kerbal Space Program, the add tool of Universe Sandbox and the move disc of Homeworld.
//
//   - The ghost of the object sits on its position. Dragging it is a slingshot: pull back, and the
//     release launches it the other way.
//   - A ring around it in the plane moves it on the plane, a knob above it sets the height.
//   - Six colored handles push the velocity along the local axes: prograde/retrograde (yellow),
//     radially out/in (cyan), normal up/down (magenta).
//   - The tip of the velocity arrow can be dragged freely.
//
// Handles have a fixed size on screen like the gizmos of 3D editors; their world length follows
// from the depth. Dragging a handle by its own length changes the velocity by as much as the
// arrow shows for that length, so arrow and handles agree.

// Handle identifies a part of the launch pad.
type Handle int

const (
	HandleNone Handle = iota - 1
	HandleGhost
	HandleRing
	HandleHeight
	HandleTip
	HandlePrograde
	HandleRetrograde
	HandleOut
	HandleIn
	HandleUp
	HandleDown
)

// Sizes on screen, in points (scaled by the render scale).
const (
	axisPoints   = 64 // length of an axis handle
	ringPoints   = 30 // radius of the move ring
	hitPoints    = 12 // pick radius of a handle
	arrowPerCirc = 2.4
)

// pad is the geometry of the launch pad for the current view.
type pad struct {
	pos              vec.Vec3
	radial, prograde vec.Vec3
	normal           vec.Vec3
	a                float64 // world length of an axis handle
	k                float64 // world length of the arrow per velocity unit
}

// padGeometry measures the launch pad at the current camera; ok is false if the object is behind
// the camera or the view has no size.
func (e *Engine) padGeometry() (pad, bool) {
	l := &e.launch
	if !l.Active || e.Cam.H <= 0 {
		return pad{}, false
	}
	p := pad{pos: l.Position(), normal: l.Normal}
	p.radial, p.prograde = l.basis()
	vw := e.Cam.View()
	_, _, depth, ok := vw.Project(p.pos.X, p.pos.Y, p.pos.Z)
	if !ok {
		return pad{}, false
	}
	p.a = axisPoints * e.RenderScale() * depth / e.Cam.Focal()
	ref := l.Circular
	if ref <= 0 {
		ref = math.Max(l.Escape, 1)
	}
	p.k = arrowPerCirc * p.a / ref
	return p, true
}

// axis returns the world direction of an axis handle.
func (p pad) axis(h Handle) vec.Vec3 {
	switch h {
	case HandlePrograde:
		return p.prograde
	case HandleRetrograde:
		return p.prograde.Scale(-1)
	case HandleOut:
		return p.radial
	case HandleIn:
		return p.radial.Scale(-1)
	case HandleUp:
		return p.normal
	case HandleDown:
		return p.normal.Scale(-1)
	}
	return vec.Vec3{}
}

// anchor is the world position of a handle.
func (e *Engine) anchor(p pad, h Handle) vec.Vec3 {
	switch h {
	case HandleHeight:
		return p.pos.Add(p.normal.Scale(1.9 * p.a))
	case HandleTip:
		return p.pos.Add(e.launch.Velocity().Scale(p.k))
	case HandleGhost, HandleRing:
		return p.pos
	}
	return p.pos.Add(p.axis(h).Scale(p.a))
}

// visible reports whether the handle's axis shows on screen: axes pointing at the camera cannot be
// dragged and would cover the ghost.
func (e *Engine) visible(p pad, h Handle) bool {
	if h == HandleGhost || h == HandleRing || h == HandleTip {
		return true
	}
	x0, y0, ok0 := e.screen(p.pos)
	x1, y1, ok1 := e.screen(e.anchor(p, h))
	return ok0 && ok1 && math.Hypot(x1-x0, y1-y0) > 0.6*axisPoints*e.RenderScale()
}

// screen projects a world point; ok is false behind the camera.
func (e *Engine) screen(w vec.Vec3) (x, y float64, ok bool) {
	vw := e.Cam.View()
	x, y, _, ok = vw.Project(w.X, w.Y, w.Z)
	return x, y, ok
}

// HitLaunch returns the handle of the launch pad at (x, y) in render pixels, or HandleNone.
func (e *Engine) HitLaunch(x, y float64) Handle {
	p, ok := e.padGeometry()
	if !ok {
		return HandleNone
	}
	ps := e.RenderScale()
	hit := hitPoints * ps
	order := []Handle{HandleTip, HandleHeight, HandlePrograde, HandleRetrograde, HandleOut, HandleIn, HandleUp, HandleDown}
	if e.launch.Speed == 0 {
		order = order[1:] // the tip is at the ghost
	}
	for _, h := range order {
		if !e.visible(p, h) {
			continue
		}
		if hx, hy, ok := e.screen(e.anchor(p, h)); ok && math.Hypot(hx-x, hy-y) < hit {
			return h
		}
	}
	gx, gy, ok := e.screen(p.pos)
	if !ok {
		return HandleNone
	}
	d := math.Hypot(gx-x, gy-y)
	switch {
	case d < 1.2*hit:
		return HandleGhost
	case math.Abs(d-ringPoints*ps) < 0.6*hit:
		return HandleRing
	}
	return HandleNone
}

// drag is the state of a handle drag.
type drag struct {
	handle       Handle
	x0, y0       float64
	vel          vec.Vec3 // relative velocity at the start
	height       float64
	geometry     pad
	sling        vec.Vec3 // current pull of the slingshot (world)
	slingActive  bool
	launchOnDrop bool
}

// BeginDrag starts dragging handle h at (x, y).
func (e *Engine) BeginDrag(h Handle, x, y float64) {
	p, ok := e.padGeometry()
	if !ok || h == HandleNone {
		e.drag = drag{handle: HandleNone}
		return
	}
	e.drag = drag{handle: h, x0: x, y0: y, vel: e.launch.Velocity(), height: e.launch.Height, geometry: p}
}

// DragTo moves the dragged handle to (x, y); snap enables magnetic snapping (Cmd turns it off).
func (e *Engine) DragTo(x, y float64, snap bool) {
	d := &e.drag
	l := &e.launch
	p := d.geometry
	if d.handle == HandleNone || !l.Active {
		return
	}
	switch d.handle {
	case HandleRing:
		height := l.Height
		e.aimAt(x, y)
		l.Height = height
		e.planLaunch()
	case HandleHeight:
		l.Height = d.height + e.alongAxis(p, p.normal, x, y)*p.a
		e.planLaunch()
	case HandleTip, HandleGhost:
		hit, ok := e.onPlaneThrough(p.pos, p.normal, x, y)
		if !ok {
			return
		}
		pull := hit.Sub(p.pos)
		if d.handle == HandleTip {
			normal := p.normal.Scale(d.vel.Dot(p.normal))
			e.setVelocity(pull.Scale(1/p.k).Add(normal), snap)
			return
		}
		// Slingshot: the velocity points away from the pull.
		d.sling, d.slingActive = pull, true
		d.launchOnDrop = pull.Len() > 0.25*p.a
		if d.launchOnDrop {
			e.setVelocity(pull.Scale(-1/p.k), snap)
		}
	default:
		u := p.axis(d.handle)
		e.setVelocity(d.vel.Add(u.Scale(e.alongAxis(p, u, x, y)*p.a/p.k)), snap)
	}
}

// EndDrag finishes a drag; it reports whether the slingshot was released (the caller launches).
func (e *Engine) EndDrag() bool {
	release := e.drag.handle == HandleGhost && e.drag.launchOnDrop
	e.drag = drag{handle: HandleNone}
	return release
}

// alongAxis is the mouse movement since the drag started, along the screen image of world
// direction u, in multiples of the axis length.
func (e *Engine) alongAxis(p pad, u vec.Vec3, x, y float64) float64 {
	x0, y0, ok0 := e.screen(p.pos)
	x1, y1, ok1 := e.screen(p.pos.Add(u.Scale(p.a)))
	if !ok0 || !ok1 {
		return 0
	}
	sx, sy := x1-x0, y1-y0
	l2 := sx*sx + sy*sy
	if l2 < 4 { // the axis points at the camera: no usable direction on screen
		return 0
	}
	return ((x-e.drag.x0)*sx + (y-e.drag.y0)*sy) / l2
}

// onPlaneThrough intersects the view ray through (x, y) with the plane through p with normal n
// (or, if the ray grazes it, with the plane perpendicular to the view).
func (e *Engine) onPlaneThrough(p, n vec.Vec3, x, y float64) (vec.Vec3, bool) {
	eye := e.Cam.Position()
	dir := e.Cam.Unproject(x, y).Sub(eye)
	dir = dir.Scale(1 / dir.Len())
	if math.Abs(dir.Dot(n)) < 0.15 {
		n = e.Cam.Forward()
	}
	t := p.Sub(eye).Dot(n) / dir.Dot(n)
	if t <= 0 || math.IsNaN(t) || math.IsInf(t, 0) {
		return vec.Vec3{}, false
	}
	return eye.Add(dir.Scale(t)), true
}

// Gizmo line colors.
const (
	GizmoGrid = iota
	GizmoRadial
	GizmoDrop
	GizmoArrow
	GizmoPath
	GizmoCompass
	GizmoAxisPrograde
	GizmoAxisRadial
	GizmoAxisNormal
	GizmoSling
)

// Segment is a line on screen, in render pixels.
type Segment struct {
	X0, Y0, X1, Y1 float32
	Color          int
}

// Marker kinds: dots of handles, the ghost and anchors for labels.
const (
	MarkGhost = iota
	MarkPrograde
	MarkRetrograde
	MarkOut
	MarkIn
	MarkUp
	MarkDown
	MarkHeight
	MarkTip
	MarkLabelPrograde // anchors for the compass labels and the speedometer
	MarkLabelOut
	MarkLabelCenter
	MarkGauge
)

// Marker is a point of the launch pad on screen, in render pixels.
type Marker struct {
	X, Y float32
	Kind int
}

// Gizmo projects the launch pad for the current view: the rings of the plane around the attractor,
// the line to the position, the drop to the plane, the compass, the axis handles, the velocity
// arrow, the slingshot band and the preview path.
func (e *Engine) Gizmo(segs []Segment, marks []Marker) ([]Segment, []Marker) {
	segs, marks = segs[:0], marks[:0]
	l := &e.launch
	p, ok := e.padGeometry()
	if !ok {
		return segs, marks
	}
	line := func(a, b vec.Vec3, c int) {
		x0, y0, ok0 := e.screen(a)
		x1, y1, ok1 := e.screen(b)
		if ok0 && ok1 {
			segs = append(segs, Segment{float32(x0), float32(y0), float32(x1), float32(y1), c})
		}
	}
	mark := func(w vec.Vec3, kind int) {
		if x, y, ok := e.screen(w); ok {
			marks = append(marks, Marker{float32(x), float32(y), kind})
		}
	}
	circle := func(center, u, v vec.Vec3, r float64, c int) {
		const n = 72
		prev := center.Add(u.Scale(r))
		for i := 1; i <= n; i++ {
			phi := 2 * math.Pi * float64(i) / n
			q := center.Add(u.Scale(r * math.Cos(phi))).Add(v.Scale(r * math.Sin(phi)))
			line(prev, q, c)
			prev = q
		}
	}
	// The plane: rings around the attractor through the position.
	foot := l.Foot.Sub(l.Center)
	r := math.Hypot(foot.Dot(p.radial), foot.Dot(p.prograde))
	if r < 1e-9 {
		r = math.Max(l.Radius, 1)
	}
	for _, k := range []float64{0.5, 1, 1.5} {
		circle(l.Center, p.radial, p.prograde, k*r, GizmoGrid)
	}
	line(l.Center, l.Foot, GizmoRadial)
	if l.Height != 0 {
		line(l.Foot, p.pos, GizmoDrop)
	}
	// Compass and move ring in the plane through the position.
	ringWorld := float64(ringPoints) / axisPoints * p.a
	circle(p.pos, p.radial, p.prograde, ringWorld, GizmoCompass)
	for i := 0; i < 12; i++ {
		phi := 2 * math.Pi * float64(i) / 12
		u := p.radial.Scale(math.Cos(phi)).Add(p.prograde.Scale(math.Sin(phi)))
		inner := 0.85
		if i%3 == 0 {
			inner = 0.7
		}
		line(p.pos.Add(u.Scale(inner*ringWorld)), p.pos.Add(u.Scale(ringWorld)), GizmoCompass)
	}
	// Axis handles.
	for h, c := range map[Handle]int{
		HandlePrograde: GizmoAxisPrograde, HandleRetrograde: GizmoAxisPrograde,
		HandleOut: GizmoAxisRadial, HandleIn: GizmoAxisRadial, HandleUp: GizmoAxisNormal, HandleDown: GizmoAxisNormal,
	} {
		if e.visible(p, h) {
			line(p.pos.Add(p.axis(h).Scale(ringWorld)), e.anchor(p, h), c)
		}
	}
	if e.visible(p, HandleHeight) {
		line(e.anchor(p, HandleUp), e.anchor(p, HandleHeight), GizmoDrop)
	}
	for h, k := range map[Handle]int{
		HandlePrograde: MarkPrograde, HandleRetrograde: MarkRetrograde, HandleOut: MarkOut, HandleIn: MarkIn,
		HandleUp: MarkUp, HandleDown: MarkDown, HandleHeight: MarkHeight,
	} {
		if e.visible(p, h) {
			mark(e.anchor(p, h), k)
		}
	}
	mark(p.pos.Add(p.prograde.Scale(1.1*p.a)).Add(p.radial.Scale(0.75*p.a)), MarkLabelPrograde) // beside the arrow
	mark(p.pos.Add(p.radial.Scale(1.6*p.a)), MarkLabelOut)
	mark(p.pos.Add(p.radial.Scale(-1.6*p.a)), MarkLabelCenter)
	// Preview path, velocity arrow, slingshot band.
	for i := 1; i < len(l.path); i += 2 {
		line(l.path[i-1], l.path[min(i+1, len(l.path)-1)], GizmoPath)
	}
	if l.Speed > 0 {
		tip := e.anchor(p, HandleTip)
		line(p.pos, tip, GizmoArrow)
		x0, y0, ok0 := e.screen(p.pos)
		x1, y1, ok1 := e.screen(tip)
		if ok0 && ok1 {
			dx, dy := x1-x0, y1-y0
			if d := math.Hypot(dx, dy); d > 4 {
				ux, uy := dx/d, dy/d
				ps := e.RenderScale()
				for _, side := range []float64{1, -1} {
					segs = append(segs, Segment{float32(x1), float32(y1),
						float32(x1 - ps*(12*ux-7*side*uy)), float32(y1 - ps*(12*uy+7*side*ux)), GizmoArrow})
				}
			}
		}
		mark(tip, MarkTip)
	}
	if d := e.drag; d.handle == HandleGhost && d.slingActive {
		line(p.pos, p.pos.Add(d.sling), GizmoSling)
	}
	mark(p.pos, MarkGhost)
	mark(p.pos, MarkGauge)
	return segs, marks
}
