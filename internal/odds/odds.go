// Package odds normalises bookmaker markets onto the odds_snapshots vocabulary
// and computes margins.
package odds

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Market values stored in odds_snapshots.market.
const (
	ML           = "ml"        // winner incl. OT/SO (US style, "Home/Away")
	OneX2        = "1x2"       // 60 min with draw (EU style)
	PuckLine     = "puck_line" // handicap, line from the home team's view
	Total        = "total"
	P1OneX2      = "p1_1x2"
	P1Total      = "p1_total"
	CorrectScore = "correct_score"
	BTTS         = "btts"
	Other        = "other"
)

// Quote is one normalised price.
type Quote struct {
	Market    string
	Selection string
	Line      *float64
	Price     float64
	RawMarket string
}

// NormalizeAPIHockey maps an API-HOCKEY bet name and value ("3Way Result"/"Home",
// "Over/Under"/"Over 5.5", "Asian Handicap"/"Away +1.5") to a Quote.
// Unknown markets come back as Other so nothing is lost.
func NormalizeAPIHockey(betName, value string, price float64) Quote {
	b := strings.ToLower(betName)
	q := Quote{Market: Other, Selection: strings.TrimSpace(value), Price: price, RawMarket: betName}
	firstPeriod := strings.Contains(b, "1st period") || strings.Contains(b, "first period")
	otherPeriod := strings.Contains(b, "2nd period") || strings.Contains(b, "3rd period") ||
		strings.Contains(b, "second period") || strings.Contains(b, "third period")
	if otherPeriod || strings.Contains(b, "home") && strings.Contains(b, "total") || strings.Contains(b, "away") && strings.Contains(b, "total") {
		return q // team totals and later periods: keep raw
	}
	switch {
	case strings.Contains(b, "3way") || strings.Contains(b, "3 way") || b == "match winner" || b == "1x2":
		q.Market = OneX2
		if firstPeriod {
			q.Market = P1OneX2
		}
		q.Selection = side(value)
	case b == "home/away" || strings.Contains(b, "money") || strings.Contains(b, "incl. ot") || strings.Contains(b, "including ot"):
		if firstPeriod {
			return q
		}
		q.Market = ML
		q.Selection = side(value)
	case strings.Contains(b, "over/under") || strings.Contains(b, "total"):
		q.Market = Total
		if firstPeriod {
			q.Market = P1Total
		}
		q.Selection, q.Line = overUnder(value)
	case strings.Contains(b, "handicap") || strings.Contains(b, "puck line") || strings.Contains(b, "spread"):
		if firstPeriod {
			return q
		}
		q.Market = PuckLine
		q.Selection, q.Line = handicap(value)
	case strings.Contains(b, "correct score") || strings.Contains(b, "exact score"):
		if firstPeriod {
			return q
		}
		q.Market = CorrectScore
		q.Selection = strings.ReplaceAll(strings.TrimSpace(value), ":", "-")
	case strings.Contains(b, "both teams"):
		q.Market = BTTS
		q.Selection = strings.ToLower(strings.TrimSpace(value))
	}
	if q.Selection == "" || (q.Market != Other && q.Market != CorrectScore && q.Market != BTTS && q.Selection == "?") {
		q.Market = Other
		q.Selection = strings.TrimSpace(value)
	}
	return q
}

func side(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "home", "1":
		return "home"
	case "draw", "x":
		return "draw"
	case "away", "2":
		return "away"
	}
	return "?"
}

func overUnder(v string) (string, *float64) {
	f := strings.Fields(strings.ToLower(v))
	if len(f) != 2 || (f[0] != "over" && f[0] != "under") {
		return "?", nil
	}
	l, err := strconv.ParseFloat(strings.ReplaceAll(f[1], ",", "."), 64)
	if err != nil {
		return "?", nil
	}
	return f[0], &l
}

// handicap returns the side and the line from the home team's view
// ("Away +1.5" -> away, -1.5), so both selections of one line share a value.
func handicap(v string) (string, *float64) {
	f := strings.Fields(strings.ToLower(v))
	if len(f) != 2 {
		return "?", nil
	}
	l, err := strconv.ParseFloat(f[1], 64)
	if err != nil {
		return "?", nil
	}
	s := side(f[0])
	switch s {
	case "home":
	case "away":
		l = -l
	default:
		return "?", nil
	}
	return s, &l
}

// ParsePrice reads a decimal price sent as a JSON number or string.
func ParsePrice(raw json.RawMessage) (float64, error) {
	s := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	p, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil {
		return 0, fmt.Errorf("price %q: %w", s, err)
	}
	if p <= 1 || math.IsInf(p, 0) || math.IsNaN(p) {
		return 0, fmt.Errorf("price %q: not a decimal price > 1", s)
	}
	return p, nil
}

// Margin is the overround of a complete book: sum(1/price) - 1.
func Margin(prices ...float64) float64 {
	s := 0.0
	for _, p := range prices {
		s += 1 / p
	}
	return s - 1
}

// SnapshotKind labels a capture by time to start: t24h (20-28 h before),
// t1h (30-90 min before), closing (last 10 min before start), adhoc otherwise.
func SnapshotKind(captured, start time.Time) string {
	d := start.Sub(captured)
	switch {
	case d >= 20*time.Hour && d <= 28*time.Hour:
		return "t24h"
	case d >= 30*time.Minute && d <= 90*time.Minute:
		return "t1h"
	case d >= 0 && d <= 10*time.Minute:
		return "closing"
	default:
		return "adhoc"
	}
}
