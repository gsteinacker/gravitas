package app

import "galaxy/internal/scene"

// Star formation and supernova rates for the inspector (T-080). Both are averaged over a window
// of simulation time, like the UV light that measures star formation rates in real galaxies
// (about 100 Myr), because single events are rare: one star particle is 10⁵–10⁶ M☉.

// StarRates are the rates of the scene; the counts are in SimState.
type StarRates struct {
	Formation bool    // the scene forms stars from gas
	Units     bool    // the scene has physical units: SFR in M☉/yr, SNRate per century
	SFR       float64 // star formation rate (M☉/yr, or mass per time unit)
	SNRate    float64 // supernovae per century (real ones), or exploded populations per time unit
	GasShare  float64 // gas mass / (gas + star mass)
	Window    float64 // averaging window in years (Units) or time units
}

const (
	rateWindowYears = 100e6
	rateWindowUnits = 5.0
	rateSamples     = 200 // samples per window
	// snPerMsun is the number of core-collapse supernovae per formed solar mass: stars above
	// 8 M☉ for a Kroupa IMF.
	snPerMsun = 0.01
)

type rateSample struct{ t, formed, exploded float64 }

// rateMeter keeps the cumulative formed and exploded mass over the last window.
type rateMeter struct {
	samples []rateSample
}

// add records the totals at time t; a time before the last sample (restart, load) starts over.
func (r *rateMeter) add(t, formed, exploded, window float64) {
	if n := len(r.samples); n > 0 {
		last := r.samples[n-1]
		if t < last.t || formed < last.formed || exploded < last.exploded {
			r.samples = r.samples[:0]
		} else if t-last.t < window/rateSamples {
			return
		}
	}
	r.samples = append(r.samples, rateSample{t, formed, exploded})
	// Keep one sample at or before the start of the window.
	drop := 0
	for drop+1 < len(r.samples) && r.samples[drop+1].t <= t-window {
		drop++
	}
	if drop > 0 {
		r.samples = append(r.samples[:0], r.samples[drop:]...)
	}
}

// rates returns the mass formed and exploded per time unit over the window (0 before the
// first interval).
func (r *rateMeter) rates() (formed, exploded float64) {
	if len(r.samples) < 2 {
		return 0, 0
	}
	a, b := r.samples[0], r.samples[len(r.samples)-1]
	dt := b.t - a.t
	if dt <= 0 {
		return 0, 0
	}
	return (b.formed - a.formed) / dt, (b.exploded - a.exploded) / dt
}

// starRates converts the measured rates into the units of the scene.
func starRates(sp *scene.Spec, r *rateMeter, gasShare float64) StarRates {
	out := StarRates{GasShare: gasShare, Window: rateWindowUnits}
	if sp == nil {
		return out
	}
	out.Formation = sp.Params.Gas.SFDensity > 0
	formed, exploded := r.rates()
	out.SFR, out.SNRate = formed, exploded
	if msun, ok := sp.MassUnit(); ok && sp.YearsPer > 0 {
		out.Units, out.Window = true, rateWindowYears
		out.SFR = formed * msun / sp.YearsPer
		out.SNRate = exploded * msun * snPerMsun / sp.YearsPer * 100
	}
	return out
}

// rateWindow is the averaging window in simulation time units.
func rateWindow(sp *scene.Spec) float64 {
	if sp != nil && sp.YearsPer > 0 {
		return rateWindowYears / sp.YearsPer
	}
	return rateWindowUnits
}
