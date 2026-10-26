package cosmo

import (
	"math"
	"runtime"
	"sync"

	"galaxy/internal/sim"
)

func init() {
	sim.ComovingSolver = func(p sim.Params, n int) sim.Solver { return NewPM(p.Cosmology.Box, n) }
}

// PM is the periodic particle-mesh solver of the comoving mode (T-074): masses are assigned to a
// g³ grid with cloud-in-cell weights, the Poisson equation ∇²φ = 4πG (ρ − ρ̄) is solved by FFT with
// the Green's function −4πG/k² (mean mode removed), and the force −∇φ (spectral gradient ik φ) is
// interpolated back with the same weights (the CIC window deconvolved in k), so the force of a body on itself vanishes and momentum
// is conserved. Forces are smoothed on about two grid cells.
type PM struct {
	G    int
	L    float64
	rho  []complex128
	grad [3][]complex128
	phi  []float64
}

// NewPM chooses a grid of about one cell per body (a power of two, 16–128 per side).
func NewPM(l float64, n int) *PM {
	g := 16
	for g*g*g < n && g < 128 {
		g *= 2
	}
	c := g * g * g
	return &PM{G: g, L: l, rho: make([]complex128, c),
		grad: [3][]complex128{make([]complex128, c), make([]complex128, c), make([]complex128, c)}}
}

// cic returns the 8 grid cells and weights of a position.
func (p *PM) cic(x, y, z float64, f func(idx int, w float64)) {
	g, h := p.G, p.L/float64(p.G)
	gx, gy, gz := (x+p.L/2)/h-0.5, (y+p.L/2)/h-0.5, (z+p.L/2)/h-0.5
	ix, iy, iz := int(math.Floor(gx)), int(math.Floor(gy)), int(math.Floor(gz))
	dx, dy, dz := gx-float64(ix), gy-float64(iy), gz-float64(iz)
	for c := range 8 {
		wx, wy, wz := 1-dx, 1-dy, 1-dz
		jx, jy, jz := ix, iy, iz
		if c&1 != 0 {
			wx, jx = dx, ix+1
		}
		if c&2 != 0 {
			wy, jy = dy, iy+1
		}
		if c&4 != 0 {
			wz, jz = dz, iz+1
		}
		jx, jy, jz = (jx%g+g)%g, (jy%g+g)%g, (jz%g+g)%g
		f((jx*g+jy)*g+jz, wx*wy*wz)
	}
}

// solve deposits the masses and fills the three gradient grids (real space) and φ.
func (p *PM) solve(s *sim.State) {
	g := p.G
	h := p.L / float64(g)
	clear(p.rho)
	var total float64
	for i := range s.Len() {
		m := s.Mass[i]
		total += m
		p.cic(s.PosX[i], s.PosY[i], s.PosZ[i], func(idx int, w float64) { p.rho[idx] += complex(w*m/(h*h*h), 0) })
	}
	fft3Parallel(p.rho, g, false)
	kf := 2 * math.Pi / p.L
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
				if k2 == 0 {
					p.rho[idx] = 0 // the mean density does not pull (ρ − ρ̄)
					for a := range 3 {
						p.grad[a][idx] = 0
					}
					continue
				}
				// Deconvolve the cloud-in-cell window of deposit and interpolation, W = Π sinc²(k_i h/2).
				w := sinc2(kx*h/2) * sinc2(ky*h/2) * sinc2(kz*h/2)
				phi := p.rho[idx] * complex(-4*math.Pi/k2/(w*w), 0)
				p.rho[idx] = phi
				for a, ka := range [3]float64{kx, ky, kz} {
					if (a == 0 && i == g/2) || (a == 1 && j == g/2) || (a == 2 && k == g/2) {
						p.grad[a][idx] = 0 // the Nyquist mode has no defined derivative
					} else {
						p.grad[a][idx] = complex(0, ka) * phi
					}
				}
			}
		}
	}
	n := float64(g * g * g)
	fft3Parallel(p.rho, g, true)
	if cap(p.phi) < len(p.rho) {
		p.phi = make([]float64, len(p.rho))
	}
	p.phi = p.phi[:len(p.rho)]
	for i := range p.rho {
		p.phi[i] = real(p.rho[i]) / n
	}
	for a := range 3 {
		fft3Parallel(p.grad[a], g, true)
	}
}

// Accelerations writes −∇φ at every body (sim.Solver).
func (p *PM) Accelerations(s *sim.State, _ sim.Params) {
	p.solve(s)
	n := float64(p.G * p.G * p.G)
	for i := range s.Len() {
		var ax, ay, az float64
		p.cic(s.PosX[i], s.PosY[i], s.PosZ[i], func(idx int, w float64) {
			ax -= w * real(p.grad[0][idx]) / n
			ay -= w * real(p.grad[1][idx]) / n
			az -= w * real(p.grad[2][idx]) / n
		})
		s.AccX[i], s.AccY[i], s.AccZ[i] = ax, ay, az
	}
}

// Potential returns φ at every body (after Accelerations), for the Layzer–Irvine check.
func (p *PM) Potential(s *sim.State) []float64 {
	out := make([]float64, s.Len())
	for i := range s.Len() {
		p.cic(s.PosX[i], s.PosY[i], s.PosZ[i], func(idx int, w float64) { out[i] += w * p.phi[idx] })
	}
	return out
}

func sinc2(x float64) float64 {
	if x == 0 {
		return 1
	}
	s := math.Sin(x) / x
	return s * s
}

// fft3Parallel is fft3 with the lines of every axis spread over the CPUs.
func fft3Parallel(x []complex128, g int, inverse bool) {
	workers := runtime.GOMAXPROCS(0)
	for axis := range 3 {
		var wg sync.WaitGroup
		for w := range workers {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				line := make([]complex128, g)
				for ab := w; ab < g*g; ab += workers {
					a, b := ab/g, ab%g
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
			}(w)
		}
		wg.Wait()
	}
}
