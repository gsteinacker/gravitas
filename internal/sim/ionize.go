package sim

import (
	"math"
	"slices"
)

// Photoionization with shadowing (T-037). Gas is deposited onto a coarse 3D grid of neutral and
// ionized density (from the previous ionization state); for every gas body the optical depth
// towards every source is integrated along the ray: τ = IonOpacity ∫ρ_neutral dl + IonRecomb ∫ρ_ion² dl.
// Neutral gas absorbs, ionized gas only through recombinations (∝ n²), which limits the ionization
// front like a Strömgren sphere. The ionized fraction is max over sources of exp(−τ). Ionized gas is heated towards
// UIon, shadowed gas cools towards UFloor (updateGas). Dense clumps therefore cast shadows behind
// which cold gas survives while the exposed gas is heated, expands and streams away: pillars and
// "heads" like the Horsehead form at the edge of molecular clouds.
//
// Simplifications: recombinations as a local opacity (no explicit photon budget or recombination
// time), no diffuse field, one frequency, no geometric dilution; the grid limits shadow resolution.

const ionGrid = 64

type ionizer struct {
	rho, rhoIon           []float64 // neutral and ionized density per cell
	minX, minY, minZ, inv float64   // grid origin and 1/cell size
	cell                  float64
	sources               []int
	qx, qy, qz            []float64 // scratch for the grid bounds
}

const (
	ionTail     = 0.005 // fraction of gas per side and axis that may lie outside the grid
	ionMargin   = 0.1   // grid margin around the neutral gas, relative to its extent
	minGridGas  = 100   // fewer relevant bodies than this: the grid covers all gas
	ionRelevant = 0.1   // optical depth across its smoothing length that puts a body into the grid
)

// quantileRange returns the ionTail and 1−ionTail quantiles of v (reordering v).
func quantileRange(v []float64) (lo, hi float64) {
	slices.Sort(v)
	k := int(ionTail * float64(len(v)-1))
	return v[k], v[len(v)-1-k]
}

// ionize updates s.Ion of all gas bodies. It returns false if there are no sources.
func (m *Simulation) ionize() bool {
	s := m.State
	io := &m.ion
	io.sources = io.sources[:0]
	for i := range s.Len() {
		if s.Lum[i] > 0 && s.Alive[i] {
			io.sources = append(io.sources, i)
		}
	}
	if len(io.sources) == 0 {
		return false
	}
	// The grid covers the bulk of the gas that matters for absorption: bodies with an optical depth
	// of at least ionRelevant across their own smoothing length (neutral gas, dense ionized flows),
	// per axis between the ionTail and 1−ionTail quantiles, widened by ionMargin. Thin ionized gas
	// and gas streaming away do not coarsen it; outside it only misses its small column. Without
	// smoothing lengths yet, or with too few such bodies, the grid covers all gas.
	kappa, recomb := m.Params.Gas.IonOpacity, m.Params.Gas.IonRecomb
	for pass := 0; pass == 0 || (pass < 2 && len(io.qx) < minGridGas); pass++ {
		io.qx, io.qy, io.qz = io.qx[:0], io.qy[:0], io.qz[:0]
		for i := range s.Len() {
			if s.Kind[i] != Gas {
				continue
			}
			rn, ri := s.Rho[i]*(1-s.Ion[i]), s.Rho[i]*s.Ion[i]
			if pass == 1 || s.H[i] == 0 || (kappa*rn+recomb*ri*ri)*s.H[i] >= ionRelevant {
				io.qx, io.qy, io.qz = append(io.qx, s.PosX[i]), append(io.qy, s.PosY[i]), append(io.qz, s.PosZ[i])
			}
		}
	}
	if len(io.qx) == 0 {
		return true
	}
	minX, maxX := quantileRange(io.qx)
	minY, maxY := quantileRange(io.qy)
	minZ, maxZ := quantileRange(io.qz)
	pad := ionMargin * math.Max(math.Max(maxX-minX, maxY-minY), maxZ-minZ)
	minX, minY, minZ, maxX, maxY, maxZ = minX-pad, minY-pad, minZ-pad, maxX+pad, maxY+pad, maxZ+pad
	io.cell = math.Max(math.Max(maxX-minX, maxY-minY), math.Max(maxZ-minZ, 1e-9)) / (ionGrid - 1)
	io.inv = 1 / io.cell
	io.minX, io.minY, io.minZ = minX, minY, minZ
	io.rho, io.rhoIon = resize(io.rho, ionGrid*ionGrid*ionGrid), resize(io.rhoIon, ionGrid*ionGrid*ionGrid)
	clear(io.rho)
	clear(io.rhoIon)
	vol := io.cell * io.cell * io.cell
	for i := range s.Len() {
		if s.Kind[i] == Gas {
			io.deposit(s.PosX[i], s.PosY[i], s.PosZ[i], s.Mass[i]*(1-s.Ion[i])/vol, s.Mass[i]*s.Ion[i]/vol)
		}
	}
	ParallelFor(s.Len(), 1024, 0, func(lo, hi int) {
		for i := lo; i < hi; i++ {
			if s.Kind[i] != Gas {
				continue
			}
			ion := 0.0
			for _, src := range io.sources {
				neutral, ionized2 := io.column(s.PosX[src], s.PosY[src], s.PosZ[src], s.PosX[i], s.PosY[i], s.PosZ[i])
				ion = math.Max(ion, math.Exp(-kappa*neutral-recomb*ionized2))
			}
			s.Ion[i] = ion
		}
	})
	return true
}

