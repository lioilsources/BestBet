package calib

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestIsotonicMonotoneAndPools(t *testing.T) {
	p := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	y := []bool{false, true, false, true, true}
	c, err := FitIsotonic(p, y, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(c.Y); i++ {
		if c.Y[i] < c.Y[i-1] {
			t.Fatalf("not monotone: %v", c.Y)
		}
	}
	// 0.2 (true) and 0.3 (false) violate order and pool to 0.5.
	if math.Abs(c.Apply(0.25)-0.5) > 1e-9 {
		t.Fatalf("pooled value: %v (%+v)", c.Apply(0.25), c)
	}
}

func TestPlattRecoversDistortion(t *testing.T) {
	// True p = sigmoid(2*logit(raw)); Platt must find A~2, B~0.
	r := rand.New(rand.NewPCG(1, 2))
	var raw []float64
	var y []bool
	for i := 0; i < 20000; i++ {
		x := 0.05 + 0.9*r.Float64()
		truth := sigmoid(2 * logit(x))
		raw = append(raw, x)
		y = append(y, r.Float64() < truth)
	}
	c, _ := FitPlatt(raw, y)
	if math.Abs(c.A-2) > 0.15 || math.Abs(c.B) > 0.1 {
		t.Fatalf("%+v", c)
	}
}

func TestReliability(t *testing.T) {
	bins := Reliability([]float64{0.05, 0.15, 0.95, 1.0}, []bool{false, true, true, true}, 10)
	if bins[0].N != 1 || bins[1].Observed != 1 || bins[9].N != 2 {
		t.Fatalf("%+v", bins)
	}
}
