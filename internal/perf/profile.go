package perf

import (
	"time"

	"galaxy/internal/i18n"
	"galaxy/internal/prefs"
	"galaxy/internal/scene"
)

// Profile is a performance level the user can choose. The automatic profile aims at 30 steps/s
// and lets the governor adapt the quality; fixed profiles trade stars for speed.
type Profile struct {
	ID             string  // as stored in prefs.Profile; "" = automatic
	TargetSPS      float64 // steps per second the star count is chosen for
	MaxRenderScale float64 // upper bound of render pixels per window point (2 = full Retina)
}

// Profiles in the order of the settings window.
var Profiles = []Profile{
	{ID: "", TargetSPS: 30, MaxRenderScale: 2},
	{ID: "low", TargetSPS: 60, MaxRenderScale: 1},
	{ID: "medium", TargetSPS: 45, MaxRenderScale: 1.5},
	{ID: "high", TargetSPS: 30, MaxRenderScale: 2},
	{ID: "max", TargetSPS: 12, MaxRenderScale: 2},
}

// ProfileByID returns the profile, or the automatic one for unknown IDs.
func ProfileByID(id string) Profile {
	for _, p := range Profiles {
		if p.ID == id {
			return p
		}
	}
	return Profiles[0]
}

// Name is the label in the user's language.
func (p Profile) Name() string {
	switch p.ID {
	case "low":
		return i18n.T("Niedrig", "Low")
	case "medium":
		return i18n.T("Mittel", "Medium")
	case "high":
		return i18n.T("Hoch", "High")
	case "max":
		return i18n.T("Maximal", "Maximum")
	}
	return i18n.T("Automatisch", "Automatic")
}

// Stars is the star count of a gas-free scene for this profile on a calibrated machine; fallback
// if the machine is not calibrated.
func (p Profile) Stars(c prefs.Calibration, fallback int) int { return p.SceneStars(c, nil, fallback) }

// SceneStars is Stars for a scene, taking its gas into account; sp may be nil.
func (p Profile) SceneStars(c prefs.Calibration, sp *scene.Spec, fallback int) int {
	gas := 0.0
	if sp != nil {
		gas = GasFraction(sp)
	}
	if n := StarsForGas(c, time.Duration(float64(time.Second)/p.TargetSPS), gas); n > 0 {
		return n
	}
	return fallback
}
