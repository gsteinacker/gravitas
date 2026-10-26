// Package cosmo builds cosmological initial conditions (T-073): a ΛCDM power spectrum, the linear
// growth factor and Zel'dovich displacements on a periodic grid. Standard library only; headless.
package cosmo

import (
	"math"
	"math/bits"
	"math/cmplx"
)

// fft transforms x in place (length a power of two); inverse uses e^{+i…} and does not divide by n.
func fft(x []complex128, inverse bool) {
	n := len(x)
	shift := 64 - uint(bits.Len(uint(n-1)))
	for i := range n {
		if j := int(bits.Reverse64(uint64(i)) >> shift); j > i {
			x[i], x[j] = x[j], x[i]
		}
	}
	sign := -1.0
	if inverse {
		sign = 1
	}
	for size := 2; size <= n; size <<= 1 {
		w := cmplx.Exp(complex(0, sign*2*math.Pi/float64(size)))
		for start := 0; start < n; start += size {
			wk := complex(1, 0)
			for k := range size / 2 {
				a, b := x[start+k], x[start+k+size/2]*wk
				x[start+k], x[start+k+size/2] = a+b, a-b
				wk *= w
			}
		}
	}
}

// fft3 transforms a g×g×g grid (index (i·g + j)·g + k) in place along all three axes.
func fft3(x []complex128, g int, inverse bool) {
	line := make([]complex128, g)
	for axis := range 3 {
		for a := range g {
			for b := range g {
				idx := func(t int) int {
					switch axis {
					case 0:
						return (t*g+a)*g + b
					case 1:
						return (a*g+t)*g + b
					}
					return (a*g+b)*g + t
				}
				for t := range g {
					line[t] = x[idx(t)]
				}
				fft(line, inverse)
				for t := range g {
					x[idx(t)] = line[t]
				}
			}
		}
	}
}
