// Package marketprior starts from the market and adjusts it (00 §3.2: "prior =
// trh"). The prior is the Shin-devigged 1X2 price of one snapshot kind; layers
// shift the home/away log-odds by s = Σ β_j·x_j:
//
//	p_home ∝ p0_home·e^s,  p_draw ∝ p0_draw,  p_away ∝ p0_away·e^−s
//
// β is fitted by Newton on the multinomial likelihood with a small ridge. All
// β = 0 is the market itself, so every layer is measured against the market,
// not against our own baseline. Configuration: docs/DECISIONS.md 2026-11-08.
package marketprior

import (
	"math"
	"sort"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/devig"
)

// Version names the model.
const Version = "market-prior-v1"

// Kinds are the prior snapshots in report order; closing is a diagnostic only.
var Kinds = []string{"opening", "t24h", "t1h", "closing"}

var sels = []string{"home", "draw", "away"}

// Prior is the devigged market at one snapshot.
type Prior struct {
	P      [3]float64
	Source string // the sharp book, or "median" of the books present
}

// PriorAt returns the prior of one snapshot kind: the sharp book when it has all
// three selections, otherwise the median price per selection over the books.
func PriorAt(quotes []canon.OddsQuote, kind, sharp string) (Prior, bool) {
	byBook := map[string]map[string]float64{}
	for _, q := range quotes {
		if q.Market != "1x2" || q.Kind != kind {
			continue
		}
		if byBook[q.Book] == nil {
			byBook[q.Book] = map[string]float64{}
		}
		byBook[q.Book][q.Selection] = q.Price
	}
	if p, ok := shin(byBook[sharp]); ok {
		return Prior{P: p, Source: sharp}, true
	}
	books := make([]string, 0, len(byBook))
	for b := range byBook {
		books = append(books, b)
	}
	sort.Strings(books)
	med := map[string]float64{}
	for _, s := range sels {
		var xs []float64
		for _, b := range books {
			if v, ok := byBook[b][s]; ok {
				xs = append(xs, v)
			}
		}
		if len(xs) == 0 {
			return Prior{}, false
		}
		med[s], _ = devig.Median(xs)
	}
	p, ok := shin(med)
	return Prior{P: p, Source: "median"}, ok
}

// Shin1X2 devigs one book's 1X2 prices.
func Shin1X2(prices map[string]float64) ([3]float64, bool) { return shin(prices) }

func shin(prices map[string]float64) ([3]float64, bool) {
	var out [3]float64
	ps := make([]float64, 3)
	for i, s := range sels {
		v, ok := prices[s]
		if !ok || v <= 1 {
			return out, false
		}
		ps[i] = v
	}
	q, _, err := devig.Shin(ps)
	if err != nil {
		return out, false
	}
	copy(out[:], q)
	return out, true
}

// Obs is one game with a prior, its layer features and (for training) the result.
type Obs struct {
	ID    int64
	At    time.Time
	Prior [3]float64
	X     map[string]float64
	Y     int // 0 home, 1 draw, 2 away (60 min)
}

// Adjust applies the shift s to a prior.
func Adjust(p0 [3]float64, s float64) [3]float64 {
	h, d, a := p0[0]*math.Exp(s), p0[1], p0[2]*math.Exp(-s)
	z := h + d + a
	return [3]float64{h / z, d / z, a / z}
}

// Model holds the fitted coefficients.
type Model struct {
	Beta map[string]float64
	N    map[string]int // training games where the layer is non-zero
}

// Config of the fit.
type Config struct {
	Ridge float64 // default 1
	MinN  int     // default 100
}