// corners returns the lower grid node of the cell containing (x, y, z) and the trilinear weights
// inside it (cloud-in-cell); ok is false outside the grid.
func (io *ionizer) corners(x, y, z float64) (k int, wx, wy, wz float64, ok bool) {
	fx, fy, fz := (x-io.minX)*io.inv, (y-io.minY)*io.inv, (z-io.minZ)*io.inv
	if fx < 0 || fy < 0 || fz < 0 || fx >= ionGrid-1 || fy >= ionGrid-1 || fz >= ionGrid-1 {
		return 0, 0, 0, 0, false
	}
	cx, cy, cz := int(fx), int(fy), int(fz)
	return (cz*ionGrid+cy)*ionGrid + cx, fx - float64(cx), fy - float64(cy), fz - float64(cz), true
}

// The eight corner offsets of a cell in the flat grid.
var cornerOffsets = [8]int{0, 1, ionGrid, ionGrid + 1, ionGrid * ionGrid, ionGrid*ionGrid + 1, ionGrid*ionGrid + ionGrid, ionGrid*ionGrid + ionGrid + 1}

func cornerWeights(wx, wy, wz float64) [8]float64 {
	ux, uy, uz := 1-wx, 1-wy, 1-wz
	return [8]float64{ux * uy * uz, wx * uy * uz, ux * wy * uz, wx * wy * uz, ux * uy * wz, wx * uy * wz, ux * wy * wz, wx * wy * wz}
}

// deposit spreads neutral and ionized density over the eight surrounding grid nodes.
func (io *ionizer) deposit(x, y, z, neutral, ionized float64) {
	k, wx, wy, wz, ok := io.corners(x, y, z)
	if !ok {
		return
	}
	for c, w := range cornerWeights(wx, wy, wz) {
		io.rho[k+cornerOffsets[c]] += w * neutral
		io.rhoIon[k+cornerOffsets[c]] += w * ionized
	}
}

// sample interpolates neutral and ionized density at (x, y, z) (zero outside the grid).
func (io *ionizer) sample(x, y, z float64) (neutral, ionized float64) {
	k, wx, wy, wz, ok := io.corners(x, y, z)
	if !ok {
		return 0, 0
	}
	for c, w := range cornerWeights(wx, wy, wz) {
		neutral += w * io.rho[k+cornerOffsets[c]]
		ionized += w * io.rhoIon[k+cornerOffsets[c]]
	}
	return neutral, ionized
}

// column integrates ρ_neutral and ρ_ion² along the segment from the source to the target, in
// half-cell steps, leaving out the last half cell before the target (a body does not shadow itself).
func (io *ionizer) column(sx, sy, sz, tx, ty, tz float64) (neutral, ionized2 float64) {
	dx, dy, dz := tx-sx, ty-sy, tz-sz
	d := math.Sqrt(dx*dx + dy*dy + dz*dz)
	end := d - 0.5*io.cell
	if end <= 0 {
		return 0, 0
	}
	steps := int(2*end*io.inv) + 1
	ds := end / float64(steps)
	for k := range steps {
		f := (float64(k) + 0.5) * ds / d
		n, i := io.sample(sx+f*dx, sy+f*dy, sz+f*dz)
		neutral += n * ds
		ionized2 += i * i * ds
	}
	return neutral, ionized2
}
