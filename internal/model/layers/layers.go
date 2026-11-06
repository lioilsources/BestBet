// Package layers stacks measurable inputs on the periods baseline (plan 01 §5
// fáze 2). Each layer k is a feature pair (x_home, x_away) that enters as
//
//	log λ_home += β_k · x_home,   log λ_away += β_k · x_away
//
// with the baseline held fixed. The β are fitted jointly (weighted Poisson
// likelihood, Newton, small ridge), so an ablation is "same games, layer on vs
// off" and every β is reported, never assumed.
//
// Stacking is out of sample: a training game enters the β fit with the baseline
// intensities that were predicted for it before it was played (Inputs.OOS, filled
// by Predict in walk-forward order; live, from the predictions table). Fitting β
// against a baseline refitted on the same games would let the baseline's
// hindsight absorb exactly the recent-form signal the wave layers measure.
//
// Layers (fixed a priori, docs/DECISIONS.md):
//   - goalie: x_home = adj of the away starter (00 §1.4.5).
//   - wave-level: Kalman level gap minus the baseline's expected goal gap.
//   - wave-vel: Kalman velocity gap (00 §1.4.2: "vlna = rychlost").
//   - b2b: back-to-back of the opponent minus own (fatigue, 00 §1.4.5).
package layers

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/features"
	"github.com/lioilsources/BestBet/internal/model/periods"
	"github.com/lioilsources/BestBet/internal/rating"
)

// Layer names.
const (
	Goalie    = "goalie"
	WaveLevel = "wave-level"
	WaveVel   = "wave-vel"
	B2B       = "b2b"
)

// All lists the layers in report order.
var All = []string{Goalie, WaveLevel, WaveVel, B2B}

// Inputs are causal timelines over all matches (each entry uses only earlier games).
type Inputs struct {
	Goalie map[int64]map[canon.Side]rating.GoalieState
	Wave   map[int64]map[canon.Side]rating.WaveState
	Rest   map[int64]map[canon.Side]features.Rest
	// OOS holds the baseline (λ_home, λ_away) first predicted for each game. Not
	// safe for concurrent use; the walk-forward is sequential.
	OOS map[int64][2]float64
}

// BuildInputs computes every timeline with the default configs.
func BuildInputs(matches []canon.Match) Inputs {
	return Inputs{
		Goalie: rating.GoalieTimeline(matches, rating.DefaultGoalieConfig()),
		Wave:   rating.WaveTimeline(matches, rating.DefaultWaveConfig()),
		Rest:   features.RestTimeline(matches),
		OOS:    map[int64][2]float64{},
	}
}

// Config selects layers.
type Config struct {
	Base   periods.Config
	Layers []string
	Ridge  float64 // L2 on β in weighted games (default 1)
	MinN   int     // a layer needs this many training games with x ≠ 0 (default 100)
}

// Version names the model built from cfg.
func Version(layers []string) string {
	if len(layers) == 0 {
		return periods.Version
	}
	return periods.Version + "+" + strings.Join(layers, "+")
}

// Model is the baseline plus fitted layer coefficients.
type Model struct {
	Base  *periods.Model
	Cfg   Config
	Beta  map[string]float64
	N     map[string]int // training games where the layer is active
	in    Inputs
	names []string // active (fitted) layers
}

// Fit trains the baseline on train, then the layer coefficients with it fixed.
func Fit(train []canon.Match, asOf time.Time, cfg Config, in Inputs) (*Model, error) {
	if cfg.Ridge == 0 {
		cfg.Ridge = 1
	}
	if cfg.MinN == 0 {
		cfg.MinN = 100
	}
	base, err := periods.Fit(train, asOf, cfg.Base)
	if err != nil {
		return nil, err
	}
	m := &Model{Base: base, Cfg: cfg, Beta: map[string]float64{}, N: map[string]int{}, in: in}
	for _, l := range cfg.Layers {
		if !known(l) {
			return nil, fmt.Errorf("layers: unknown layer %q", l)
		}
	}
	m.fit(train, asOf)
	return m, nil
}

func known(l string) bool {
	for _, k := range All {
		if k == l {
			return true
		}
	}
	return false
}

// x returns the layer feature pair for a match given the baseline intensities.
func (m *Model) x(layer string, g canon.Match, lh, la float64) (xh, xa float64) {
	switch layer {
	case Goalie:
		st, ok := m.in.Goalie[g.ID]
		if !ok {
			return 0, 0
		}
		return st[canon.Away].Adj(), st[canon.Home].Adj()
	case WaveLevel:
		st, ok := m.in.Wave[g.ID]
		if !ok || st[canon.Home].Games < 5 || st[canon.Away].Games < 5 {
			return 0, 0
		}
		// Kalman gap (in goals) minus what the baseline already expects. The
		// league home advantage is in both, so it cancels to first order.
		d := (st[canon.Home].Level - st[canon.Away].Level) - (lh - la - m.homeGap())
		return d / 2, -d / 2
	case WaveVel:
		st, ok := m.in.Wave[g.ID]
		if !ok || st[canon.Home].Games < 5 || st[canon.Away].Games < 5 {
			return 0, 0
		}
		// Velocity is per game; ×10 reads as "level change over ten games".
		d := 10 * (st[canon.Home].Velocity - st[canon.Away].Velocity)
		return d / 2, -d / 2
	case B2B:
		st, ok := m.in.Rest[g.ID]
		if !ok {
			return 0, 0
		}
		d := b(st[canon.Away].BackToBack) - b(st[canon.Home].BackToBack)
		return d, -d
	}
	return 0, 0
}

