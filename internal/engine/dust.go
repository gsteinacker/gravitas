package engine

import (
	"math"
	"slices"

	"galaxy/internal/sim"
	"galaxy/internal/view"
)

// Dust absorption of stars and emission (T-035, T-040; ported from render/dust.go for T-053). Every gas body deposits the optical depth of
// its neutral column, τ = κ m / A (A its footprint in world units at its depth), onto a screen map,
// together with the τ-weighted first and second moments of its depth. A star at depth d sees the
// part of the column in front of it, τ_front = τ · Φ((d − μ)/σ), with μ and σ the mean and spread of
// the dust depth along that line of sight (a Gaussian approximation of the dust layer), and is
// attenuated by exp(−τ_front k) per color channel. Stars in the middle of an edge-on disk are
// therefore half extinguished, and a dust lane in front of a bulge is dark.
//
// Bodies deposit with cloud-in-cell weights into the level of a pyramid whose cells match their
// footprint, so large bodies spread their column without per-pixel work; lookups sum all levels.

const (
	dustBaseW, dustBaseH = 320, 200 // finest level
	dustLevels           = 5        // cells 1, 2, 4, 8, 16 times the finest
)

// Extinction relative to V per channel (roughly Cardelli et al. 1989 for R, V, B).
var dustK = [3]float64{0.75, 1.0, 1.3}

type dustLevel struct {
	w, h        int
	cell        float64 // cell size in pixels
	tau, d1, d2 []float64
}

type dustMap struct {
	levels [dustLevels]dustLevel
}

func (dm *dustMap) reset(screenW float64) {
	for l := range dm.levels {
		lv := &dm.levels[l]
		lv.w, lv.h = dustBaseW>>l, dustBaseH>>l
		lv.cell = screenW / float64(dustBaseW) * float64(int(1)<<l)
		n := (lv.w + 1) * (lv.h + 1)
		lv.tau, lv.d1, lv.d2 = resizeFloats(lv.tau, n), resizeFloats(lv.d1, n), resizeFloats(lv.d2, n)
		clear(lv.tau)
		clear(lv.d1)
		clear(lv.d2)
	}
}

// deposit adds a body of optical depth-area product τA (τ times its footprint in pixels²) at screen
// (sx, sy) with footprint diameter size (pixels) and depth.
func (dm *dustMap) deposit(sx, sy, size, tauArea, depth float64) {
	l := 0
	for l < dustLevels-1 && dm.levels[l].cell < 0.5*size {
		l++
	}
	lv := &dm.levels[l]
	fx, fy := sx/lv.cell-0.5, sy/lv.cell-0.5
	cx, cy := int(math.Floor(fx)), int(math.Floor(fy))
	wx, wy := fx-float64(cx), fy-float64(cy)
	tau := tauArea / (lv.cell * lv.cell)
	for k, w := range [4]float64{(1 - wx) * (1 - wy), wx * (1 - wy), (1 - wx) * wy, wx * wy} {
		x, y := cx+k&1, cy+k>>1
		if x < 0 || y < 0 || x > lv.w || y > lv.h || w == 0 {
			continue
		}
		i := y*(lv.w+1) + x
		t := w * tau
		lv.tau[i] += t
		lv.d1[i] += t * depth
		lv.d2[i] += t * depth * depth
	}
}

// sample returns the total optical depth and its depth moments at screen (sx, sy).
func (dm *dustMap) sample(sx, sy float64) (tau, d1, d2 float64) {
	for l := range dm.levels {
		lv := &dm.levels[l]
		fx, fy := sx/lv.cell-0.5, sy/lv.cell-0.5
		cx, cy := int(math.Floor(fx)), int(math.Floor(fy))
		wx, wy := fx-float64(cx), fy-float64(cy)
		for k, w := range [4]float64{(1 - wx) * (1 - wy), wx * (1 - wy), (1 - wx) * wy, wx * wy} {
			x, y := cx+k&1, cy+k>>1
			if x < 0 || y < 0 || x > lv.w || y > lv.h {
				continue
			}
			i := y*(lv.w+1) + x
			tau, d1, d2 = tau+w*lv.tau[i], d1+w*lv.d1[i], d2+w*lv.d2[i]
		}
	}
	return tau, d1, d2
}

