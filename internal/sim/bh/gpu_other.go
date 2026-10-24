//go:build !(darwin && cgo)

package bh

import "errors"

// GPUAvailable reports whether the Metal tree walk can be used. It needs macOS and cgo.
func GPUAvailable() error { return errors.New("GPU tree walk needs macOS with cgo") }

type gpuBuffers struct{ lastSeconds float64 }

func (b *Solver) walkGPU(float64) { panic("unreachable: GPUAvailable reported an error") }

// GPUName is the name of the Metal device, or "" if the GPU walk is unavailable.
func GPUName() string { return "" }

// LastGPUSeconds is the kernel time of the last GPU walk, 0 before the first one.
func (b *Solver) LastGPUSeconds() float64 { return 0 }
