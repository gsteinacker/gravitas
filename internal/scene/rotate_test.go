package scene

import (
	"math"
	"testing"

	"galaxy/internal/sim"
	"galaxy/internal/vec"
)

func angularMomentumZ(s *sim.State) float64 {
	var l float64
	for i := range s.Len() {
		l += s.Mass[i] * (s.PosX[i]*s.VelY[i] - s.PosY[i]*s.VelX[i])
	}
	return l
}

func TestSpinUpCloudGivesBeta(t *testing.T) {
	const n, f = 4000, 0.4
	s := sim.NewState(n)
	AddGasClump(s, n, 1, 1, vec.Vec3{}, vec.Vec3{}, 0.01, 0, 0, 7)
	k0 := sim.KineticEnergy(s, 0)
	SpinUpCloud(s, 0, vec.Vec3{}, vec.Vec3{Z: 1}, f)
	_, w := sim.Energy(s, 0)
	beta := (sim.KineticEnergy(s, 0) - k0) / math.Abs(w)
	if math.Abs(beta-CloudBeta(f))/CloudBeta(f) > 0.1 {
		t.Errorf("β = %.4f, want %.4f", beta, CloudBeta(f))
	}
	if angularMomentumZ(s) <= 0 {
		t.Error("cloud does not rotate about +z")
	}
}

func TestSpinUpClusterKeepsEnergyAndRotates(t *testing.T) {
	const n = 3000
	s := sim.NewState(n)
	AddPlummer(s, n, 1, 0.5, vec.Vec3{}, vec.Vec3{}, 3)
	k0, l0 := sim.KineticEnergy(s, 0), angularMomentumZ(s)
	SpinUpCluster(s, 0, vec.Vec3{}, vec.Vec3{}, vec.Vec3{Z: 1}, 1, 3)
	if d := math.Abs(sim.KineticEnergy(s, 0)-k0) / k0; d > 1e-12 {
		t.Errorf("kinetic energy changed by %.3g", d)
	}
	aligned := 0
	for i := range s.Len() {
		if s.PosX[i]*s.VelY[i]-s.PosY[i]*s.VelX[i] >= 0 {
			aligned++
		}
	}
	if aligned != n || angularMomentumZ(s) <= math.Abs(l0) {
		t.Errorf("%d of %d stars aligned, L_z %.3g (before %.3g)", aligned, n, angularMomentumZ(s), l0)
	}
}
