package perf

import (
	"encoding/binary"
	"syscall"
)

func probeOS(h *Hardware) {
	if s, err := syscall.Sysctl("machdep.cpu.brand_string"); err == nil {
		h.CPU = s
	}
	if n, err := syscall.SysctlUint32("hw.perflevel0.physicalcpu"); err == nil {
		h.PerfCores = int(n)
		if e, err := syscall.SysctlUint32("hw.perflevel1.physicalcpu"); err == nil {
			h.EffCores = int(e)
		}
	}
	// syscall.Sysctl drops a trailing zero byte; pad the little-endian uint64 back to 8 bytes.
	if s, err := syscall.Sysctl("hw.memsize"); err == nil && len(s) <= 8 {
		var b [8]byte
		copy(b[:], s)
		h.MemoryBytes = binary.LittleEndian.Uint64(b[:])
	}
}
