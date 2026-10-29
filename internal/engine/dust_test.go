package engine

import "testing"

func TestDustShareFadesInHotGas(t *testing.T) {
	const lo, hi = 1.0, 10.0
	if d := dustShare(0, 0.5, lo, hi); d != 1 {
		t.Errorf("cold neutral gas carries %g of the dust, want 1", d)
	}
	if d := dustShare(0.5, 0.5, lo, hi); d != 0.5 {
		t.Errorf("half ionized gas carries %g, want 0.5", d)
	}
	mid := dustShare(0, 3.1622776601683795, lo, hi) // √10: halfway in log T
	if mid < 0.49 || mid > 0.51 {
		t.Errorf("dust at the middle of the sputtering range %g, want 0.5", mid)
	}
	if d := dustShare(0, 20, lo, hi); d != 0 {
		t.Errorf("hot gas keeps %g of its dust", d)
	}
	if d := dustShare(0, 20, 0, 0); d != 1 {
		t.Errorf("without units the temperature must not matter: %g", d)
	}
}
