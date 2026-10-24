package perf

import (
	"runtime"
	"strings"
	"testing"
)

func TestProbeDescribesThisMachine(t *testing.T) {
	h := Probe()
	if h.LogicalCPUs != runtime.NumCPU() || h.CPU == "" {
		t.Errorf("probe %+v", h)
	}
	if runtime.GOOS == "darwin" && (h.MemoryBytes < 1<<30 || h.MemoryBytes > 1<<45) {
		t.Errorf("memory %d bytes is implausible", h.MemoryBytes)
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" && (h.PerfCores == 0 || !strings.Contains(h.CPU, "Apple")) {
		t.Errorf("Apple silicon not recognized: %+v", h)
	}
	t.Log(h.String())
	if h.Key() == "" || h.String() == "" {
		t.Error("empty description")
	}
}
