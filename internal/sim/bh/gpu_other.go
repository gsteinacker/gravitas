//go:build !(darwin && cgo)

package bh

import "errors"

// GPUAvailable reports whether the Metal tree walk can be used. It needs macOS and cgo.
func GPUAvailable() error { return errors.New("GPU tree walk needs macOS with cgo") }

type gpuBuffers struct{ lastSeconds float64 }

func (b *Solver) walkGPU(float64) { panic("unreachable: GPUAvailable reported an error") }
