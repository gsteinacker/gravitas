package sph

import (
	"math"
	"sync"

	"galaxy/internal/sim"
)

// Solver implements sim.Hydro. All buffers are reused between calls.
type Solver struct {
	gas   []int32
	tree  tree
	P, C  []float64 // pressure and sound speed per body (gas only)
	todo  []int32   // bodies whose smoothing length is still iterating
	again []bool
}

const (
	maxDensityIterations = 6
	hTolerance           = 0.05 // relative deviation of h from its target that ends the iteration
	betaFactor           = 2    // β = 2α (Monaghan)
	hMinFactor           = 1e-3
)

var stackPool = sync.Pool{New: func() any { s := make([]int32, 0, 64); return &s }}

func defaults(g sim.GasParams) (gamma, ngb, alpha float64) {
	gamma, ngb, alpha = g.Gamma, g.Neighbors, g.Alpha
	if gamma == 0 {
		gamma = 5.0 / 3
	}
	if ngb == 0 {
		ngb = 40
	}
	if alpha == 0 {
		alpha = 1
	}
	return gamma, ngb, alpha
}

// Forces implements sim.Hydro.
func (sv *Solver) Forces(s *sim.State, p *sim.Params, dudt, divv []float64) {
	sv.gas = sv.gas[:0]
	for i := range s.Len() {
		if s.Kind[i] == sim.Gas && s.Alive[i] {
			sv.gas = append(sv.gas, int32(i))
		}
	}
	clear(dudt)
	clear(divv)
	if len(sv.gas) < 2 {
		return
	}
	gamma, ngb, alpha := defaults(p.Gas)
	x, y, z, m, h := s.PosX, s.PosY, s.PosZ, s.Mass, s.H
	sv.initH(s, ngb)
	sv.tree.build(sv.gas, x, y, z, h)

	// Density and smoothing length: h = ½ (3 N m / 4π ρ)^⅓ for N neighbors inside 2h. Each body
	// iterates on its own until h is within hTolerance of that target (ρ_i depends only on h_i);
	// since h carries over between steps, most bodies are done after the first pass.
	sv.todo = append(sv.todo[:0], sv.tree.order...)
	for it := 0; len(sv.todo) > 0; it++ {
		last := it == maxDensityIterations-1
		sv.again = resizeBool(sv.again, len(sv.todo))
		sim.ParallelFor(len(sv.todo), 256, 0, func(lo, hi int) {
			sp := stackPool.Get().(*[]int32)
			for k, i := range sv.todo[lo:hi] {
				hi := h[i]
				var rho float64
				*sp = sv.tree.each(x[i], y[i], z[i], 2*hi, false, *sp, func(j int32, r2 float64) {
					rho += m[j] * W(math.Sqrt(r2), hi)
				})
				s.Rho[i] = rho
				target := 0.5 * math.Cbrt(3*ngb*m[i]/(4*math.Pi*rho))
				redo := !last && math.Abs(target-hi) > hTolerance*hi
				if redo {
					h[i] = math.Max(0.5*hi, math.Min(2*hi, target)) // damped update
				}
				sv.again[lo+k] = redo
			}
			stackPool.Put(sp)
		})
		j := 0
		for k, i := range sv.todo {
			if sv.again[k] {
				sv.todo[j] = i
				j++
			}
		}
		sv.todo = sv.todo[:j]
		if j > 0 {
			sv.tree.refitH(h)
		}
	}

	if cap(sv.P) < s.Len() {
		sv.P, sv.C = make([]float64, s.Len()), make([]float64, s.Len())
	}
	P, C := sv.P[:s.Len()], sv.C[:s.Len()]
	for _, i := range sv.gas {
		u := math.Max(s.U[i], 0)
		P[i] = (gamma - 1) * s.Rho[i] * u
		C[i] = math.Sqrt(gamma * (gamma - 1) * u)
	}

	// Forces, energy equation and velocity divergence (symmetric variable-h formulation).
	sim.ParallelFor(len(sv.gas), 256, 0, func(lo, hi int) {
		sp := stackPool.Get().(*[]int32)
		for _, i := range sv.tree.order[lo:hi] {
			hi, rhoi, pi := h[i], s.Rho[i], P[i]
			fi := pi / (rhoi * rhoi)
			var ax, ay, az, du, dv float64
			*sp = sv.tree.each(x[i], y[i], z[i], 2*hi, true, *sp, func(j int32, r2 float64) {
				if j == i || r2 == 0 {
					return
				}
				r := math.Sqrt(r2)
				ex, ey, ez := (x[i]-x[j])/r, (y[i]-y[j])/r, (z[i]-z[j])/r
				vx, vy, vz := s.VelX[i]-s.VelX[j], s.VelY[i]-s.VelY[j], s.VelZ[i]-s.VelZ[j]
				vr := vx*ex + vy*ey + vz*ez // v_ij · r̂_ij
				dwi, dwj := DW(r, hi), DW(r, h[j])
				dwm := 0.5 * (dwi + dwj)
				fj := P[j] / (s.Rho[j] * s.Rho[j])
				var visc float64
				if vr < 0 { // approaching: Monaghan viscosity
					hm := 0.5 * (hi + h[j])
					mu := hm * vr * r / (r2 + 0.01*hm*hm)
					visc = (-alpha*0.5*(C[i]+C[j])*mu + betaFactor*alpha*mu*mu) / (0.5 * (rhoi + s.Rho[j]))
				}
				f := -m[j] * (fi*dwi + fj*dwj + visc*dwm)
				ax, ay, az = ax+f*ex, ay+f*ey, az+f*ez
				du += m[j] * (fi*dwi + 0.5*visc*dwm) * vr
				dv -= m[j] * dwi * vr
			})
			s.AccX[i] += ax
			s.AccY[i] += ay
			s.AccZ[i] += az
			dudt[i] = du
			divv[i] = dv / rhoi
		}
		stackPool.Put(sp)
	})
}

// initH gives bodies without a smoothing length a first guess from the mean density of their box.
func (sv *Solver) initH(s *sim.State, ngb float64) {
	var missing bool
	for _, i := range sv.gas {
		if s.H[i] <= 0 {
			missing = true
			break
		}
	}
	if !missing {
		return
	}
	minX, minY, minZ := math.Inf(1), math.Inf(1), math.Inf(1)
	maxX, maxY, maxZ := math.Inf(-1), math.Inf(-1), math.Inf(-1)
	for _, i := range sv.gas {
		minX, maxX = math.Min(minX, s.PosX[i]), math.Max(maxX, s.PosX[i])
		minY, maxY = math.Min(minY, s.PosY[i]), math.Max(maxY, s.PosY[i])
		minZ, maxZ = math.Min(minZ, s.PosZ[i]), math.Max(maxZ, s.PosZ[i])
	}
	vol := math.Max((maxX-minX)*(maxY-minY)*(maxZ-minZ), 1e-30)
	guess := 0.5 * math.Cbrt(3*ngb*vol/(4*math.Pi*float64(len(sv.gas))))
	for _, i := range sv.gas {
		if s.H[i] <= 0 {
			s.H[i] = guess
		}
	}
	_ = hMinFactor
}

func resizeBool(s []bool, n int) []bool {
	if cap(s) < n {
		return make([]bool, n)
	}
	return s[:n]
}
