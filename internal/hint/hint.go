// Package hint is the market rule of plan 03 §3 (market-value-v1): a soft CZ
// book whose devigged 1X2 is at least MinEdge below the sharp book's on one
// selection, at a price with EV > 0 against the sharp probability. It uses no
// model at all; the information is the gap between books.
package hint

import (
	"math"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/marketprior"
)

// Version is the rule version written to the ledger.
const Version = "market-value-v1"

// Selections of the 1X2 market in prior order.
var Selections = [3]string{"home", "draw", "away"}

// Rule holds the fixed thresholds.
type Rule struct {
	Sharp    string
	Soft     []string
	MinEdge  float64 // p_sharp − p_soft
	MaxPrice float64
}

// Default is the pre-registered rule.
func Default() Rule {
	return Rule{Sharp: "pinnacle", Soft: []string{"betano.cz", "merkurxtip.cz"}, MinEdge: 0.03, MaxPrice: 10}
}

// Hint is one qualifying price.
type Hint struct {
	Book       string    `json:"book"`
	Market     string    `json:"market"`
	Selection  string    `json:"selection"`
	Price      float64   `json:"price"`
	PSharp     float64   `json:"p_sharp"`
	PSoft      float64   `json:"p_soft"`
	Edge       float64   `json:"edge"`
	EV         float64   `json:"ev"`
	CapturedAt time.Time `json:"captured_at"`
}

// Entry is one recorded hint of the ledger.
type Entry struct {
	ID         int64     `json:"id"`
	GameID     int64     `json:"game_id"`
	Rule       string    `json:"rule"`
	RecordedAt time.Time `json:"recorded_at"`
	Hint
}

// Books1X2 groups the 1X2 quotes per book (one price per selection, the latest).
func Books1X2(quotes []canon.OddsQuote) map[string]map[string]canon.OddsQuote {
	out := map[string]map[string]canon.OddsQuote{}
	for _, q := range quotes {
		if q.Market != "1x2" {
			continue
		}
		if out[q.Book] == nil {
			out[q.Book] = map[string]canon.OddsQuote{}
		}
		if cur, ok := out[q.Book][q.Selection]; !ok || q.CapturedAt.After(cur.CapturedAt) {
			out[q.Book][q.Selection] = q
		}
	}
	return out
}

// Fair is the Shin-devigged 1X2 of one book.
func Fair(book map[string]canon.OddsQuote) ([3]float64, bool) {
	prices := map[string]float64{}
	for s, q := range book {
		prices[s] = q.Price
	}
	return marketprior.Shin1X2(prices)
}

// Find returns the best qualifying price among the latest quotes, or nil.
func (r Rule) Find(quotes []canon.OddsQuote) *Hint {
	books := Books1X2(quotes)
	ps, ok := Fair(books[r.Sharp])
	if !ok {
		return nil
	}
	var best *Hint
	for _, b := range r.Soft {
		pb, ok := Fair(books[b])
		if !ok {
			continue
		}
		for i, s := range Selections {
			q := books[b][s]
			ev := ps[i]*q.Price - 1
			edge := ps[i] - pb[i]
			if edge < r.MinEdge || ev <= 0 || q.Price > r.MaxPrice {
				continue
			}
			if best == nil || ev > best.EV {
				best = &Hint{Book: b, Market: "1x2", Selection: s, Price: q.Price, PSharp: ps[i], PSoft: pb[i], Edge: edge, EV: ev, CapturedAt: q.CapturedAt}
			}
		}
	}
	return best
}

// Index maps a 1X2 selection to its position.
func Index(sel string) int {
	for i, s := range Selections {
		if s == sel {
			return i
		}
	}
	return -1
}

// CLV is ln(price · p_close): positive when the price beat the sharp close.
func CLV(price, pClose float64) float64 { return math.Log(price * pClose) }