// Fit estimates β for the named layers on training observations.
func Fit(train []Obs, names []string, cfg Config) *Model {
	if cfg.Ridge == 0 {
		cfg.Ridge = 1
	}
	if cfg.MinN == 0 {
		cfg.MinN = 100
	}
	m := &Model{Beta: map[string]float64{}, N: map[string]int{}}
	for _, o := range train {
		for _, l := range names {
			if o.X[l] != 0 {
				m.N[l]++
			}
		}
	}
	var act []string
	for _, l := range names {
		m.Beta[l] = 0
		if m.N[l] >= cfg.MinN {
			act = append(act, l)
		}
	}
	k := len(act)
	if k == 0 {
		return m
	}
	beta := make([]float64, k)
	for iter := 0; iter < 50; iter++ {
		grad := make([]float64, k)
		hess := make([][]float64, k)
		for i := range hess {
			hess[i] = make([]float64, k)
			hess[i][i] = cfg.Ridge
			grad[i] = -cfg.Ridge * beta[i]
		}
		for _, o := range train {
			s := 0.0
			for j, l := range act {
				s += beta[j] * o.X[l]
			}
			p := Adjust(o.Prior, s)
			// d log p_y / ds = t_y − (p_h − p_a), with t = (+1, 0, −1).
			t := [3]float64{1, 0, -1}[o.Y]
			g := t - (p[0] - p[2])
			v := p[0] + p[2] - (p[0]-p[2])*(p[0]-p[2]) // −d²/ds²
			for i, li := range act {
				grad[i] += g * o.X[li]
				for j, lj := range act {
					hess[i][j] += v * o.X[li] * o.X[lj]
				}
			}
		}
		step, ok := solve(hess, grad)
		if !ok {
			break
		}
		mx := 0.0
		for i := range beta {
			beta[i] += step[i]
			mx = math.Max(mx, math.Abs(step[i]))
		}
		if mx < 1e-9 {
			break
		}
	}
	for j, l := range act {
		m.Beta[l] = beta[j]
	}
	return m
}

// Predict returns the adjusted 1X2 of one observation.
func (m *Model) Predict(o Obs) [3]float64 {
	s := 0.0
	for l, b := range m.Beta {
		s += b * o.X[l]
	}
	return Adjust(o.Prior, s)
}

// Walk fits on everything before each 7-day window and predicts the window.
// Observations are sorted by time; the first windows with fewer than minTrain
// training games are skipped. betas collects β per window (for the report).
func Walk(obs []Obs, names []string, cfg Config, minTrain int) (pred map[int64][3]float64, betas map[string][]float64) {
	os := append([]Obs(nil), obs...)
	sort.Slice(os, func(i, j int) bool { return os[i].At.Before(os[j].At) })
	pred, betas = map[int64][3]float64{}, map[string][]float64{}
	if len(os) == 0 {
		return
	}
	step := 7 * 24 * time.Hour
	start := os[0].At.Truncate(24 * time.Hour)
	for d := start; !d.After(os[len(os)-1].At); d = d.Add(step) {
		end := d.Add(step)
		var train, win []Obs
		for _, o := range os {
			switch {
			case o.At.Before(d):
				train = append(train, o)
			case o.At.Before(end):
				win = append(win, o)
			}
		}
		if len(win) == 0 || len(train) < minTrain {
			continue
		}
		m := Fit(train, names, cfg)
		for l, b := range m.Beta {
			betas[l] = append(betas[l], b)
		}
		for _, o := range win {
			pred[o.ID] = m.Predict(o)
		}
	}
	return pred, betas
}

func solve(a [][]float64, b []float64) ([]float64, bool) {
	n := len(b)
	m := make([][]float64, n)
	for i := range a {
		m[i] = append(append([]float64(nil), a[i]...), b[i])
	}
	for c := 0; c < n; c++ {
		p := c
		for r := c + 1; r < n; r++ {
			if math.Abs(m[r][c]) > math.Abs(m[p][c]) {
				p = r
			}
		}
		if math.Abs(m[p][c]) < 1e-12 {
			return nil, false
		}
		m[c], m[p] = m[p], m[c]
		for r := 0; r < n; r++ {
			if r == c {
				continue
			}
			f := m[r][c] / m[c][c]
			for k := c; k <= n; k++ {
				m[r][k] -= f * m[c][k]
			}
		}
	}
	x := make([]float64, n)
	for i := range x {
		x[i] = m[i][n] / m[i][i]
	}
	return x, true
}
