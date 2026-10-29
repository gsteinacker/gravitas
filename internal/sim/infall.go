package sim

import (
	"math"

	"galaxy/internal/vec"
)

// Gas infall (T-081, ADR-0016): real disk galaxies keep forming stars because fresh gas keeps
// raining in from their halos, about as fast as they turn it into stars (Fraternali & Binney 2008,
// Putman et al. 2012). Each InfallSource adds gas bodies of a fixed mass at a mean rate around a
// center body (the galaxy's black hole): above or below the disk plane at the height of the
// extraplanar gas, at a radius drawn from the exponential gas disk, rotating with a lag behind
// the circular speed. The disk plane follows the angular momentum of the baryons around the
// center, so it also works for tilted, colliding and merged galaxies. The energy of every new
// body (kinetic, thermal and potential) is booked in ExternalEnergy, like accretion.

// InfallSource describes the gas falling onto one galaxy.
type InfallSource struct {
	Center uint64  `json:"center"` // ID of the body the gas falls towards (the galaxy's black hole)
	Rate   float64 `json:"rate"`   // mass per time unit
	Mass   float64 `json:"mass"`   // mass of a new gas body
	Scale  float64 `json:"scale"`  // exponential scale length of the landing radius
	RMax   float64 `json:"rMax"`   // largest landing radius
	Height float64 `json:"height"` // starting height above or below the disk plane
}

const (
	// infallLag is the rotation of extraplanar gas relative to the circular speed: it lags by
	// 15–25 km/s per kpc of height (Fraternali & Binney 2006), about 20 % at 1–2 kpc.
	infallLag     = 0.8
	infallSalt    = 0x1f4a11
	maxInfallStep = 4 // bodies per source and step at most
)

// infall adds the gas due in this step.
func (m *Simulation) infall(dt float64) {
	src := m.Params.Infall
	if len(src) == 0 || m.Params.Cosmology != nil {
		return
	}
	if len(m.InfallPending) != len(src) {
		m.InfallPending = make([]float64, len(src))
	}
	for k := range src {
		f := &src[k]
		if f.Rate <= 0 || f.Mass <= 0 {
			continue
		}
		m.InfallPending[k] += f.Rate * dt
		for j := range maxInfallStep {
			if m.InfallPending[k] < f.Mass {
				break
			}
			c := m.infallCenter(f)
			if c < 0 {
				m.InfallPending[k] = math.Min(m.InfallPending[k], f.Mass)
				break
			}
			m.InfallPending[k] -= f.Mass
			m.addInfall(f, c, uint64(k)<<48^uint64(m.Steps)<<3^uint64(j))
		}
	}
}

// infallCenter returns the index of the center body of f, or after it was swallowed in a merger,
// that of the heaviest black hole (which f follows from then on); -1 without any.
func (m *Simulation) infallCenter(f *InfallSource) int {
	s := m.State
	best := -1
	for i := range s.Len() {
		if !s.Alive[i] {
			continue
		}
		if s.ID[i] == f.Center {
			return i
		}
		if s.Kind[i] == BlackHole && (best < 0 || s.Mass[i] > s.Mass[best]) {
			best = i
		}
	}
	if best >= 0 {
		f.Center = s.ID[best]
	}
	return best
}

// addInfall adds one gas body of source f around body c; seed makes it reproducible.
func (m *Simulation) addInfall(f *InfallSource, c int, seed uint64) {
	s, p := m.State, &m.Params
	rnd := func(j uint64) float64 { return unitRandom(seed, infallSalt+j) }

	// Landing radius: surface density ∝ exp(−R/Scale), i.e. R ~ Gamma(2, Scale), up to RMax.
	R := f.RMax
	for j := range uint64(8) {
		if r := -f.Scale * math.Log((1-rnd(2*j))*(1-rnd(2*j+1))); r <= f.RMax {
			R = r
			break
		}
	}
	center := vec.Vec3{X: s.PosX[c], Y: s.PosY[c], Z: s.PosZ[c]}
	vCenter := CoordinateVelocity(vec.Vec3{X: s.VelX[c], Y: s.VelY[c], Z: s.VelZ[c]}, p.C)
	r3 := math.Hypot(R, f.Height)
	normal, inside := m.diskFrame(c, center, vCenter, math.Max(f.RMax, r3), r3)
	e1 := normal.Cross(vec.Vec3{X: 1})
	if e1.Len() < 0.1 {
		e1 = normal.Cross(vec.Vec3{Y: 1})
	}
	e1 = e1.Scale(1 / e1.Len())
	e2 := normal.Cross(e1)
	phi := 2 * math.Pi * rnd(20)
	radial := e1.Scale(math.Cos(phi)).Add(e2.Scale(math.Sin(phi)))
	side := 1.0
	if rnd(21) < 0.5 {
		side = -1
	}
	pos := center.Add(radial.Scale(R)).Add(normal.Scale(side * f.Height))
	vc := math.Sqrt(inside / math.Max(r3, 1e-12))
	vel := vCenter.Add(normal.Cross(radial).Scale(infallLag * vc))
	u := ProperVelocity(vel, p.C)

	b := Body{Pos: pos, Vel: u, Mass: f.Mass, Kind: Gas, U: p.Gas.UFloor}
	m.ExternalEnergy += kinetic(b.Mass, u.X, u.Y, u.Z, p.C) + b.Mass*b.U + m.potentialOf(b)
	i := s.Add(b)
	s.H[i] = 2 * p.Softening // a first guess; the SPH iteration adapts it
	m.accValid = false
	m.InfallMass += f.Mass
}

// diskFrame returns the direction of the angular momentum of stars and gas within rMax of the
// center (relative to its motion; +z if there is none) and the gravitating mass within r.
func (m *Simulation) diskFrame(c int, center, vCenter vec.Vec3, rMax, r float64) (normal vec.Vec3, inside float64) {
	s, p := m.State, &m.Params
	var L vec.Vec3
	for i := range s.Len() {
		if !s.Alive[i] {
			continue
		}
		d := vec.Vec3{X: s.PosX[i] - center.X, Y: s.PosY[i] - center.Y, Z: s.PosZ[i] - center.Z}
		d2 := d.Dot(d)
		if d2 <= r*r || i == c {
			inside += p.SourceMass(s.Kind[i], s.Mass[i])
		}
		if d2 > rMax*rMax || (s.Kind[i] != Star && s.Kind[i] != Gas) {
			continue
		}
		v := CoordinateVelocity(vec.Vec3{X: s.VelX[i], Y: s.VelY[i], Z: s.VelZ[i]}, p.C).Sub(vCenter)
		L = L.Add(d.Cross(v).Scale(s.Mass[i]))
	}
	if l := L.Len(); l > 0 {
		return L.Scale(1 / l), inside
	}
	return vec.Vec3{Z: 1}, inside
}

// potentialOf is the potential energy of body b with all bodies of the state (pair model).
func (m *Simulation) potentialOf(b Body) float64 {
	s, p := m.State, &m.Params
	eps2 := p.Softening * p.Softening
	var e float64
	for j := range s.Len() {
		if !s.Alive[j] {
			continue
		}
		dx, dy, dz := s.PosX[j]-b.Pos.X, s.PosY[j]-b.Pos.Y, s.PosZ[j]-b.Pos.Z
		mj := p.SourceMass(s.Kind[j], s.Mass[j])
		e += PairPotential(mj, b.Mass, dx*dx+dy*dy+dz*dz, eps2, PairRS(s.Kind[j], mj, b.Kind, b.Mass, p.C))
	}
	return e
}
