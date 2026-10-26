package cosmo

import (
	"math"
	"math/cmplx"
	"math/rand/v2"
	"testing"
)

func TestFFTMatchesDFT(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	x := make([]complex128, 16)
	for i := range x {
		x[i] = complex(rng.NormFloat64(), rng.NormFloat64())
	}
	y := append([]complex128(nil), x...)
	fft(y, false)
	for k := range x {
		var s complex128
		for j := range x {
			s += x[j] * cmplx.Exp(complex(0, -2*math.Pi*float64(j*k)/16))
		}
		if cmplx.Abs(s-y[k]) > 1e-9 {
			t.Fatalf("bin %d: %v, want %v", k, y[k], s)
		}
	}
	fft(y, true)
	for i := range x {
		if cmplx.Abs(y[i]/16-x[i]) > 1e-12 {
			t.Fatalf("round trip %d: %v, want %v", i, y[i]/16, x[i])
		}
	}
}

func TestGrowthFactor(t *testing.T) {
	eds := Cosmology{H: 0.7, OmegaM: 1}
	for _, a := range []float64{0.1, 0.5, 0.8} {
		if d := eds.Growth(a); math.Abs(d-a) > 1e-3 {
			t.Errorf("Einstein–de Sitter D(%v) = %v, want a", a, d)
		}
	}
	// ΛCDM grows more slowly late: D(z = 1) ≈ 0.61 for Ω_m = 0.31.
	if d := Planck2018.Growth(0.5); d < 0.59 || d > 0.63 {
		t.Errorf("ΛCDM D(z=1) = %v", d)
	}
	if f := Planck2018.GrowthRate(1); math.Abs(f-0.525) > 0.01 {
		t.Errorf("f(z=0) = %v, want ≈ 0.525", f)
	}
}

func TestPowerIsNormalizedToSigma8(t *testing.T) {
	p := NewPower(Planck2018)
	if s := p.Sigma(8); math.Abs(s-0.81) > 1e-6 {
		t.Errorf("σ₈ = %v", s)
	}
	// The spectrum turns over near k ≈ 0.02 h/Mpc.
	if !(p.At(0.02) > p.At(0.002) && p.At(0.02) > p.At(0.2)) {
		t.Error("no turnover near k = 0.02 h/Mpc")
	}
}

// TestZeldovichVarianceFollowsSpectrum: the variance of the realized density equals the sum of
// P(k)/V over the grid modes, and the displacements are the gradient field with −∇·Ψ = δ.
func TestZeldovichVarianceFollowsSpectrum(t *testing.T) {
	p := NewPower(Planck2018)
	const g, l = 32, 100.0
	f := Zeldovich(p, g, l, 3)
	var mean, v float64
	for _, d := range f.Delta {
		mean += d
		v += d * d
	}
	n := float64(len(f.Delta))
	mean /= n
	v = v/n - mean*mean
	kf := 2 * math.Pi / l
	var want float64
	for i := range g {
		for j := range g {
			for k := range g {
				if i == g/2 || j == g/2 || k == g/2 || (i == 0 && j == 0 && k == 0) {
					continue
				}
				fr := func(x int) float64 {
					if x > g/2 {
						x -= g
					}
					return float64(x) * kf
				}
				want += p.At(math.Sqrt(fr(i)*fr(i)+fr(j)*fr(j)+fr(k)*fr(k))) / (l * l * l)
			}
		}
	}
	if math.Abs(v/want-1) > 0.05 {
		t.Errorf("variance %.4g, want %.4g", v, want)
	}
	if math.Abs(mean) > 1e-9 {
		t.Errorf("mean density contrast %v", mean)
	}
	// −∇·Ψ = δ, checked with central differences at a few points (smooth on the grid scale only
	// approximately, so compare the correlation over the whole grid).
	var dot, nd, nn float64
	h := l / g
	at := func(a, i, j, k int) float64 { return f.Disp[a][((i+g)%g*g+(j+g)%g)*g+(k+g)%g] }
	for i := range g {
		for j := range g {
			for k := range g {
				div := (at(0, i+1, j, k)-at(0, i-1, j, k))/(2*h) + (at(1, i, j+1, k)-at(1, i, j-1, k))/(2*h) +
					(at(2, i, j, k+1)-at(2, i, j, k-1))/(2*h)
				d := f.Delta[(i*g+j)*g+k]
				dot += -div * d
				nd += div * div
				nn += d * d
			}
		}
	}
	if corr := dot / math.Sqrt(nd*nn); corr < 0.8 {
		t.Errorf("correlation of −∇·Ψ with δ: %v", corr)
	}
}
