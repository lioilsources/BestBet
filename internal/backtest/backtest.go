// Package backtest runs walk-forward evaluation (plan 01 §5 fáze 1, 00 §3.3):
// train on everything before d, predict games in [d, d+step), move d. Every
// model metric is reported next to the market's (devigged closing) and next to
// a naive base-rate baseline; a model number alone is a forbidden output (01 §6).
package backtest

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/devig"
)

// Predictor is a fitted model.
type Predictor interface {
	Predict(canon.Match) (canon.Prediction, error)
}

// FitFunc trains a model on matches strictly before asOf.
type FitFunc func(train []canon.Match, asOf time.Time) (Predictor, error)

// Config controls the walk.
type Config struct {
	From     time.Time     // first prediction window start
	To       time.Time     // last window ends here (exclusive)
	Step     time.Duration // window length (default 7 days)
	MinTrain int           // skip windows with fewer played training games
}

// Row is one predicted, played game.
type Row struct {
	Match      canon.Match
	Pred       canon.Prediction
	Base1X2    [3]float64  // naive: training frequencies of home/draw/away
	Market1X2  *[3]float64 // devigged closing (median of books), nil if no odds
	MarketML   *[2]float64
	Outcome1X2 int  // 0 home, 1 draw, 2 away (60 min)
	HomeWonML  bool // incl. OT/SO
}

// Run walks forward. odds may be nil.
func Run(all []canon.Match, odds map[int64][]canon.OddsQuote, fit FitFunc, cfg Config) ([]Row, error) {
	if cfg.Step == 0 {
		cfg.Step = 7 * 24 * time.Hour
	}
	ms := append([]canon.Match(nil), all...)
	canon.SortMatches(ms)
	var rows []Row
	for d := cfg.From; d.Before(cfg.To); d = d.Add(cfg.Step) {
		end := d.Add(cfg.Step)
		var window []canon.Match
		nTrain := 0
		for _, m := range ms {
			if m.StartAt.Before(d) && m.Played() {
				nTrain++
			}
			if !m.StartAt.Before(d) && m.StartAt.Before(end) && m.Played() && m.Final != nil {
				window = append(window, m)
			}
		}
		if len(window) == 0 || nTrain < cfg.MinTrain {
			continue
		}
		// Training data never contains results from the window or later.
		train := make([]canon.Match, 0, nTrain)
		for _, m := range ms {
			if m.StartAt.Before(d) {
				train = append(train, m)
			}
		}
		model, err := fit(train, d)
		if err != nil {
			return nil, fmt.Errorf("fit at %s: %w", d.Format("2006-01-02"), err)
		}
		base := baseRates(train)
		for _, m := range window {
			p, err := model.Predict(m)
			if err != nil {
				return nil, fmt.Errorf("predict %d: %w", m.ID, err)
			}
			r := Row{Match: m, Pred: p, Base1X2: base, HomeWonML: m.Final.Home > m.Final.Away}
			switch {
			case m.Reg.Home > m.Reg.Away:
				r.Outcome1X2 = 0
			case m.Reg.Home == m.Reg.Away:
				r.Outcome1X2 = 1
			default:
				r.Outcome1X2 = 2
			}
			r.Market1X2, r.MarketML = marketProbs(odds[m.ID])
			rows = append(rows, r)
		}
	}
	return rows, nil
}

func baseRates(train []canon.Match) [3]float64 {
	var c [3]float64
	n := 0.0
	for _, m := range train {
		if !m.Played() {
			continue
		}
		switch {
		case m.Reg.Home > m.Reg.Away:
			c[0]++
		case m.Reg.Home == m.Reg.Away:
			c[1]++
		default:
			c[2]++
		}
		n++
	}
	for i := range c {
		c[i] = (c[i] + 1) / (n + 3) // Laplace
	}
	return c
}

// marketProbs takes the latest pre-start capture per book (closing if present),
// the median price per selection across books, and Shin-devigs it.
func marketProbs(quotes []canon.OddsQuote) (*[3]float64, *[2]float64) {
	pick := func(market string, sels []string) []float64 {
		latest := map[string]map[string]canon.OddsQuote{} // book -> sel -> quote
		for _, q := range quotes {
			if q.Market != market {
				continue
			}
			if latest[q.Book] == nil {
				latest[q.Book] = map[string]canon.OddsQuote{}
			}
			if cur, ok := latest[q.Book][q.Selection]; !ok || q.CapturedAt.After(cur.CapturedAt) {
				latest[q.Book][q.Selection] = q
			}
		}
		prices := make([]float64, len(sels))
		for i, s := range sels {
			var xs []float64
			for _, bm := range latest {
				if q, ok := bm[s]; ok {
					xs = append(xs, q.Price)
				}
			}
			if len(xs) == 0 {
				return nil
			}
			prices[i], _ = devig.Median(xs)
		}
		p, _, err := devig.Shin(prices)
		if err != nil {
			return nil
		}
		return p
	}
	var m1x2 *[3]float64
	var mml *[2]float64
	if p := pick("1x2", []string{"home", "draw", "away"}); p != nil {
		m1x2 = &[3]float64{p[0], p[1], p[2]}
	}
	if p := pick("ml", []string{"home", "away"}); p != nil {
		mml = &[2]float64{p[0], p[1]}
	}
	return m1x2, mml
}