// homeGap is the baseline's expected goal gap of two average teams.
func (m *Model) homeGap() float64 {
	return math.Exp(m.Base.Mu+m.Base.HomeAdv) - math.Exp(m.Base.Mu)
}

func b(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

type row struct {
	gh, ga, lh, la, w float64
	xh, xa            []float64
}

func (m *Model) fit(train []canon.Match, asOf time.Time) {
	cand := m.Cfg.Layers
	var rows []row
	for _, g := range train {
		if !g.Played() || !g.StartAt.Before(asOf) {
			continue
		}
		oos, ok := m.in.OOS[g.ID]
		if !ok {
			continue
		}
		lh, la := oos[0], oos[1]
		r := row{gh: float64(g.Reg.Home), ga: float64(g.Reg.Away), lh: lh, la: la, w: m.Base.Weight(g.StartAt)}
		for _, l := range cand {
			xh, xa := m.x(l, g, lh, la)
			if xh != 0 || xa != 0 {
				m.N[l]++
			}
			r.xh = append(r.xh, xh)
			r.xa = append(r.xa, xa)
		}
		rows = append(rows, r)
	}
	var idx []int
	for i, l := range cand {
		if m.N[l] >= m.Cfg.MinN {
			idx = append(idx, i)
			m.names = append(m.names, l)
		} else {
			m.Beta[l] = 0 // too little data: layer stays off
		}
	}
	k := len(idx)
	if k == 0 {
		return
	}
	beta := make([]float64, k)
	for iter := 0; iter < 30; iter++ {
		grad := make([]float64, k)
		hess := make([][]float64, k)
		for i := range hess {
			hess[i] = make([]float64, k)
			hess[i][i] = m.Cfg.Ridge
			grad[i] = -m.Cfg.Ridge * beta[i]
		}
		for _, r := range rows {
			eh, ea := 0.0, 0.0
			for j, c := range idx {
				eh += beta[j] * r.xh[c]
				ea += beta[j] * r.xa[c]
			}
			lh, la := r.lh*math.Exp(eh), r.la*math.Exp(ea)
			for i, ci := range idx {
				grad[i] += r.w * ((r.gh-lh)*r.xh[ci] + (r.ga-la)*r.xa[ci])
				for j, cj := range idx {
					hess[i][j] += r.w * (lh*r.xh[ci]*r.xh[cj] + la*r.xa[ci]*r.xa[cj])
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
		if mx < 1e-8 {
			break
		}
	}
	for j, c := range idx {
		m.Beta[cand[c]] = beta[j]
	}
}

// solve returns A⁻¹b for a small symmetric positive definite A (Gauss–Jordan).
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

// Terms returns each active layer's log multipliers for a match.
func (m *Model) Terms(match canon.Match) (lh, la float64, terms map[string][2]float64) {
	lh, la = m.Base.Lambdas(match.HomeTeam, match.AwayTeam)
	terms = map[string][2]float64{}
	for _, l := range m.names {
		xh, xa := m.x(l, match, lh, la)
		terms[l] = [2]float64{m.Beta[l] * xh, m.Beta[l] * xa}
	}
	return lh, la, terms
}

// Lambdas returns the layered intensities.
func (m *Model) Lambdas(match canon.Match) (float64, float64) {
	lh, la, terms := m.Terms(match)
	for _, t := range terms {
		lh *= math.Exp(t[0])
		la *= math.Exp(t[1])
	}
	return lh, la
}

// Predict returns the prediction with one contribution per active layer: the
// change of the home regulation win in percentage points when that layer's
// term is removed (07 §4).
func (m *Model) Predict(match canon.Match) (canon.Prediction, error) {
	lh0, la0, terms := m.Terms(match)
	if _, seen := m.in.OOS[match.ID]; !seen && m.in.OOS != nil && match.StartAt.After(m.Base.AsOf) {
		m.in.OOS[match.ID] = [2]float64{lh0, la0}
	}
	lh, la := lh0, la0
	names := make([]string, 0, len(terms))
	for l, t := range terms {
		lh *= math.Exp(t[0])
		la *= math.Exp(t[1])
		names = append(names, l)
	}
	sort.Strings(names)
	type tv struct {
		L    string
		B, H float64
		A    float64
	}
	var hashed []tv
	for _, l := range names {
		hashed = append(hashed, tv{l, m.Beta[l], terms[l][0], terms[l][1]})
	}
	p, err := m.Base.PredictLambdas(match, lh, la, hashed)
	if err != nil {
		return p, fmt.Errorf("layers predict: %w", err)
	}
	p.ModelVersion = Version(m.Cfg.Layers)
	p.Contributions = m.Base.StrengthContributions(match)
	full := m.Base.HomeWin(lh, la)
	for _, l := range m.Cfg.Layers {
		t, ok := terms[l]
		if !ok {
			continue
		}
		without := m.Base.HomeWin(lh/math.Exp(t[0]), la/math.Exp(t[1]))
		p.Contributions = append(p.Contributions, canon.Contribution{Input: l, PP: 100 * (full - without)})
	}
	return p, nil
}
