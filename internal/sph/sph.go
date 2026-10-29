package sph

import (
	"math"
	"sync"
	"sync/atomic"

	"galaxy/internal/sim"
)

// Solver implements sim.Hydro. All buffers are reused between calls.
type Solver struct {
	gas    []int32
	tree   tree
	P, C   []float64 // pressure and sound speed per body (gas only)
	todo   []int32   // bodies whose smoothing length is still iterating
	again  []bool    // per tree position: iterate again
	leaves []int32   // leaves with evaluated bodies
}

// courant is the Courant factor of the time step condition dt ≤ courant · h / v_sig, with the
// signal velocity v_sig,i = max_j (c_i + c_j − 3 min(0, v_ij · r̂_ij)) (Monaghan 1997).
const courant = 0.3

// Bound implements sim.Hydro: dt[i] ≤ courant · h_i / (2 c_i) for the current internal energy.
func (sv *Solver) Bound(s *sim.State, p *sim.Params, dt []float64) {
	gamma, _, _ := defaults(p.Gas)
	sim.ParallelFor(s.Len(), 4096, 0, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			if s.Kind[i] != sim.Gas || s.U[i] <= 0 {
				continue
			}
			if c := math.Sqrt(gamma * (gamma - 1) * s.U[i]); c > 0 {
				dt[i] = min(dt[i], courant*s.H[i]/(2*c))
			}
		}
	})
}

const (
	maxDensityIterations = 6
	hTolerance           = 0.05 // relative deviation of h from its target that ends the iteration
	betaFactor           = 2    // β = 2α (Monaghan)
	// hMinSoftening bounds h from below by this fraction of the gravitational softening: gas is
	// not resolved below it anyway, and tiny h would force tiny Courant steps (as in GADGET).
	hMinSoftening = 0.25
	// sphGrain is the smallest chunk of the parallel loops. Bodies are in tree order, so a chunk
	// is a region of space; dense regions have many more neighbors per body. Small chunks let
	// the dynamic scheduling spread a dense core over all cores (T-085).
	sphGrain = 16
)

// walk is the scratch of a tree walk: the node stack and the candidate leaves of a group walk.
type walk struct {
	stack []int32
	cand  []int32
}

var walkPool = sync.Pool{New: func() any { return &walk{stack: make([]int32, 0, 64), cand: make([]int32, 0, 64)} }}

