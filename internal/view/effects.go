package view

import (
	"fmt"
	"strings"
)

// Effects selects the optional render passes (M4).
type Effects struct {
	Glow    bool // bloom (T-016)
	Doppler bool // relativistic color shift (T-017)
	Lensing bool // gravitational lens around black holes (T-018)
	Trails  bool // fading star trails (T-019)
	Dark    bool // draw dark matter as a faint violet haze
	Gas     bool // draw gas as a faint glow
	Dust    bool // dust in the gas absorbs and reddens the light behind it
}

// ParseEffects reads a comma-separated list such as "glow,lens,doppler,trails".
func ParseEffects(list string) (Effects, error) {
	var fx Effects
	for _, name := range strings.Split(list, ",") {
		switch strings.TrimSpace(name) {
		case "":
		case "glow":
			fx.Glow = true
		case "doppler":
			fx.Doppler = true
		case "lens":
			fx.Lensing = true
		case "trails":
			fx.Trails = true
		case "dark":
			fx.Dark = true
		case "gas":
			fx.Gas = true
		case "dust":
			fx.Dust = true
		default:
			return fx, fmt.Errorf("unknown effect %q (want glow, doppler, lens, trails, dark, gas, dust)", name)
		}
	}
	return fx, nil
}
