// Package sph implements the hydrodynamics of gas bodies with Smoothed Particle Hydrodynamics
// (ADR-0007): cubic spline kernel, adaptive smoothing lengths, the symmetric variable-h formulation
// (exact momentum conservation), Monaghan artificial viscosity and the internal energy equation.
package sph

import "math"

// W is the cubic spline kernel (Monaghan & Lattanzio 1985) with support 2h.
func W(r, h float64) float64 {
	q := r / h
	sigma := 1 / (math.Pi * h * h * h)
	switch {
	case q < 1:
		return sigma * (1 - 1.5*q*q + 0.75*q*q*q)
	case q < 2:
		t := 2 - q
		return sigma * 0.25 * t * t * t
	default:
		return 0
	}
}

// DW is dW/dr.
func DW(r, h float64) float64 {
	q := r / h
	sigma := 1 / (math.Pi * h * h * h * h)
	switch {
	case q < 1:
		return sigma * (-3*q + 2.25*q*q)
	case q < 2:
		t := 2 - q
		return -sigma * 0.75 * t * t
	default:
		return 0
	}
}
