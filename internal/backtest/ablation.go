package backtest

import (
	"fmt"
	"math"
	"strings"
)

// Delta is the paired log-loss difference (variant − baseline) on shared games;
// negative = the layer helps. CI is a normal 95 % interval of the mean.
type Delta struct {
	Season       int // 0 = all
	N            int
	MeanDelta1X2 float64
	CILo, CIHi   float64
	P            float64 // two-sided p-value of mean Δ = 0 (normal approximation)
	MeanDeltaML  float64
}

// PairedDelta compares two runs game by game.
func PairedDelta(base, variant []Row) map[int]Delta {
	byID := map[int64]Row{}
	for _, r := range base {
		byID[r.Match.ID] = r
	}
	type acc struct{ d, d2, ml []float64 }
	groups := map[int]*acc{0: {}}
	for _, v := range variant {
		b, ok := byID[v.Match.ID]
		if !ok {
			continue
		}
		pb, pv := p1x2(b), p1x2(v)
		d := -logc(pv[v.Outcome1X2]) + logc(pb[b.Outcome1X2])
		mlv, mlb := v.Pred.Markets["ml"]["home"], b.Pred.Markets["ml"]["home"]
		if !v.HomeWonML {
			mlv, mlb = 1-mlv, 1-mlb
		}
		dml := -logc(mlv) + logc(mlb)
		for _, k := range []int{0, v.Match.Season} {
			if groups[k] == nil {
				groups[k] = &acc{}
			}
			groups[k].d = append(groups[k].d, d)
			groups[k].ml = append(groups[k].ml, dml)
		}
	}
	out := map[int]Delta{}
	for k, a := range groups {
		n := float64(len(a.d))
		if n < 2 {
			continue
		}
		mean, ml := 0.0, 0.0
		for i := range a.d {
			mean += a.d[i]
			ml += a.ml[i]
		}
		mean /= n
		ml /= n
		v := 0.0
		for _, x := range a.d {
			v += (x - mean) * (x - mean)
		}
		se := math.Sqrt(v / (n - 1) / n)
		pv := 1.0
		if se > 0 {
			pv = math.Erfc(math.Abs(mean/se) / math.Sqrt2)
		}
		out[k] = Delta{Season: k, N: len(a.d), MeanDelta1X2: mean, CILo: mean - 1.96*se, CIHi: mean + 1.96*se, P: pv, MeanDeltaML: ml}
	}
	return out
}

// RenderAblation renders the paired table for one layer.
func RenderAblation(layer string, deltas map[int]Delta, holdout int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "| Sezona | Zápasů | Δ log-loss 1X2 (%s − baseline) | 95%% interval | Δ log-loss ML |\n|---|---|---|---|---|\n", layer)
	var seasons []int
	for k := range deltas {
		if k != 0 {
			seasons = append(seasons, k)
		}
	}
	sortInts(seasons)
	for _, s := range append(seasons, 0) {
		d, ok := deltas[s]
		if !ok {
			continue
		}
		label := "**celkem**"
		if s != 0 {
			label = fmt.Sprintf("%d/%02d", s, (s+1)%100)
			if holdout != 0 && s >= holdout {
				label += " (odložená)"
			}
		}
		fmt.Fprintf(&b, "| %s | %d | %s | %s až %s | %s |\n", label, d.N, signed(d.MeanDelta1X2), signed(d.CILo), signed(d.CIHi), signed(d.MeanDeltaML))
	}
	return b.String()
}

func signed(f float64) string {
	return strings.Replace(fmt.Sprintf("%+.4f", f), ".", ",", 1)
}

func sortInts(xs []int) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
}
