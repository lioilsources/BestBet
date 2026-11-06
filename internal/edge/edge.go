// Package edge compares model probabilities with bookmaker prices (plan 01
// §2.3, 00 §1.4.3): Shin-devigged implied probability per book, edge in
// percentage points, EV, fractional Kelly and the expected log growth used to
// rank the day's bets. A selection is "green" only when the edge clears the
// market threshold, EV > 0, and the market is on the CLV allow list (a market
// where the model has not shown positive CLV is never green, however large
// the edge).
package edge

import (
	"math"
	"sort"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/devig"
	"github.com/lioilsources/BestBet/internal/model/markets"
)

// Policy is the threshold set (01 §2.3).
type Policy struct {
	Edge1X2    float64         // 1x2 and ml (0.03)
	EdgeOther  float64         // correct score, periods, totals, lines (0.05)
	Kelly      float64         // fraction (0.25)
	MaxStake   float64         // cap as share of bankroll (0.02)
	CLVMarkets map[string]bool // markets with proven positive CLV; empty = nothing green
}

// DefaultPolicy has no CLV-proven market yet (brána G1 not passed).
func DefaultPolicy() Policy {
	return Policy{Edge1X2: 0.03, EdgeOther: 0.05, Kelly: 0.25, MaxStake: 0.02, CLVMarkets: map[string]bool{}}
}

// Offer is one price at one book with the model's view of it.
type Offer struct {
	Book      string   `json:"book"`
	Market    string   `json:"market"`
	Selection string   `json:"selection"`
	Line      *float64 `json:"line,omitempty"`
	Price     float64  `json:"price"`
	PModel    float64  `json:"p_model"`
	PImplied  float64  `json:"p_implied"` // devigged (Shin) when the book quotes the full market, else 1/price
	Edge      float64  `json:"edge"`      // p_model − p_implied
	EV        float64  `json:"ev"`        // p_model · price − 1
	Stake     float64  `json:"stake"`     // fractional Kelly, capped; 0 when EV ≤ 0
	LogGrowth float64  `json:"log_growth"`
	Green     bool     `json:"green"`
	Why       string   `json:"why,omitempty"` // reason when not green
}

// Key maps a quote onto the markets key ("home", "over@5.5", "away@-1.5").
func Key(q canon.OddsQuote) string {
	if q.Line != nil {
		return markets.Key(q.Selection, *q.Line)
	}
	return q.Selection
}

// groupKey identifies one book market (and line) whose selections are devigged together.
func groupKey(q canon.OddsQuote) string {
	l := ""
	if q.Line != nil {
		l = markets.Key("", *q.Line) // both sides of a line share the home-view value
	}
	return q.Book + "|" + q.Market + "|" + l
}

func groupSize(market string) int {
	switch market {
	case "1x2", "p1_1x2":
		return 3
	case "ml", "total", "p1_total", "puck_line", "btts", "dnb":
		return 2
	}
	return 0 // correct score etc.: never devigged per book
}

// Offers evaluates every quote of an allowed book that the model prices.
func Offers(p canon.MarketProbs, quotes []canon.OddsQuote, allowed map[string]bool, pol Policy) []Offer {
	groups := map[string][]canon.OddsQuote{}
	for _, q := range quotes {
		if allowed[q.Book] {
			groups[groupKey(q)] = append(groups[groupKey(q)], q)
		}
	}
	var out []Offer
	for _, g := range groups {
		implied := map[int]float64{}
		if n := groupSize(g[0].Market); n > 0 && len(g) == n {
			prices := make([]float64, n)
			for i, q := range g {
				prices[i] = q.Price
			}
			if ps, _, err := devig.Shin(prices); err == nil {
				for i := range ps {
					implied[i] = ps[i]
				}
			}
		}
		for i, q := range g {
			pm, ok := p[q.Market][Key(q)]
			if !ok || q.Price <= 1 {
				continue
			}
			o := Offer{Book: q.Book, Market: q.Market, Selection: q.Selection, Line: q.Line, Price: q.Price, PModel: pm, PImplied: 1 / q.Price}
			if v, ok := implied[i]; ok {
				o.PImplied = v
			}
			o.Edge = pm - o.PImplied
			o.EV = pm*q.Price - 1
			if o.EV > 0 {
				f := pol.Kelly * o.EV / (q.Price - 1)
				o.Stake = math.Min(f, pol.MaxStake)
				o.LogGrowth = pm*math.Log1p(o.Stake*(q.Price-1)) + (1-pm)*math.Log1p(-o.Stake)
			}
			th := pol.EdgeOther
			if q.Market == "1x2" || q.Market == "ml" {
				th = pol.Edge1X2
			}
			switch {
			case o.EV <= 0:
				o.Why = "EV ≤ 0"
			case o.Edge < th:
				o.Why = "hrana pod prahem"
			case !pol.CLVMarkets[q.Market]:
				o.Why = "trh bez prokázaného CLV"
			default:
				o.Green = true
			}
			out = append(out, o)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LogGrowth != out[j].LogGrowth {
			return out[i].LogGrowth > out[j].LogGrowth
		}
		if out[i].Market != out[j].Market {
			return out[i].Market < out[j].Market
		}
		if out[i].Book != out[j].Book {
			return out[i].Book < out[j].Book
		}
		return Key(canon.OddsQuote{Selection: out[i].Selection, Line: out[i].Line}) < Key(canon.OddsQuote{Selection: out[j].Selection, Line: out[j].Line})
	})
	return out
}
