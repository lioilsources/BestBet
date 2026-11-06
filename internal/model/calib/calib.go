// Package calib maps raw model probabilities to calibrated ones (isotonic via
// PAVA, Platt scaling) and computes reliability bins. Binary events only;
// multi-way markets are calibrated one-vs-rest and renormalised by the caller.
package calib

import (
	"fmt"
	"math"
	"sort"
)

// Isotonic is a fitted monotone step function.
type Isotonic struct {
	Lo []float64 // block lower bounds (sorted)
	X  []float64 // block upper bounds (sorted)
	Y  []float64 // calibrated value per block
}

// FitIsotonic fits a non-decreasing map p -> P(y=1) with the pool-adjacent-
// violators algorithm. w may be nil (unit weights).
func FitIsotonic(p []float64, y []bool, w []float64) (Isotonic, error) {
	n := len(p)
	if n == 0 || len(y) != n || (w != nil && len(w) != n) {
		return Isotonic{}, fmt.Errorf("isotonic: bad input lengths")
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return p[idx[a]] < p[idx[b]] })
	type block struct{ sumW, sumWY, minX, maxX float64 }
	var st []block
	for _, i := range idx {
		wi := 1.0
		if w != nil {
			wi = w[i]
		}
		yi := 0.0
		if y[i] {
			yi = 1
		}
		st = append(st, block{wi, wi * yi, p[i], p[i]})
		for len(st) > 1 {
			a, b := st[len(st)-2], st[len(st)-1]
			if a.sumWY/a.sumW <= b.sumWY/b.sumW {
				break
			}
			st = st[:len(st)-2]
			st = append(st, block{a.sumW + b.sumW, a.sumWY + b.sumWY, a.minX, b.maxX})
		}
	}
	out := Isotonic{}
	for _, b := range st {
		out.Lo = append(out.Lo, b.minX)
		out.X = append(out.X, b.maxX)
		out.Y = append(out.Y, b.sumWY/b.sumW)
	}
	return out, nil
}

// Apply returns the calibrated value: constant inside a block, linear between
// neighbouring blocks, clamped to (eps, 1-eps) so log-loss stays finite.
func (c Isotonic) Apply(p float64) float64 {
	const eps = 1e-4
	if len(c.X) == 0 {
		return p
	}
	var v float64
	i := sort.SearchFloat64s(c.X, p) // first block with upper bound >= p
	switch {
	case i >= len(c.X):
		v = c.Y[len(c.Y)-1]
	case p >= c.Lo[i] || i == 0:
		v = c.Y[i]
	default:
		// Gap between block i-1 (ends at X[i-1]) and block i (starts at Lo[i]).
		x0, x1 := c.X[i-1], c.Lo[i]
		t := (p - x0) / (x1 - x0)
		v = c.Y[i-1] + t*(c.Y[i]-c.Y[i-1])
	}
	return math.Min(1-eps, math.Max(eps, v))
}

// Platt is sigmoid(A * logit(p) + B).
type Platt struct{ A, B float64 }

// FitPlatt fits A, B by Newton's method on the log-likelihood (with a tiny ridge).
func FitPlatt(p []float64, y []bool) (Platt, error) {
	if len(p) == 0 || len(p) != len(y) {
		return Platt{}, fmt.Errorf("platt: bad input lengths")
	}
	a, b := 1.0, 0.0
	for iter := 0; iter < 100; iter++ {
		var g0, g1, h00, h01, h11 float64
		for i := range p {
			x := logit(p[i])
			q := sigmoid(a*x + b)
			yi := 0.0
			if y[i] {
				yi = 1
			}
			d := q - yi
			g0 += d * x
			g1 += d
			v := q * (1 - q)
			h00 += v * x * x
			h01 += v * x
			h11 += v
		}
		const ridge = 1e-6
		h00 += ridge
		h11 += ridge
		det := h00*h11 - h01*h01
		if det == 0 {
			break
		}
		da := (h11*g0 - h01*g1) / det
		db := (h00*g1 - h01*g0) / det
		a -= da
		b -= db
		if math.Abs(da)+math.Abs(db) < 1e-10 {
			break
		}
	}
	return Platt{A: a, B: b}, nil
}

// Apply calibrates p.
func (c Platt) Apply(p float64) float64 { return sigmoid(c.A*logit(p) + c.B) }

// Bin is one reliability-diagram bucket.
type Bin struct {
	Lo, Hi   float64
	N        int
	MeanP    float64
	Observed float64
}

// Reliability buckets predictions into n equal-width bins.
func Reliability(p []float64, y []bool, n int) []Bin {
	bins := make([]Bin, n)
	for i := range bins {
		bins[i].Lo = float64(i) / float64(n)
		bins[i].Hi = float64(i+1) / float64(n)
	}
	for i, x := range p {
		k := int(x * float64(n))
		if k >= n {
			k = n - 1
		}
		if k < 0 {
			k = 0
		}
		bins[k].N++
		bins[k].MeanP += x
		if y[i] {
			bins[k].Observed++
		}
	}
	for i := range bins {
		if bins[i].N > 0 {
			bins[i].MeanP /= float64(bins[i].N)
			bins[i].Observed /= float64(bins[i].N)
		}
	}
	return bins
}

func logit(p float64) float64 {
	p = math.Min(1-1e-9, math.Max(1e-9, p))
	return math.Log(p / (1 - p))
}

func sigmoid(x float64) float64 { return 1 / (1 + math.Exp(-x)) }
