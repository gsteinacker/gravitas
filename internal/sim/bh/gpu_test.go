package bh

import (
	"fmt"
	"testing"
)

func TestGPUWalkMatchesCPU(t *testing.T) {
	if err := GPUAvailable(); err != nil {
		t.Skip(err)
	}
	s, p := galaxy(20000)
	want := accelerations(s, &Solver{}, p)
	gpu := &Solver{GPU: true}
	got := accelerations(s, gpu, p)
	med, mx := relErrors(starsOnly(s, got), starsOnly(s, want))
	t.Logf("GPU vs CPU walk: median %.2g, max %.2g (float32), GPU time %.2f ms", med, mx, gpu.gpu.lastSeconds*1000)
	if med > 1e-5 || mx > 1e-3 {
		t.Errorf("GPU walk deviates from CPU walk: median %.3g, max %.3g", med, mx)
	}
}

func BenchmarkGPUvsCPU(b *testing.B) {
	if err := GPUAvailable(); err != nil {
		b.Skip(err)
	}
	for _, n := range []int{100_000, 1_000_000} {
		s, p := galaxy(n)
		for _, gpu := range []bool{false, true} {
			name := "CPU"
			if gpu {
				name = "GPU"
			}
			b.Run(fmt.Sprintf("N=%d/%s", n, name), func(b *testing.B) {
				solver := &Solver{GPU: gpu}
				solver.Accelerations(s, p)
				var kernel float64
				for b.Loop() {
					solver.Accelerations(s, p)
					kernel += solver.gpu.lastSeconds
				}
				if gpu {
					b.ReportMetric(kernel/float64(b.N)*1000, "kernel-ms")
				}
			})
		}
	}
}
