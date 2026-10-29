package sim

import (
	"math"
	"slices"
)

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
	// Infall adds fresh gas around galaxies (infall.go); set by the scene.
	Infall []InfallSource `json:"infall,omitempty"`
	// Cosmology switches to comoving integration in an expanding, periodic box (comoving.go).
	Cosmology *Cosmology `json:"cosmology,omitempty"`
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
	UIon         float64 `json:"uIon,omitempty"`         // internal energy of photoionized gas (0: no ionization)
	IonOpacity   float64 `json:"ionOpacity,omitempty"`   // optical depth per neutral column density (mass per area)
	IonRecomb    float64 `json:"ionRecomb,omitempty"`    // optical depth per ∫ρ_ion² dl (recombinations in ionized gas)
	IonEvery     int     `json:"ionEvery,omitempty"`     // recompute ionization every n steps (0 means 4)
	SNEnergy     float64 `json:"snEnergy,omitempty"`     // supernova energy per formed stellar mass (0: none)
	SNDelayMin   float64 `json:"snDelayMin,omitempty"`   // earliest explosion after birth (supernova.go)
	SNDelayMax   float64 `json:"snDelayMax,omitempty"`   // latest explosion after birth
}

// Hydro computes the hydrodynamic part of the gas model.
type Hydro interface {
	// Forces evaluates the gas bodies i with active[i] (all gas if active is nil) at the current
	// positions: it updates their s.Rho and s.H and writes their entries of out; other entries
	// stay as they are. Inactive gas enters as neighbors with its current (predicted) Rho, H, U
	// and velocity. step[i] is the step body i is currently integrated with (nil or 0: unknown);
	// out.Dt of an evaluated body is at most LimiterRatio times the smallest step among its
	// neighbors (Saitoh & Makino 2009). A neighbor j whose step is longer than LimiterRatio times
	// the out.Dt of an evaluated body gets out.Wake[j] lowered to that out.Dt, so it can be woken
	// up. Forces must not touch s.Acc*: it runs concurrently with the gravity solver.
	Forces(s *State, p *Params, active []bool, step []float64, out *HydroOut)
	// Bound lowers dt[i] of every gas body to the sound crossing step of its current internal
	// energy, which may have grown since the last evaluation (supernovae).
	Bound(s *State, p *Params, dt []float64)
}

// HydroOut is the result of hydro evaluations, per body.
type HydroOut struct {
	AccX, AccY, AccZ []float64 // pressure and viscosity acceleration
	DuDt, DivV       []float64 // change of the internal energy, velocity divergence
	Dt               []float64 // largest stable time step (Courant condition, step limiter)
	// Wake is, as float64 bits, the smallest Dt of an evaluated neighbor that requires a shorter
	// step of this body (+Inf: none); updated atomically, reset by the integrator.
	Wake []uint64
}

// Resize makes room for n bodies; new entries are zero.
func (o *HydroOut) Resize(n int) {
	for _, f := range []*[]float64{&o.AccX, &o.AccY, &o.AccZ, &o.DuDt, &o.DivV, &o.Dt} {
		old := len(*f)
		*f = resize(*f, n)
		if n > old {
			clear((*f)[old:])
		}
	}
	old := len(o.Wake)
	o.Wake = resize(o.Wake, n)
	for i := old; i < n; i++ {
		o.Wake[i] = noWake
	}
}

// noWake is the Wake value of a body no neighbor wants to wake up.
var noWake = math.Float64bits(math.Inf(1))

// LimiterRatio is the largest ratio of the time steps of neighboring gas bodies.
const LimiterRatio = 4

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
	// FormedMass is the mass of these stars.
	FormedMass float64
	// Supernovae counts populations that exploded since Reset, SNMass is their mass (supernova.go).
	Supernovae int
	SNMass     float64
	// InfallMass is the gas mass added by infall since Reset; InfallPending is the mass each
	// source has accumulated towards its next body (infall.go).
	InfallMass    float64
	InfallPending []float64
	// HolesFormed counts black holes formed by collapse since Reset (collapse.go).
	HolesFormed int
	// ScaleFactor is a in the comoving mode (comoving.go), 0 otherwise.
	ScaleFactor float64
	comoving    Solver // periodic force solver of the comoving mode
	// Substeps is the number of inner substeps used near black holes in the last step.
	Substeps int
	// HydroSubsteps is the number of hydro evaluations in the last step (individual gas steps,
	// blockstep.go); HydroLoad is their cost in units of an evaluation of all gas.
	HydroSubsteps int
	HydroLoad     float64

	accValid bool
	holes    []int // alive black holes, refreshed every step

	// Multiple time stepping buffers (split.go).
	isNear              []bool
	near                []int
	farX, farY, farZ    []float64
	fastX, fastY, fastZ []float64

	histories   map[uint64]*holeHistory // past positions of black holes by ID (retarded.go)
	accretion   map[uint64]float64      // recent accretion rate of black holes by ID (accrete.go)
	hydro       HydroOut                // last hydro evaluation of every gas body (gas.go)
	gas         gasSteps                // individual time steps of gas (blockstep.go)
	ion         ionizer                 // photoionization grid (ionize.go)
	holeHists   []*holeHistory          // histories of m.holes, bound once per step
	dragWeights []float64               // scratch of dragBack (framedrag.go)
}

