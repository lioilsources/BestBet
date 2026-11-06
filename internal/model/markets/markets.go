// Package markets derives every market from one score distribution
// (plan 00 §0 bod 3, 01 §2.2). Hockey convention (docs/DATA_CONTRACT.md):
// all markets are settled on 60 minutes except ML, which includes OT/SO.
package markets

import (
	"fmt"
	"math"

	"github.com/lioilsources/BestBet/internal/canon"
)

// Market names match odds_snapshots.market.
const (
	OneX2        = "1x2"
	ML           = "ml"
	DoubleChance = "dc"
	DNB          = "dnb"
	Total        = "total"
	PuckLine     = "puck_line"
	CorrectScore = "correct_score"
	BTTS         = "btts"
	P1OneX2      = "p1_1x2"
	P1Total      = "p1_total"
)

// Lines priced by default.
var (
	TotalLines    = []float64{3.5, 4.5, 5.5, 6.5, 7.5}
	P1TotalLines  = []float64{0.5, 1.5, 2.5}
	PuckLineLines = []float64{-2.5, -1.5, 1.5, 2.5} // home handicap
)

// Key builds "sel@line".
func Key(sel string, line float64) string {
	return fmt.Sprintf("%s@%s", sel, trim(line))
}

func trim(f float64) string {
	s := fmt.Sprintf("%.2f", f)
	for s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

// Derive computes all markets. reg is the 60-min matrix, p1 the first-period
// matrix (may be nil), otHome = P(home wins OT/SO | tie after 60).
func Derive(reg, p1 canon.ScoreMatrix, otHome float64) canon.MarketProbs {
	m := canon.MarketProbs{}
	home, draw, away := outcome(reg)
	m[OneX2] = map[string]float64{"home": home, "draw": draw, "away": away}
	m[ML] = map[string]float64{"home": home + draw*otHome, "away": away + draw*(1-otHome)}
	m[DoubleChance] = map[string]float64{"1x": home + draw, "12": home + away, "x2": draw + away}
	if home+away > 0 {
		m[DNB] = map[string]float64{"home": home / (home + away), "away": away / (home + away)}
	}

	tot := map[string]float64{}
	for _, l := range TotalLines {
		over := sumWhere(reg, func(h, a int) bool { return float64(h+a) > l })
		tot[Key("over", l)] = over
		tot[Key("under", l)] = sumWhere(reg, func(h, a int) bool { return float64(h+a) < l })
	}
	m[Total] = tot

	pl := map[string]float64{}
	for _, l := range PuckLineLines {
		pl[Key("home", l)] = sumWhere(reg, func(h, a int) bool { return float64(h-a)+l > 0 })
		pl[Key("away", l)] = sumWhere(reg, func(h, a int) bool { return float64(h-a)+l < 0 })
	}
	m[PuckLine] = pl

	cs := map[string]float64{}
	for h := range reg {
		for a := range reg[h] {
			if h <= 7 && a <= 7 {
				cs[fmt.Sprintf("%d-%d", h, a)] = reg[h][a]
			}
		}
	}
	m[CorrectScore] = cs
	m[BTTS] = map[string]float64{
		"yes": sumWhere(reg, func(h, a int) bool { return h > 0 && a > 0 }),
		"no":  sumWhere(reg, func(h, a int) bool { return h == 0 || a == 0 }),
	}

	if p1 != nil {
		h1, d1, a1 := outcome(p1)
		m[P1OneX2] = map[string]float64{"home": h1, "draw": d1, "away": a1}
		pt := map[string]float64{}
		for _, l := range P1TotalLines {
			pt[Key("over", l)] = sumWhere(p1, func(h, a int) bool { return float64(h+a) > l })
			pt[Key("under", l)] = sumWhere(p1, func(h, a int) bool { return float64(h+a) < l })
		}
		m[P1Total] = pt
	}
	return m
}

func outcome(s canon.ScoreMatrix) (home, draw, away float64) {
	for h := range s {
		for a, p := range s[h] {
			switch {
			case h > a:
				home += p
			case h == a:
				draw += p
			default:
				away += p
			}
		}
	}
	t := home + draw + away
	if t > 0 && math.Abs(t-1) > 1e-12 {
		home, draw, away = home/t, draw/t, away/t
	}
	return
}

func sumWhere(s canon.ScoreMatrix, f func(h, a int) bool) float64 {
	t, all := 0.0, 0.0
	for h := range s {
		for a, p := range s[h] {
			all += p
			if f(h, a) {
				t += p
			}
		}
	}
	if all > 0 {
		return t / all
	}
	return 0
}