// leafGrain is the smallest chunk of leaves (of up to leafSize bodies) in the group walks.
const leafGrain = 2

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
func (sv *Solver) Forces(s *sim.State, p *sim.Params, active []bool, step []float64, out *sim.HydroOut) {
	sv.gas = sv.gas[:0]
	for i := range s.Len() {
		if s.Kind[i] == sim.Gas && s.Alive[i] {
			sv.gas = append(sv.gas, int32(i))
		}
	}
	isActive := func(i int32) bool { return active == nil || active[i] }
	if len(sv.gas) < 2 {
		for _, i := range sv.gas {
			if isActive(i) {
				out.AccX[i], out.AccY[i], out.AccZ[i] = 0, 0, 0
				out.DuDt[i], out.DivV[i], out.Dt[i] = 0, 0, math.Inf(1)
			}
		}
		return
	}
	gamma, ngb, alpha := defaults(p.Gas)
	x, y, z, m, h := s.PosX, s.PosY, s.PosZ, s.Mass, s.H
	sv.initH(s, ngb)
	hMin := hMinSoftening * p.Softening
	sv.tree.build(sv.gas, x, y, z, h)

	// The leaves holding evaluated bodies; group walks go over them.
	sv.leaves = sv.leaves[:0]
	for _, leaf := range sv.tree.leaves {
		nd := &sv.tree.nodes[leaf]
		for _, i := range sv.tree.order[nd.lo:nd.hi] {
			if isActive(i) {
				sv.leaves = append(sv.leaves, leaf)
				break
			}
		}
	}

	// Density and smoothing length: h = ½ (3 N m / 4π ρ)^⅓ for N neighbors inside 2h. Each body
	// iterates on its own until h is within hTolerance of that target (ρ_i depends only on h_i);
	// since h carries over between steps, most bodies are done after the first pass.
	sv.again = resizeBool(sv.again, len(sv.tree.order))
	for it := 0; it == 0 || len(sv.todo) > 0; it++ {
		last := it == maxDensityIterations-1
		// setH stores the density of body i and moves h towards its target; it reports whether
		// h must iterate again.
		setH := func(i int32, rho float64) bool {
			hi := h[i]
			s.Rho[i] = rho
			target := math.Max(hMin, 0.5*math.Cbrt(3*ngb*m[i]/(4*math.Pi*rho)))
			redo := !last && math.Abs(target-hi) > hTolerance*hi
			if redo {
				h[i] = math.Max(hMin, math.Max(0.5*hi, math.Min(2*hi, target))) // damped update
			}
			return redo
		}
		if it == 0 {
			// The evaluated bodies in tree order: one group walk per leaf (again by tree position).
			sim.ParallelFor(len(sv.leaves), leafGrain, 0, func(lo, hi int) {
				w := walkPool.Get().(*walk)
				for _, leaf := range sv.leaves[lo:hi] {
					w.stack, w.cand = sv.tree.candidates(leaf, false, w.stack, w.cand)
					nd := &sv.tree.nodes[leaf]
					for k := nd.lo; k < nd.hi; k++ {
						i := sv.tree.order[k]
						if !isActive(i) {
							sv.again[k] = false
							continue
						}
						hi, rho := h[i], 0.0
						sv.tree.scan(w.cand, x[i], y[i], z[i], 2*hi, false, func(j int32, r2 float64) {
							rho += m[j] * W(math.Sqrt(r2), hi)
						})
						sv.again[k] = setH(i, rho)
					}
				}
				walkPool.Put(w)
			})
			sv.todo = sv.todo[:0]
			for _, leaf := range sv.leaves {
				nd := &sv.tree.nodes[leaf]
				for k := nd.lo; k < nd.hi; k++ {
					if sv.again[k] {
						sv.todo = append(sv.todo, sv.tree.order[k])
					}
				}
			}
		} else {
			// The few bodies that still iterate: one walk each.
			sim.ParallelFor(len(sv.todo), sphGrain, 0, func(lo, hi int) {
				w := walkPool.Get().(*walk)
				for k, i := range sv.todo[lo:hi] {
					hi, rho := h[i], 0.0
					w.stack = sv.tree.each(x[i], y[i], z[i], 2*hi, false, w.stack, func(j int32, r2 float64) {
						rho += m[j] * W(math.Sqrt(r2), hi)
					})
					sv.again[lo+k] = setH(i, rho)
				}
				walkPool.Put(w)
			})
			j := 0
			for k, i := range sv.todo {
				if sv.again[k] {
					sv.todo[j] = i
					j++
				}
			}
			sv.todo = sv.todo[:j]
		}
		if len(sv.todo) > 0 {
			sv.tree.refitH(h)
		}
	}
	sv.tree.sortLeaves()

	if cap(sv.P) < s.Len() {
		sv.P, sv.C = make([]float64, s.Len()), make([]float64, s.Len())
	}
	P, C := sv.P[:s.Len()], sv.C[:s.Len()]
	sim.ParallelFor(len(sv.gas), 4096, 0, func(lo, hi int) {
		for _, i := range sv.gas[lo:hi] {
			u := math.Max(s.U[i], 0)
			P[i] = (gamma - 1) * s.Rho[i] * u
			C[i] = math.Sqrt(gamma * (gamma - 1) * u)
		}
	})

	// Forces, energy equation and velocity divergence (symmetric variable-h formulation).
	sim.ParallelFor(len(sv.leaves), leafGrain, 0, func(lo, hi int) {
		w := walkPool.Get().(*walk)
		for _, leaf := range sv.leaves[lo:hi] {
			w.stack, w.cand = sv.tree.candidates(leaf, true, w.stack, w.cand)
			nd := &sv.tree.nodes[leaf]
			for _, i := range sv.tree.order[nd.lo:nd.hi] {
				if !isActive(i) {
					continue
				}
				hi, rhoi, pi := h[i], s.Rho[i], P[i]
				fi := pi / (rhoi * rhoi)
				var ax, ay, az, du, dv float64
				vsig, minStep, maxStep := 2*C[i], math.Inf(1), 0.0
				sv.tree.scan(w.cand, x[i], y[i], z[i], 2*hi, true, func(j int32, r2 float64) {
					if j == i || r2 == 0 {
						return
					}
					if step != nil && step[j] > 0 {
						minStep, maxStep = min(minStep, step[j]), max(maxStep, step[j])
					}
					r := math.Sqrt(r2)
					ex, ey, ez := (x[i]-x[j])/r, (y[i]-y[j])/r, (z[i]-z[j])/r
					vx, vy, vz := s.VelX[i]-s.VelX[j], s.VelY[i]-s.VelY[j], s.VelZ[i]-s.VelZ[j]
					vr := vx*ex + vy*ey + vz*ez // v_ij · r̂_ij
					vsig = max(vsig, C[i]+C[j]-3*min(0, vr))
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
				out.AccX[i], out.AccY[i], out.AccZ[i] = ax, ay, az
				out.DuDt[i], out.DivV[i] = du, dv/rhoi
				dt := sim.LimiterRatio * minStep
				if vsig > 0 {
					dt = min(dt, courant*hi/vsig)
				}
				out.Dt[i] = dt
				if maxStep > sim.LimiterRatio*dt {
					// Neighbors with much longer steps must be woken up (Saitoh & Makino 2009).
					sv.tree.scan(w.cand, x[i], y[i], z[i], 2*hi, true, func(j int32, _ float64) {
						if step[j] > sim.LimiterRatio*dt {
							lowerWake(&out.Wake[j], dt)
						}
					})
				}
			}
		}
		walkPool.Put(w)
	})
}

// lowerWake sets *w (float64 bits) to min(*w, dt), atomically.
func lowerWake(w *uint64, dt float64) {
	bits := math.Float64bits(dt)
	for {
		old := atomic.LoadUint64(w)
		if math.Float64frombits(old) <= dt || atomic.CompareAndSwapUint64(w, old, bits) {
			return
		}
	}
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
}

func resizeBool(s []bool, n int) []bool {
	if cap(s) < n {
		return make([]bool, n)
	}
	return s[:n]
}