func New(s *State, solver Solver, p Params) *Simulation {
	return &Simulation{State: s, Solver: solver, Params: p}
}

// Reset replaces the state and restarts the clock.
func (m *Simulation) Reset(s *State, p Params) {
	m.State, m.Params, m.Time, m.Steps, m.accValid, m.ExternalEnergy = s, p, 0, 0, false, 0
	m.StarsFormed, m.HolesFormed = 0, 0
	m.FormedMass, m.Supernovae, m.SNMass = 0, 0, 0
	m.InfallMass, m.InfallPending = 0, nil
	m.accretion = nil
	m.ScaleFactor, m.comoving = 0, nil
	m.Epoch++
	m.histories = nil
}

// Restore resets the simulation to a saved state, including its clock and energy bookkeeping.
func (m *Simulation) Restore(s *State, p Params, time float64, steps int, external float64) {
	m.Reset(s, p)
	m.Time, m.Steps, m.ExternalEnergy = time, steps, external
}

// Forces is the result of the last slow force evaluation. Saving it lets a run continue exactly
// without recomputing it, which matters for gas: the smoothing-length iteration is not idempotent.
type Forces struct {
	AccX, AccY, AccZ, Rho, DuDt, DivV []float64
	// Hydro accelerations and time steps of gas (since T-085; AccX… are gravity only).
	HAccX, HAccY, HAccZ, HDt []float64
}

// LastForces returns a copy of the last force evaluation, or nil if there is none.
func (m *Simulation) LastForces() *Forces {
	if !m.accValid {
		return nil
	}
	s := m.State
	return &Forces{
		AccX: slices.Clone(s.AccX), AccY: slices.Clone(s.AccY), AccZ: slices.Clone(s.AccZ),
		Rho: slices.Clone(s.Rho), DuDt: slices.Clone(m.hydro.DuDt), DivV: slices.Clone(m.hydro.DivV),
		HAccX: slices.Clone(m.hydro.AccX), HAccY: slices.Clone(m.hydro.AccY), HAccZ: slices.Clone(m.hydro.AccZ),
		HDt: slices.Clone(m.hydro.Dt),
	}
}

// RestoreForces installs a saved force evaluation for the current state (after Restore).
func (m *Simulation) RestoreForces(f *Forces) {
	s, n := m.State, m.State.Len()
	if f == nil || len(f.AccX) != n || len(f.Rho) != n {
		return
	}
	if m.Hydro != nil && m.hasGas() && len(f.HDt) != n {
		return // saved before T-085: AccX includes the hydro forces, recompute them
	}
	copy(s.AccX, f.AccX)
	copy(s.AccY, f.AccY)
	copy(s.AccZ, f.AccZ)
	copy(s.Rho, f.Rho)
	m.hydro = HydroOut{
		AccX: slices.Clone(f.HAccX), AccY: slices.Clone(f.HAccY), AccZ: slices.Clone(f.HAccZ),
		DuDt: slices.Clone(f.DuDt), DivV: slices.Clone(f.DivV), Dt: slices.Clone(f.HDt),
	}
	m.hydro.Resize(n)
	m.accValid = true
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

// Step advances the simulation by Params.Dt. Gas is integrated with individual time steps within
// it (blockstep.go).
func (m *Simulation) Step() {
	m.step()
}

func (m *Simulation) step() {
	if m.Params.Cosmology != nil {
		m.stepComoving()
		m.Steps++
		return
	}
	s, dt := m.State, m.Params.Dt
	m.collectHoles()
	if len(m.holes) > 0 {
		m.stepSplit()
		if m.Params.C > 0 {
			m.accrete()
		}
	} else {
		if !m.accValid {
			m.initForces()
			m.accValid = true
		}
		own := m.collectGas()
		kick(s, 0.5*dt)
		if own {
			m.openGas(dt)
		}
		drift(s, dt, m.Params.C, own)
		m.advanceGas(dt, nil)
		kick(s, 0.5*dt)
		m.Substeps = 1
	}
	m.updateGas(dt)
	m.supernovae(dt)
	m.infall(dt)
	m.collapse()
	m.decayAccretion(dt)
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

// drift moves the bodies by dt·v, except gas if skipGas (it moves in its own steps). With c > 0
// the coordinate velocity is v = u / √(1 + u²/c²).
func drift(s *State, dt, c float64, skipGas bool) {
	n := s.Len()
	if skipGas {
		for i := range n {
			if s.Kind[i] != Gas {
				driftOne(s, i, dt, c)
			}
		}
		return
	}
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

func driftOne(s *State, i int, dt, c float64) {
	f := dt / gamma(s.VelX[i], s.VelY[i], s.VelZ[i], c)
	s.PosX[i] += f * s.VelX[i]
	s.PosY[i] += f * s.VelY[i]
	s.PosZ[i] += f * s.VelZ[i]
}