// front returns the optical depth (in V) between the camera and depth at screen (sx, sy).
func (dm *dustMap) front(sx, sy, depth float64) float64 {
	tau, d1, d2 := dm.sample(sx, sy)
	if tau <= 1e-6 {
		return 0
	}
	mu := d1 / tau
	sigma := math.Sqrt(math.Max(d2/tau-mu*mu, 0))
	sigma = math.Max(sigma, 1e-3*mu)
	return tau * 0.5 * math.Erfc(-(depth-mu)/(math.Sqrt2*sigma))
}

// transmission returns the attenuation per color channel for optical depth tauV.
func transmission(tauV float64) (r, g, b float32) {
	if tauV <= 0 {
		return 1, 1, 1
	}
	return float32(math.Exp(-tauV * dustK[0])), float32(math.Exp(-tauV * dustK[1])), float32(math.Exp(-tauV * dustK[2]))
}

func resizeFloats(s []float64, n int) []float64 {
	if cap(s) < n {
		return make([]float64, n)
	}
	return s[:n]
}

// Dust constants of render/stars.go and render/post.go.
const (
	dustPercentile = 0.9
	dustAutoTau    = 1.5
	// DustStrength is the blue optical depth of a fully dense pixel of the dust image (the
	// background attenuation drawn by the front end).
	DustStrength = 2.5
)

// buildDust deposits the neutral gas into the dust map and returns the opacity κ (V-band optical
// depth per unit column): the scene's physical one, or an automatic one that makes the densest
// dustPercentile of the covered screen reach optical depth dustAutoTau.
func (l *looker) buildDust(bodies []float32, n int, vw *view.View, w, h, focal, ps, sceneKappa float64) float64 {
	kappa := sceneKappa
	if kappa == 0 {
		kappa = l.dustNorm
		if kappa == 0 {
			kappa = 1
		}
	}
	l.dust.reset(w)
	for i := range n {
		b := bodies[i*FloatsPerBody : i*FloatsPerBody+FloatsPerBody : i*FloatsPerBody+FloatsPerBody]
		if sim.Kind(b[7]+0.5) != sim.Gas {
			continue
		}
		sx, sy, depth, ok := vw.Project(float64(b[0]), float64(b[1]), float64(b[2]))
		if !ok {
			continue
		}
		size := math.Max(2*ps, math.Min(80*ps, 2*float64(b[10])*focal/depth))
		if sx < -size || sy < -size || sx > w+size || sy > h+size {
			continue
		}
		// τ · footprint in pixels² = κ m (focal/depth)² for the neutral mass (dust survives there).
		tauArea := kappa * float64(b[3]) * (1 - float64(b[9])) * (focal / depth) * (focal / depth)
		l.dust.deposit(sx, sy, size, tauArea, depth)
	}
	if sceneKappa > 0 {
		return kappa
	}
	lit := l.sortBuf[:0]
	for cy := range gridH {
		for cx := range gridW {
			x, y := (float64(cx)+0.5)*w/gridW, (float64(cy)+0.5)*h/gridH
			if tau, _, _ := l.dust.sample(x, y); tau > 0 {
				lit = append(lit, float32(tau))
			}
		}
	}
	l.sortBuf = lit
	if len(lit) > 0 {
		slices.Sort(lit)
		p := float64(lit[int(dustPercentile*float64(len(lit)-1))])
		target := kappa * dustAutoTau / p
		if l.dustNorm == 0 {
			l.dustNorm = target
		} else {
			l.dustNorm += smoothing * (target - l.dustNorm)
		}
	}
	return kappa
}
