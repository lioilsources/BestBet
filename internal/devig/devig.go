// Package devig removes the bookmaker margin from a complete set of decimal prices.
// Shin is the default (plan 01 §2.3): margin sits unevenly on longshots.
package devig

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// ErrIncomplete is returned for fewer than two prices or a price <= 1.
var ErrIncomplete = errors.New("devig: need >= 2 decimal prices > 1")

func implied(prices []float64) ([]float64, float64, error) {
	if len(prices) < 2 {
		return nil, 0, ErrIncomplete
	}
	q := make([]float64, len(prices))
	s := 0.0
	for i, p := range prices {
		if !(p > 1) || math.IsInf(p, 0) {
			return nil, 0, ErrIncomplete
		}
		q[i] = 1 / p
		s += q[i]
	}
	return q, s, nil
}

// Proportional scales implied probabilities to sum to 1.
func Proportional(prices []float64) ([]float64, error) {
	q, s, err := implied(prices)
	if err != nil {
		return nil, err
	}
	for i := range q {
		q[i] /= s
	}
	return q, nil
}

// Power finds k with sum(q_i^k) = 1 (Clarke's power method).
func Power(prices []float64) ([]float64, error) {
	q, s, err := implied(prices)
	if err != nil {
		return nil, err
	}
	if math.Abs(s-1) < 1e-12 {
		return q, nil
	}
	f := func(k float64) float64 {
		t := 0.0
		for _, x := range q {
			t += math.Pow(x, k)
		}
		return t - 1
	}
	// f is decreasing in k for q in (0,1); bracket the root.
	lo, hi := 0.5, 2.0
	for f(lo) < 0 {
		lo /= 2
	}
	for f(hi) > 0 {
		hi *= 2
	}
	k := bisect(f, lo, hi)
	out := make([]float64, len(q))
	for i, x := range q {
		out[i] = math.Pow(x, k)
	}
	return out, nil
}

// Shin solves for the insider share z (Shin 1993; Jullien & Salanié form):
// p_i = (sqrt(z^2 + 4(1-z) q_i^2 / S) - z) / (2(1-z)), with sum p_i = 1.
// Returns the probabilities and z.
func Shin(prices []float64) ([]float64, float64, error) {
	q, s, err := implied(prices)
	if err != nil {
		return nil, 0, err
	}
	probs := func(z float64) []float64 {
		out := make([]float64, len(q))
		for i, x := range q {
			out[i] = (math.Sqrt(z*z+4*(1-z)*x*x/s) - z) / (2 * (1 - z))
		}
		return out
	}
	if s <= 1 {
		// No overround (or an arbitrage): nothing to remove.
		p, err := Proportional(prices)
		return p, 0, err
	}
	f := func(z float64) float64 {
		t := 0.0
		for _, p := range probs(z) {
			t += p
		}
		return t - 1
	}
	z := bisect(f, 0, 0.999)
	return probs(z), z, nil
}

func bisect(f func(float64) float64, lo, hi float64) float64 {
	flo := f(lo)
	for i := 0; i < 200 && hi-lo > 1e-13; i++ {
		mid := (lo + hi) / 2
		fm := f(mid)
		if (fm > 0) == (flo > 0) {
			lo, flo = mid, fm
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// Median returns the median of xs (used for the median of books per selection).
func Median(xs []float64) (float64, error) {
	if len(xs) == 0 {
		return 0, fmt.Errorf("median of empty set")
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2], nil
	}
	return (s[n/2-1] + s[n/2]) / 2, nil
}
