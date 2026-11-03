package app

import (
	"math"
	"time"

	"galaxy/internal/scene"
	"galaxy/internal/sim"
)

// barEvery is the wall time between two bar measurements (O(N log N) each, T-090).
const barEvery = 500 * time.Millisecond

// barMeter follows the bar of the galaxy in a galaxy scene: its strength A2 and its pattern speed
// from the change of its phase between two measurements.
type barMeter struct {
	at           time.Time // wall time of the last measurement
	time, phase  float64   // simulation time and bar phase then
	a2, omega    float64
	measured, ok bool // a measurement exists; omega is valid
}

// observe measures the bar in sn if the scene shows galaxies and enough time has passed.
func (b *barMeter) observe(sn *sim.Snapshot, sp *scene.Spec) {
	if sp == nil || sp.Group != "galaxies" {
		*b = barMeter{}
		return
	}
	now := time.Now()
	if b.measured && now.Sub(b.at) < barEvery {
		return
	}
	m := sn.Bar()
	if b.measured && sn.Time > b.time {
		// The phase is defined modulo π; between two measurements the bar turns by less than π/2.
		d := math.Remainder(m.Phase-b.phase, math.Pi)
		b.omega, b.ok = d/(sn.Time-b.time), true
	} else {
		b.ok = false
	}
	b.at, b.time, b.phase, b.a2, b.measured = now, sn.Time, m.Phase, m.A2, true
}

// BarState is the bar of the galaxy as the inspector shows it.
type BarState struct {
	Shown bool    // the scene shows galaxies
	A2    float64 // m = 2 amplitude of the stars (≳ 0.2: a bar)
	Omega float64 // pattern speed: km/s/kpc with Units, else per time unit; NaN unknown
	Units bool
}

// state converts the measurement for the scene sp.
func (b *barMeter) state(sp *scene.Spec) BarState {
	if !b.measured {
		return BarState{}
	}
	s := BarState{Shown: true, A2: b.a2, Omega: math.NaN()}
	if b.ok {
		s.Omega = math.Abs(b.omega)
		if kms, pc, ok := sp.Units(); ok {
			s.Omega, s.Units = s.Omega*kms/(pc/1000), true // 1/time unit → km/s per kpc
		}
	}
	return s
}
