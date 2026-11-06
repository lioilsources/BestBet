package devig

import (
	"math"
	"testing"
)

func sum(xs []float64) float64 {
	t := 0.0
	for _, x := range xs {
		t += x
	}
	return t
}

func TestMethodsSumToOne(t *testing.T) {
	prices := []float64{1.55, 4.40, 5.20} // ~ 7.5 % overround
	p, err := Proportional(prices)
	if err != nil || math.Abs(sum(p)-1) > 1e-12 {
		t.Fatal(p, err)
	}
	pw, err := Power(prices)
	if err != nil || math.Abs(sum(pw)-1) > 1e-9 {
		t.Fatal(pw, err)
	}
	sh, z, err := Shin(prices)
	if err != nil || math.Abs(sum(sh)-1) > 1e-9 || z <= 0 || z > 0.2 {
		t.Fatal(sh, z, err)
	}
	// Favourite-longshot: Shin and power move mass from the longshot to the favourite
	// compared with proportional scaling.
	if !(sh[0] > p[0] && sh[2] < p[2]) || !(pw[0] > p[0] && pw[2] < p[2]) {
		t.Fatalf("prop=%v shin=%v power=%v", p, sh, pw)
	}
}

func TestShinFairBookUnchanged(t *testing.T) {
	sh, z, err := Shin([]float64{2, 2})
	if err != nil || z != 0 || math.Abs(sh[0]-0.5) > 1e-12 {
		t.Fatal(sh, z, err)
	}
}

func TestShinKnownValue(t *testing.T) {
	// Two-way 1.90/1.90: symmetric, so Shin equals proportional (0.5/0.5).
	sh, _, _ := Shin([]float64{1.9, 1.9})
	if math.Abs(sh[0]-0.5) > 1e-9 {
		t.Fatal(sh)
	}
}

func TestIncomplete(t *testing.T) {
	if _, err := Proportional([]float64{2}); err == nil {
		t.Fatal("one price")
	}
	if _, _, err := Shin([]float64{2, 1}); err == nil {
		t.Fatal("price 1")
	}
}

func TestMedian(t *testing.T) {
	if m, _ := Median([]float64{3, 1, 2, 10}); m != 2.5 {
		t.Fatal(m)
	}
}
