package sim

import "math"

// Supernova feedback (T-080, ADR-0015): a star particle formed from gas is a population of
// 10⁵–10⁶ M☉. Its massive stars (> 8 M☉) explode as core-collapse supernovae between about 3 and
// 40 Myr after its birth. The model lets every such population explode once, at a delay drawn
// uniformly from [SNDelayMin, SNDelayMax] (deterministic from its ID), and adds SNEnergy times its
// mass as thermal energy to its snNeighbors nearest gas bodies (equal Δu, so Δu·Σm = E). The
// heated gas no longer forms stars (u < 3 u_floor in updateGas), expands and cools again: star
// formation regulates itself. Initial stars (Born = -Inf) never explode.
const (
	snNeighbors = 32
	snSalt      = 0x5a17e5
)

// SupernovaDelay is the time after its birth at which the population with this ID explodes.
func SupernovaDelay(id uint64, lo, hi float64) float64 {
	return lo + (hi-lo)*unitRandom(id, snSalt)
}

// supernovae lets every population whose explosion time falls into [Time, Time+dt) explode. The
// intervals of successive steps are contiguous, so each population explodes exactly once.
func (m *Simulation) supernovae(dt float64) {
	g := &m.Params.Gas
	if g.SNEnergy <= 0 {
		return
	}
	s := m.State
	for i := range s.Len() {
		if s.Kind[i] != Star || math.IsInf(s.Born[i], -1) {
			continue
		}
		t := s.Born[i] + SupernovaDelay(s.ID[i], g.SNDelayMin, g.SNDelayMax)
		if t < m.Time || t >= m.Time+dt {
			continue
		}
		e := g.SNEnergy * s.Mass[i]
		m.ExternalEnergy += m.heatNeighbors(i, e)
		m.Supernovae++
		m.SNMass += s.Mass[i]
	}
}

// heatNeighbors adds the energy e to the snNeighbors gas bodies nearest to body i and returns the
// energy actually deposited (0 without gas). A linear scan: explosions are rare compared to steps.
func (m *Simulation) heatNeighbors(i int, e float64) float64 {
	s := m.State
	var idx [snNeighbors]int
	var d2 [snNeighbors]float64
	k := 0
	far := 0 // position of the largest distance in the list once it is full
	for j := range s.Len() {
		if s.Kind[j] != Gas {
			continue
		}
		dx, dy, dz := s.PosX[j]-s.PosX[i], s.PosY[j]-s.PosY[i], s.PosZ[j]-s.PosZ[i]
		r2 := dx*dx + dy*dy + dz*dz
		if k < snNeighbors {
			idx[k], d2[k] = j, r2
			k++
			if k == snNeighbors {
				far = farthest(d2[:])
			}
			continue
		}
		if r2 < d2[far] {
			idx[far], d2[far] = j, r2
			far = farthest(d2[:])
		}
	}
	var mass float64
	for _, j := range idx[:k] {
		mass += s.Mass[j]
	}
	if mass <= 0 {
		return 0
	}
	du := e / mass
	for _, j := range idx[:k] {
		s.U[j] += du
	}
	return e
}

func farthest(d2 []float64) int {
	f := 0
	for i, v := range d2 {
		if v > d2[f] {
			f = i
		}
	}
	return f
}
