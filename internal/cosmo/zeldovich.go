package cosmo

import (
	"math"
	"math/rand/v2"
)

// Field is a Zel'dovich realization on a g³ grid in a periodic box of side L (Mpc/h).
type Field struct {
	G     int
	L     float64
	Disp  [3][]float64 // displacement Ψ today (D = 1), Mpc/h, per grid point (index (i·g + j)·g + k)
	Delta []float64    // linear density contrast today on the grid
}

// Zeldovich draws a Gaussian random density field with spectrum p in a box of side l and returns
// its displacement field Ψ(k) = i k/k² δ(k) (Zel'dovich 1970). Positions at growth D are
// q + D Ψ(q); comoving peculiar velocities are a H f D Ψ.
func Zeldovich(p Power, g int, l float64, seed uint64) *Field {
	n := g * g * g
	rng := rand.New(rand.NewPCG(seed, 0x2e1d))
	w := make([]complex128, n)
	for i := range w { // white noise in real space: its transform is Hermitian, so δ(x) is real
		w[i] = complex(rng.NormFloat64(), 0)
	}
	fft3(w, g, false)
	vol := l * l * l
	kf := 2 * math.Pi / l
	delta := make([]complex128, n)
	disp := [3][]complex128{make([]complex128, n), make([]complex128, n), make([]complex128, n)}
	freq := func(i int) float64 {
		if i > g/2 {
			i -= g
		}
		return float64(i) * kf
	}
	for i := range g {
		for j := range g {
			for k := range g {
				idx := (i*g+j)*g + k
				kx, ky, kz := freq(i), freq(j), freq(k)
				k2 := kx*kx + ky*ky + kz*kz
				if k2 == 0 || i == g/2 || j == g/2 || k == g/2 { // mean and Nyquist planes
					continue
				}
				// ⟨|W|²⟩ = n, so c = W √(P/(V n)) has ⟨|c|²⟩ = P/V; the unnormalized inverse
				// transform Σ c e^{ikx} then has the continuum normalization δ(x) = Σ δ̃/V e^{ikx}.
				c := w[idx] * complex(math.Sqrt(p.At(math.Sqrt(k2))/(vol*float64(n))), 0)
				delta[idx] = c
				for a, ka := range [3]float64{kx, ky, kz} {
					disp[a][idx] = complex(0, ka/k2) * c
				}
			}
		}
	}
	f := &Field{G: g, L: l, Delta: make([]float64, n)}
	fft3(delta, g, true)
	for i := range n {
		f.Delta[i] = real(delta[i])
	}
	for a := range 3 {
		fft3(disp[a], g, true)
		f.Disp[a] = make([]float64, n)
		for i := range n {
			f.Disp[a][i] = real(disp[a][i])
		}
	}
	return f
}