// Metrics summarises a set of rows.
type Metrics struct {
	N                 int
	LogLoss1X2        float64
	Brier1X2          float64
	BaseLogLoss1X2    float64
	NMarket1X2        int
	ModelLLOnMarket   float64 // model log-loss on the games that have market odds
	MarketLogLoss1X2  float64 // devigged closing on the same games
	LogLossML         float64
	NMarketML         int
	MarketLogLossML   float64
	ModelLLMLOnMarket float64
	DrawRate          float64
	MeanPDraw         float64
}

// ErrNoRows is returned for an empty set.
var ErrNoRows = errors.New("backtest: no rows")

// Summarise computes metrics; probabilities are clamped to [1e-6, 1].
func Summarise(rows []Row) (Metrics, error) {
	if len(rows) == 0 {
		return Metrics{}, ErrNoRows
	}
	var m Metrics
	for _, r := range rows {
		p := p1x2(r)
		m.N++
		m.LogLoss1X2 += -logc(p[r.Outcome1X2])
		m.BaseLogLoss1X2 += -logc(r.Base1X2[r.Outcome1X2])
		for k := 0; k < 3; k++ {
			y := 0.0
			if k == r.Outcome1X2 {
				y = 1
			}
			m.Brier1X2 += (p[k] - y) * (p[k] - y)
		}
		ph := r.Pred.Markets["ml"]["home"]
		pml := ph
		if !r.HomeWonML {
			pml = 1 - ph
		}
		m.LogLossML += -logc(pml)
		if r.Outcome1X2 == 1 {
			m.DrawRate++
		}
		m.MeanPDraw += p[1]
		if r.Market1X2 != nil {
			m.NMarket1X2++
			m.MarketLogLoss1X2 += -logc(r.Market1X2[r.Outcome1X2])
			m.ModelLLOnMarket += -logc(p[r.Outcome1X2])
		}
		if r.MarketML != nil {
			m.NMarketML++
			q := r.MarketML[0]
			if !r.HomeWonML {
				q = r.MarketML[1]
			}
			m.MarketLogLossML += -logc(q)
			m.ModelLLMLOnMarket += -logc(pml)
		}
	}
	n := float64(m.N)
	m.LogLoss1X2 /= n
	m.BaseLogLoss1X2 /= n
	m.Brier1X2 /= n
	m.LogLossML /= n
	m.DrawRate /= n
	m.MeanPDraw /= n
	if m.NMarket1X2 > 0 {
		m.MarketLogLoss1X2 /= float64(m.NMarket1X2)
		m.ModelLLOnMarket /= float64(m.NMarket1X2)
	}
	if m.NMarketML > 0 {
		m.MarketLogLossML /= float64(m.NMarketML)
		m.ModelLLMLOnMarket /= float64(m.NMarketML)
	}
	return m, nil
}

func p1x2(r Row) [3]float64 {
	x := r.Pred.Markets["1x2"]
	return [3]float64{x["home"], x["draw"], x["away"]}
}

func logc(p float64) float64 { return math.Log(math.Max(p, 1e-6)) }

// BySeason groups rows.
func BySeason(rows []Row) map[int][]Row {
	out := map[int][]Row{}
	for _, r := range rows {
		out[r.Match.Season] = append(out[r.Match.Season], r)
	}
	return out
}

// Seasons returns sorted keys.
func Seasons(by map[int][]Row) []int {
	var ks []int
	for k := range by {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	return ks
}

// BenjaminiHochberg returns which p-values are rejected at FDR q (00 §1.4.1).
func BenjaminiHochberg(p []float64, q float64) []bool {
	n := len(p)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return p[idx[a]] < p[idx[b]] })
	k := -1
	for rank, i := range idx {
		if p[i] <= q*float64(rank+1)/float64(n) {
			k = rank
		}
	}
	out := make([]bool, n)
	for rank := 0; rank <= k; rank++ {
		out[idx[rank]] = true
	}
	return out
}
