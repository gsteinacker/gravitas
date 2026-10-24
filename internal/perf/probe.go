// Package perf adapts the app to the machine: it describes the hardware, measures how fast the
// force calculation runs (calibration), derives performance profiles and, at run time, governs
// quality so the app stays fluid (T-046, T-047). It is headless and never imports Ebitengine.
package perf

import (
	"fmt"
	"runtime"

	"galaxy/internal/sim/bh"
)

// Hardware describes the machine.
type Hardware struct {
	CPU          string // e.g. "Apple M4 Pro"
	PerfCores    int    // performance cores, 0 if unknown
	EffCores     int    // efficiency cores, 0 if unknown
	LogicalCPUs  int
	MemoryBytes  uint64
	GPU          string // Metal device, "" without GPU walk
	Architecture string
}

// Probe describes this machine.
func Probe() Hardware {
	h := Hardware{LogicalCPUs: runtime.NumCPU(), Architecture: runtime.GOARCH, GPU: bh.GPUName()}
	probeOS(&h)
	if h.CPU == "" {
		h.CPU = runtime.GOARCH
	}
	return h
}

// Key identifies the machine for cached calibration results.
func (h Hardware) Key() string {
	return fmt.Sprintf("%s|%d|%d|%d|%s", h.CPU, h.PerfCores, h.EffCores, h.LogicalCPUs, h.GPU)
}

func (h Hardware) String() string {
	cores := fmt.Sprintf("%d Threads", h.LogicalCPUs)
	if h.PerfCores > 0 {
		cores = fmt.Sprintf("%dP+%dE", h.PerfCores, h.EffCores)
	}
	s := h.CPU + " · " + cores
	if h.MemoryBytes > 0 {
		s += fmt.Sprintf(" · %d GB", (h.MemoryBytes+1<<29)>>30)
	}
	if h.GPU != "" && h.GPU != h.CPU {
		s += " · " + h.GPU
	}
	return s
}
