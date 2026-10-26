package sim

import (
	"cmp"
	"math"
	"slices"

	"galaxy/internal/vec"
)

// Black holes from collapse (T-067, hoop conjecture): mass M inside a sphere smaller than its
// Schwarzschild radius 2GM/c² must form a black hole. The simulation cannot resolve anything
// below the softening, so it tests spheres of radius R = collapseRadius·ε: if they hold at least
// M_crit = R c²/2 (i.e. r_s(M) ≥ R) in at least minCollapseBodies bodies, those bodies become one
// hole. With the real c this practically never happens; with a reduced c, a collapsing cloud or a
// dense cluster forms one, which makes it visible but is a deliberate exaggeration (ADR-0003).
//
// The search runs every collapseEvery steps: bodies are binned into cubic cells of size R, sorted
// by cell, and spheres are tested around the center of mass of every heavy enough cell.
const (
	collapseRadius    = 2 // in softening lengths: the smallest resolved sphere
	minCollapseBodies = 20
	collapseEvery     = 10
)

// collapse turns every resolved region inside its own Schwarzschild radius into a black hole.
// Mass and momentum are conserved, the energy change is booked in ExternalEnergy, and the angular
// momentum about the center of mass becomes the spin of the hole.
func (m *Simulation) collapse() {
	s, c, eps := m.State, m.Params.C, m.Params.Softening
	if c <= 0 || eps <= 0 || m.Steps%collapseEvery != 0 {
		return
	}
	r := collapseRadius * eps
	crit := r * c * c / 2
	var total float64
	for i, k := range s.Kind {
		if k != BlackHole {
			total += s.Mass[i]
		}
	}
	if total < crit { // cheap exit for every ordinary scene
		return
	}
	cells := m.collapseCells(r)
	formed := false
	for _, cl := range cells {
		if cl.mass < crit/27 { // a sphere of radius R touches at most 27 cells of size R
			continue
		}
		members, mass := m.sphere(cl.x, cl.y, cl.z, r)
		if mass < crit || len(members) < minCollapseBodies {
			continue
		}
		h := s.Add(Body{Pos: centerOfMass(s, members), Kind: BlackHole})
		for _, i := range members {
			m.merge(h, i)
		}
		s.capSpin(h, c)
		m.HolesFormed++
		formed = true
	}
	if formed {
		s.Compact()
		m.accValid = false
	}
}

type collapseCell struct {
	key           int64
	mass, x, y, z float64 // mass and center of mass of the cell
}

// collapseCells bins the alive bodies (no holes) into cells of size r and returns the occupied
// cells with mass and center of mass.
func (m *Simulation) collapseCells(r float64) []collapseCell {
	s := m.State
	type entry struct {
		key int64
		i   int
	}
	entries := make([]entry, 0, s.Len())
	for i := range s.Len() {
		if s.Kind[i] == BlackHole || !s.Alive[i] {
			continue
		}
		entries = append(entries, entry{cellKey(s.PosX[i], s.PosY[i], s.PosZ[i], r), i})
	}
	slices.SortFunc(entries, func(a, b entry) int { return cmp.Compare(a.key, b.key) })
	var cells []collapseCell
	for k := 0; k < len(entries); {
		cl := collapseCell{key: entries[k].key}
		for ; k < len(entries) && entries[k].key == cl.key; k++ {
			i := entries[k].i
			cl.mass += s.Mass[i]
			cl.x += s.Mass[i] * s.PosX[i]
			cl.y += s.Mass[i] * s.PosY[i]
			cl.z += s.Mass[i] * s.PosZ[i]
		}
		if cl.mass > 0 {
			cl.x, cl.y, cl.z = cl.x/cl.mass, cl.y/cl.mass, cl.z/cl.mass
		}
		cells = append(cells, cl)
	}
	// The heaviest cells first, so a collapse takes its core rather than an edge.
	slices.SortFunc(cells, func(a, b collapseCell) int { return cmp.Compare(b.mass, a.mass) })
	return cells
}

// cellKey packs the integer cell coordinates (21 bits each, offset) into one key.
func cellKey(x, y, z, r float64) int64 {
	const off = 1 << 20
	f := func(v float64) int64 {
		return min(max(int64(math.Floor(v/r))+off, 0), 2*off-1)
	}
	return f(x)<<42 | f(y)<<21 | f(z)
}

// sphere returns the alive bodies (no holes) within r of (x, y, z) and their mass.
func (m *Simulation) sphere(x, y, z, r float64) ([]int, float64) {
	s := m.State
	var members []int
	var mass float64
	for i := range s.Len() {
		if s.Kind[i] == BlackHole || !s.Alive[i] {
			continue
		}
		dx, dy, dz := s.PosX[i]-x, s.PosY[i]-y, s.PosZ[i]-z
		if dx*dx+dy*dy+dz*dz < r*r {
			members = append(members, i)
			mass += s.Mass[i]
		}
	}
	return members, mass
}

func centerOfMass(s *State, members []int) (p vec.Vec3) {
	var mass float64
	for _, i := range members {
		p.X += s.Mass[i] * s.PosX[i]
		p.Y += s.Mass[i] * s.PosY[i]
		p.Z += s.Mass[i] * s.PosZ[i]
		mass += s.Mass[i]
	}
	return p.Scale(1 / mass)
}
