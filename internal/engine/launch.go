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
	Active    bool
	Plane     LaunchPlane
	Height    float64 // along the plane normal, length units
	Azimuth   float64 // radians
	Elevation float64 // radians
	Speed     float64 // velocity units, relative to the attractor
	Center    vec.Vec3
	CenterVel vec.Vec3
	Normal    vec.Vec3
	Foot      vec.Vec3 // the clicked point on the plane
	Circular  float64  // circular speed at the position
	Escape    float64  // escape speed from the whole scene at the position
	Radius    float64  // distance from the attractor
	Impact    float64  // impact parameter: closest approach to the center on a straight line
	// Binary: the object and the attractor hole orbit their common center of mass (momentum is
	// conserved: the hole gets the opposite share of the relative velocity).
	Binary     bool
	HoleMass   float64 // mass of the attractor if it is a black hole, else 0
	holeVel    vec.Vec3
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
	PresetBinary
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
		l.HoleMass = 0
		for _, i := range holes {
			sx, sy, _, ok := vw.Project(float64(sn.X[i]), float64(sn.Y[i]), float64(sn.Z[i]))
			if d := math.Hypot(sx-x, sy-y); ok && d < best {
				best = d
				l.Center = vec.Vec3{X: float64(sn.X[i]), Y: float64(sn.Y[i]), Z: float64(sn.Z[i])}
				l.CenterVel = vec.Vec3{X: float64(sn.VX[i]), Y: float64(sn.VY[i]), Z: float64(sn.VZ[i])}
				l.HoleMass = float64(sn.Mass[i])
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
	l.Binary = false
	switch preset {
	case PresetRest:
		l.Speed, l.Azimuth = 0, 0
	case PresetCircular:
		l.Speed, l.Azimuth = l.Circular, 0
	case PresetEscape:
		l.Speed, l.Azimuth = l.Escape, math.Pi/2
	case PresetCenter:
		l.Speed, l.Azimuth = math.Max(l.Speed, 0.5*l.Circular), -math.Pi/2
	case PresetBinary:
		// Relative circular orbit of the pair: the object's own mass adds to the pull.
		l.Binary = l.HoleMass > 0
		l.Speed, l.Azimuth = math.Sqrt(l.Circular*l.Circular+e.Mass/math.Max(l.Radius, 1e-9)), 0
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

// GoLaunch places the current tool's object as planned and ends the plan. For a binary, the
// relative velocity is split by mass so the pair's momentum is that of the hole before.
func (e *Engine) GoLaunch() {
	l := &e.launch
	if !l.Active {
		return
	}
	v := l.Velocity()
	if l.Binary && l.HoleMass > 0 {
		m, mh := e.Mass, l.HoleMass
		center, holeVel := l.Center, l.CenterVel.Sub(v.Scale(m/(m+mh)))
		e.Runner.Do(func(s *sim.Simulation) { setHoleVelocity(s, center, holeVel) })
		v = v.Scale(mh / (m + mh))
	}
	radial, _ := l.basis()
	e.PlaceNormal, e.PlaceRadial = l.Normal, radial // a galaxy's disk lies in the launch plane
	e.PlaceAt(l.Position(), l.CenterVel.Add(v))
	l.Active = false
}

// HoleKick is the speed change the attractor hole gets from a binary launch (momentum split by
// mass): the lighter partner takes most of the relative motion.
func (e *Engine) HoleKick() float64 {
	l := &e.launch
	if !l.Binary || l.HoleMass <= 0 {
		return 0
	}
	return l.Speed * e.Mass / (e.Mass + l.HoleMass)
}

// setHoleVelocity gives the black hole at center (as frozen at planning) the coordinate velocity v.
func setHoleVelocity(s *sim.Simulation, center, v vec.Vec3) {
	st := s.State
	best, bi := math.Inf(1), -1
	for i := range st.Len() {
		if st.Kind[i] != sim.BlackHole {
			continue
		}
		d := vec.Vec3{X: st.PosX[i], Y: st.PosY[i], Z: st.PosZ[i]}.Sub(center).Len()
		if d < best {
			best, bi = d, i
		}
	}
	if bi < 0 {
		return
	}
	u := sim.ProperVelocity(v, s.Params.C)
	st.VelX[bi], st.VelY[bi], st.VelZ[bi] = u.X, u.Y, u.Z
	s.Epoch++ // the energy changed by intent
}

// setVelocity sets the relative velocity from a vector; with snap, the speed snaps to the circular
// and escape speed and the direction to the axes of the disk when they are close.
func (e *Engine) setVelocity(v vec.Vec3, snap bool) {
	l := &e.launch
	radial, prograde := l.basis()
	l.Speed = v.Len()
	if l.Speed < 1e-12 {
		l.Speed, l.Azimuth, l.Elevation = 0, 0, 0
		e.planLaunch()
		return
	}
	l.Elevation = math.Asin(math.Max(-1, math.Min(1, v.Dot(l.Normal)/l.Speed)))
	l.Azimuth = math.Atan2(v.Dot(radial), v.Dot(prograde))
	if snap {
		for _, ref := range []float64{l.Circular, l.Escape} {
			if ref > 0 && math.Abs(l.Speed-ref) < snapSpeed*ref {
				l.Speed = ref
			}
		}
		if q := math.Round(l.Azimuth / (math.Pi / 2)); math.Abs(l.Azimuth-q*math.Pi/2) < snapAngle {
			l.Azimuth = q * math.Pi / 2
		}
		if math.Abs(l.Elevation) < snapAngle {
			l.Elevation = 0
		}
	}
	e.planLaunch()
}

// Snapping tolerances: 4 % of a reference speed, 6° of an axis.
const (
	snapSpeed = 0.04
	snapAngle = 6 * math.Pi / 180
)

// CancelLaunch ends the plan without placing anything.
func (e *Engine) CancelLaunch() { e.launch.Active = false }

// LaunchPlan returns the current plan.
func (e *Engine) LaunchPlan() Launch { return e.launch }
