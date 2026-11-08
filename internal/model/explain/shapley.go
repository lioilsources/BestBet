package explain

import "math"

// Shapley returns the exact Shapley value of each of n inputs for value(mask),
// where bit i of mask means "input i at its actual value" (otherwise at the
// league average). Exponential in n; meant for n ≤ 10.
func Shapley(n int, value func(mask uint) float64) []float64 {
	full := uint(1)<<n - 1
	v := make([]float64, full+1)
	for m := uint(0); m <= full; m++ {
		v[m] = value(m)
	}
	fact := make([]float64, n+1)
	fact[0] = 1
	for i := 1; i <= n; i++ {
		fact[i] = fact[i-1] * float64(i)
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		bit := uint(1) << i
		for m := uint(0); m <= full; m++ {
			if m&bit != 0 {
				continue
			}
			k := popcount(m)
			w := fact[k] * fact[n-k-1] / fact[n]
			out[i] += w * (v[m|bit] - v[m])
		}
	}
	return out
}

func popcount(m uint) int {
	c := 0
	for ; m != 0; m &= m - 1 {
		c++
	}
	return c
}

// Efficiency is Σ contributions − (value(all) − value(none)); ~0 by construction.
func Efficiency(n int, value func(uint) float64, phi []float64) float64 {
	s := 0.0
	for _, x := range phi {
		s += x
	}
	return math.Abs(s - (value(uint(1)<<n-1) - value(0)))
}
