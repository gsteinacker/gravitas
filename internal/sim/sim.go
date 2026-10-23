package sim

import "math"

// Params are the physical and numerical parameters of a run, in units with G = 1.
type Params struct {
	Dt        float64     `json:"dt"`                // global time step
	Softening float64     `json:"softening"`         // Plummer softening length ε
	C         float64     `json:"c"`                 // speed of light; 0 disables relativistic effects (see relativity.go)
	Gravity   GravityMode `json:"gravity,omitempty"` // propagation of the gravity of black holes (retarded.go)
	// Gas configures the hydrodynamics of gas bodies (internal/sph); zero values disable parts.
	Gas GasParams `json:"gas,omitzero"`
	// DarkMatterOff switches off the gravity of dark matter: it still moves as test particles, so
	// switching it back on finds the halos (roughly) where they were.
	DarkMatterOff bool `json:"darkMatterOff,omitempty"`
}

// GasParams configure the gas model (ADR-0007). Energies are specific (per unit mass).
type GasParams struct {
	Gamma        float64 `json:"gamma,omitempty"`        // adiabatic index (0 means 5/3)
	Neighbors    float64 `json:"neighbors,omitempty"`    // target SPH neighbor number (0 means 40)
	Alpha        float64 `json:"alpha,omitempty"`        // artificial viscosity (0 means 1)
	UFloor       float64 `json:"uFloor,omitempty"`       // cooling floor of the internal energy
	CoolTime     float64 `json:"coolTime,omitempty"`     // relaxation time to UFloor (0: adiabatic)
	SFDensity    float64 `json:"sfDensity,omitempty"`    // star formation threshold density (0: none)
	SFEfficiency float64 `json:"sfEfficiency,omitempty"` // stars formed per free-fall time (0 means 0.02)
	YoungAge     float64 `json:"youngAge,omitempty"`     // stars younger than this are drawn young (0 means 5)
}

// Hydro computes the hydrodynamic part of the gas model.
type Hydro interface {
	// Forces adds pressure and viscosity accelerations of gas bodies to s.Acc*, fills s.Rho and s.H,
	// and writes du/dt and the velocity divergence of every body (0 for non-gas) into dudt and divv.
	Forces(s *State, p *Params, dudt, divv []float64)
}

// SourceMass is the gravitating mass of a body of kind k and mass m under p.
func (p *Params) SourceMass(k Kind, m float64) float64 {
	if k == DarkMatter && p.DarkMatterOff {
		return 0
	}
	return m
}

// Solver computes star–star gravitational accelerations. Pairs involving a black hole are
// ignored by solvers (holes get zero); the integrator adds them (split.go).
type Solver interface {
	// Accelerations fills s.Acc* for all bodies from star–star forces.
	Accelerations(s *State, p Params)
}

// Simulation advances a State with a kick-drift-kick leapfrog integrator.
type Simulation struct {
	State  *State
	Solver Solver
	Hydro  Hydro // nil: gas bodies feel gravity only
	Params Params
	Time   float64
	Steps  int
	// Epoch changes whenever bodies are added or the state is replaced, i.e. whenever the total
	// energy changes for reasons other than integration error.
	Epoch int
	// ExternalEnergy is the total energy change caused by accretion and mergers since Reset,
	// computed exactly at each event. Integration error is E - E0 - ExternalEnergy.
	ExternalEnergy float64

	// StarsFormed counts stars born from gas since Reset.
	StarsFormed int
	// Substeps is the number of inner substeps used near black holes in the last step.
	Substeps int

	accValid bool
	holes    []int // alive black holes, refreshed every step

	// Multiple time stepping buffers (split.go).
	isNear              []bool
	near                []int
	farX, farY, farZ    []float64
	fastX, fastY, fastZ []float64

	histories  map[uint64]*holeHistory // past positions of black holes by ID (retarded.go)
	dudt, divv []float64               // gas: du/dt and velocity divergence (gas.go)
	holeHists  []*holeHistory          // histories of m.holes, bound once per step
}

func New(s *State, solver Solver, p Params) *Simulation {
	return &Simulation{State: s, Solver: solver, Params: p}
}

// Reset replaces the state and restarts the clock.
func (m *Simulation) Reset(s *State, p Params) {
	m.State, m.Params, m.Time, m.Steps, m.accValid, m.ExternalEnergy = s, p, 0, 0, false, 0
	m.Epoch++
	m.histories = nil
}

// Restore resets the simulation to a saved state, including its clock and energy bookkeeping.
func (m *Simulation) Restore(s *State, p Params, time float64, steps int, external float64) {
	m.Reset(s, p)
	m.Time, m.Steps, m.ExternalEnergy = time, steps, external
}

// SetDarkMatterOff switches the gravity of dark matter off or on (see Params.DarkMatterOff).
func (m *Simulation) SetDarkMatterOff(off bool) {
	m.Params.DarkMatterOff = off
	m.accValid = false
	m.Epoch++ // the conserved energy changes its meaning
}

// AddBody adds a body during a run. Accelerations are recomputed before the next kick.
func (m *Simulation) AddBody(b Body) int {
	m.accValid = false
	m.Epoch++
	i := m.State.Add(b)
	if b.Kind == BlackHole {
		m.holeAppeared(i)
	}
	return i
}

// Step advances the simulation by one time step.
func (m *Simulation) Step() {
	s, dt := m.State, m.Params.Dt
	m.collectHoles()
	if len(m.holes) > 0 {
		m.stepSplit()
		if m.Params.C > 0 {
			m.accrete()
		}
	} else {
		if !m.accValid {
			m.computeSlow()
			m.accValid = true
		}
		kick(s, 0.5*dt)
		drift(s, dt, m.Params.C)
		m.computeSlow()
		kick(s, 0.5*dt)
		m.Substeps = 1
	}
	m.updateGas(dt)
	m.Time += dt
	m.Steps++
	m.recordHoles()
}

// kick changes the velocities by h·a. With C > 0 the stored velocity is the proper velocity
// u = γv, for which du/dt = a holds exactly (see relativity.go).
func (m *Simulation) collectHoles() {
	s := m.State
	m.holes = m.holes[:0]
	for i := range s.Len() {
		if s.Alive[i] && s.Kind[i] == BlackHole {
			m.holes = append(m.holes, i)
		}
	}
}

func kick(s *State, h float64) {
	n := s.Len()
	vx, vy, vz := s.VelX[:n], s.VelY[:n], s.VelZ[:n]
	ax, ay, az := s.AccX[:n], s.AccY[:n], s.AccZ[:n]
	for i := range vx {
		vx[i] += h * ax[i]
		vy[i] += h * ay[i]
		vz[i] += h * az[i]
	}
}

// drift moves the bodies by dt·v. With c > 0 the coordinate velocity is v = u / √(1 + u²/c²).
func drift(s *State, dt, c float64) {
	n := s.Len()
	px, py, pz := s.PosX[:n], s.PosY[:n], s.PosZ[:n]
	vx, vy, vz := s.VelX[:n], s.VelY[:n], s.VelZ[:n]
	if c <= 0 {
		for i := range px {
			px[i] += dt * vx[i]
			py[i] += dt * vy[i]
			pz[i] += dt * vz[i]
		}
		return
	}
	invC2 := 1 / (c * c)
	for i := range px {
		ux, uy, uz := vx[i], vy[i], vz[i]
		f := dt / math.Sqrt(1+(ux*ux+uy*uy+uz*uz)*invC2)
		px[i] += f * ux
		py[i] += f * uy
		pz[i] += f * uz
	}
}
