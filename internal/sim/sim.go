package sim

// Params are the physical and numerical parameters of a run, in units with G = 1.
type Params struct {
	Dt        float64 // global time step
	Softening float64 // Plummer softening length ε
	C         float64 // speed of light; 0 disables relativistic effects
}

// Solver computes gravitational accelerations.
type Solver interface {
	// Accelerations fills s.Acc* for all bodies.
	Accelerations(s *State, p Params)
}

// Simulation advances a State with a kick-drift-kick leapfrog integrator.
type Simulation struct {
	State  *State
	Solver Solver
	Params Params
	Time   float64
	Steps  int
	// Epoch changes whenever bodies are added or the state is replaced, i.e. whenever the total
	// energy changes for reasons other than integration error.
	Epoch int

	accValid bool
}

func New(s *State, solver Solver, p Params) *Simulation {
	return &Simulation{State: s, Solver: solver, Params: p}
}

// Reset replaces the state and restarts the clock.
func (m *Simulation) Reset(s *State, p Params) {
	m.State, m.Params, m.Time, m.Steps, m.accValid = s, p, 0, 0, false
	m.Epoch++
}

// AddBody adds a body during a run. Accelerations are recomputed before the next kick.
func (m *Simulation) AddBody(b Body) int {
	m.accValid = false
	m.Epoch++
	return m.State.Add(b)
}

// Step advances the simulation by one time step.
func (m *Simulation) Step() {
	s, dt := m.State, m.Params.Dt
	if !m.accValid {
		m.Solver.Accelerations(s, m.Params)
		m.accValid = true
	}
	kick(s, 0.5*dt)
	drift(s, dt)
	m.Solver.Accelerations(s, m.Params)
	kick(s, 0.5*dt)
	m.Time += dt
	m.Steps++
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

func drift(s *State, dt float64) {
	n := s.Len()
	px, py, pz := s.PosX[:n], s.PosY[:n], s.PosZ[:n]
	vx, vy, vz := s.VelX[:n], s.VelY[:n], s.VelZ[:n]
	for i := range px {
		px[i] += dt * vx[i]
		py[i] += dt * vy[i]
		pz[i] += dt * vz[i]
	}
}
