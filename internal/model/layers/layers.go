// Package layers stacks measurable inputs on the periods baseline (plan 01 §5
// fáze 2): each layer is a log multiplier on λ with one coefficient fitted on
// the training window, so ablation is "same games, layer on vs off".
//
// Goalie layer (00 §1.4.5): the home starter changes the away team's expected
// goals by exp(β·adj_home), adj = log((1−SV_starter)/(1−SV_team)), and vice
// versa. β = 1 would mean "SV% translates one-to-one into goals"; β is fitted
// and reported, never assumed.
package layers

import (
	"fmt"
	"math"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/periods"
	"github.com/lioilsources/BestBet/internal/rating"
)

// Version identifies the layered model.
const Version = "periods-v1+goalie"

// Config selects layers.
type Config struct {
	Base   periods.Config
	Goalie bool
}

// Model is the baseline plus fitted layer coefficients.
type Model struct {
	Base       *periods.Model
	Cfg        Config
	GoalieBeta float64
	GoalieN    int // training games with both starters known
	timeline   map[int64]map[canon.Side]rating.GoalieState
}

// Fit trains the baseline, then the layer coefficients with the baseline fixed.
// timeline must be causal (rating.GoalieTimeline) and may cover all matches:
// each entry only uses games before its own.
func Fit(train []canon.Match, asOf time.Time, cfg Config, timeline map[int64]map[canon.Side]rating.GoalieState) (*Model, error) {
	base, err := periods.Fit(train, asOf, cfg.Base)
	if err != nil {
		return nil, err
	}
	m := &Model{Base: base, Cfg: cfg, timeline: timeline}
	if cfg.Goalie {
		m.fitGoalie(train, asOf)
	}
	return m, nil
}

type gRow struct {
	gh, ga, lh, la, adjH, adjA, w float64
}

func (m *Model) fitGoalie(train []canon.Match, asOf time.Time) {
	var rows []gRow
	for _, g := range train {
		if !g.Played() || !g.StartAt.Before(asOf) {
			continue
		}
		st, ok := m.timeline[g.ID]
		if !ok || !st[canon.Home].Known || !st[canon.Away].Known {
			continue
		}
		lh, la := m.Base.Lambdas(g.HomeTeam, g.AwayTeam)
		rows = append(rows, gRow{float64(g.Reg.Home), float64(g.Reg.Away), lh, la, st[canon.Home].Adj(), st[canon.Away].Adj(), m.Base.Weight(g.StartAt)})
	}
	m.GoalieN = len(rows)
	if len(rows) < 100 {
		return // too few games: layer stays off (β = 0)
	}
	ll := func(b float64) float64 {
		v := 0.0
		for _, r := range rows {
			lh := r.lh * math.Exp(b*r.adjA) // away starter changes home goals
			la := r.la * math.Exp(b*r.adjH)
			v += r.w * (r.gh*math.Log(lh) - lh + r.ga*math.Log(la) - la)
		}
		return v
	}
	m.GoalieBeta = periods.Golden(ll, -1, 4)
}

// Lambdas returns layered intensities and the per-side adjustments used.
func (m *Model) Lambdas(match canon.Match) (lh, la, adjH, adjA float64) {
	lh, la = m.Base.Lambdas(match.HomeTeam, match.AwayTeam)
	if m.Cfg.Goalie && m.GoalieBeta != 0 {
		if st, ok := m.timeline[match.ID]; ok {
			adjH, adjA = st[canon.Home].Adj(), st[canon.Away].Adj()
			lh *= math.Exp(m.GoalieBeta * adjA)
			la *= math.Exp(m.GoalieBeta * adjH)
		}
	}
	return
}

// Predict returns the prediction with goalie contributions in percentage points
// of the home regulation win (each side's starter vs its team's usual goalie).
func (m *Model) Predict(match canon.Match) (canon.Prediction, error) {
	lh, la, adjH, adjA := m.Lambdas(match)
	p, err := m.Base.PredictLambdas(match, lh, la, struct {
		Layers     string
		Beta       float64
		AdjH, AdjA float64
	}{Version, m.GoalieBeta, adjH, adjA})
	if err != nil {
		return p, fmt.Errorf("layers predict: %w", err)
	}
	if m.Cfg.Goalie {
		p.ModelVersion = Version
		full := m.Base.HomeWin(lh, la)
		withoutHome := m.Base.HomeWin(lh, la/math.Exp(m.GoalieBeta*adjH))
		withoutAway := m.Base.HomeWin(lh/math.Exp(m.GoalieBeta*adjA), la)
		p.Contributions = append(p.Contributions,
			canon.Contribution{Input: "goalie", Side: canon.Home, PP: 100 * (full - withoutHome)},
			canon.Contribution{Input: "goalie", Side: canon.Away, PP: 100 * (full - withoutAway)},
		)
	}
	return p, nil
}
