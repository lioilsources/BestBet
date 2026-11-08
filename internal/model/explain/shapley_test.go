package explain

import (
	"math"
	"testing"
)

func TestShapley(t *testing.T) {
	// Additive game: each input's value is its own weight.
	w := []float64{1, -2, 0.5}
	add := func(m uint) float64 {
		s := 0.0
		for i, x := range w {
			if m&(1<<i) != 0 {
				s += x
			}
		}
		return s
	}
	phi := Shapley(3, add)
	for i := range w {
		if math.Abs(phi[i]-w[i]) > 1e-12 {
			t.Fatalf("additive: %v", phi)
		}
	}
	// Pure interaction x0·x1 splits evenly.
	inter := func(m uint) float64 {
		if m&3 == 3 {
			return 1
		}
		return 0
	}
	phi = Shapley(2, inter)
	if math.Abs(phi[0]-0.5) > 1e-12 || math.Abs(phi[1]-0.5) > 1e-12 || Efficiency(2, inter, phi) > 1e-12 {
		t.Fatalf("interaction: %v", phi)
	}
}
