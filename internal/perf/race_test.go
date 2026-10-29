//go:build race

package perf

// raceEnabled: the race detector slows everything down, so time limits do not apply.
const raceEnabled = true
