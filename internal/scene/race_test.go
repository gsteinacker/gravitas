//go:build race

package scene

// raceEnabled: the race detector slows the long physics runs down beyond the test timeout.
const raceEnabled = true
