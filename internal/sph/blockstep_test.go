package sph

import (
	"math"
	"slices"
	"testing"

	"galaxy/internal/sim"
	"galaxy/internal/sim/direct"
)

// hotCoreSphere is gasSphere with a hot core: its bodies need much shorter steps than the rest,
// so the gas spreads over several rungs (individual time steps, T-085).
func hotCoreSphere(n int) *sim.State {
	s := gasSphere(n)
	for i := range s.Len() {
		if s.PosX[i]*s.PosX[i]+s.PosY[i]*s.PosY[i]+s.PosZ[i]*s.PosZ[i] < 0.3*0.3 {
			s.U[i] = 20
		}
	}
	return s
}

func momentum(s *sim.State) (p, scale float64) {
	var px, py, pz float64
	for i := range s.Len() {
		px, py, pz = px+s.Mass[i]*s.VelX[i], py+s.Mass[i]*s.VelY[i], pz+s.Mass[i]*s.VelZ[i]
		scale += s.Mass[i] * math.Sqrt(s.VelX[i]*s.VelX[i]+s.VelY[i]*s.VelY[i]+s.VelZ[i]*s.VelZ[i])
	}
	return math.Sqrt(px*px + py*py + pz*pz), scale
}

// A hot core expanding into cooler gas: bodies on different rungs must still conserve energy
// and momentum (adiabatic, exact gravity), and the steps must really differ. The same run with a
// global split of the step (before T-085) changed the energy by 1.5 %; individual steps give
// 1.9 % (ADR-0018), the momentum error stays below 1e-5.
func TestIndividualGasStepsConserveEnergyAndMomentum(t *testing.T) {
	m := sim.New(hotCoreSphere(2000), &direct.Solver{}, sim.Params{Dt: 0.001, Softening: 0.05})
	m.Hydro = &Solver{}
	m.Step()
	e0 := totalEnergy(m)
	p0, _ := momentum(m.State)
	var events int
	var load float64
	const steps = 300
	for range steps {
		m.Step()
		events += m.HydroSubsteps
		load += m.HydroLoad
	}
	if float64(events) < 1.05*steps || load > 1.05*steps {
		t.Errorf("%d evaluations with load %.1f in %d steps: the gas did not use individual steps", events, load, steps)
	}
	dE := math.Abs((totalEnergy(m) - e0) / e0)
	p, scale := momentum(m.State)
	if dE > 2.5e-2 {
		t.Errorf("relative energy change %.3g over %d steps", dE, steps)
	}
	if dp := math.Abs(p-p0) / scale; dp > 1e-4 {
		t.Errorf("relative momentum change %.3g over %d steps", dp, steps)
	}
	t.Logf("%.2f evaluations per step, load %.3f; energy %.2g, momentum %.2g", float64(events)/steps, load/steps, dE, math.Abs(p-p0)/scale)
}

// Individual steps depend only on the state: two runs agree bit by bit.
func TestIndividualGasStepsAreReproducible(t *testing.T) {
	run := func() *sim.State {
		m := sim.New(hotCoreSphere(1000), &direct.Solver{}, sim.Params{Dt: 0.004, Softening: 0.05})
		m.Hydro = &Solver{}
		for range 30 {
			m.Step()
		}
		return m.State
	}
	a, b := run(), run()
	if !slices.Equal(a.PosX, b.PosX) || !slices.Equal(a.VelY, b.VelY) || !slices.Equal(a.U, b.U) {
		t.Error("two runs with individual gas steps differ")
	}
}
