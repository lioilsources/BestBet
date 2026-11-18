// Package soccer is the sport logic of football (plan 05 S1): results after 90
// minutes, extra time and penalties mapped onto the shared game columns, and
// settlement of Asian handicap and total lines including quarter lines.
//
// Storage convention (docs/DATA_CONTRACT.md): home_60/away_60 hold the score
// after regular time (90 min plus stoppage), period_scores the two halves, ot_*
// extra time, decided_in reg | ot | so. A shootout winner is often unknown in
// free sources; then SOWinner stays empty and Final equals the score after
// extra time.
package soccer

import (
	"fmt"
	"math"

	"github.com/lioilsources/BestBet/internal/hockey"
)

// Score is one home-away pair (same type as hockey's, so the store is shared).
type Score = hockey.Score

// Result is a played game in the shared result shape.
type Result = hockey.Result

// OTType values for soccer games.
const (
	OTNone  = "none"
	OTExtra = "et_2x15"
)

// BuildResult assembles a result from two halves, optional extra time and an
// optional shootout. shootout is true when the game went to penalties;
// soWinner is "home", "away" or "" when the source does not say.
func BuildResult(h1, h2 Score, et *Score, shootout bool, soWinner string) (Result, error) {
	if h1.Home < 0 || h1.Away < 0 || h2.Home < 0 || h2.Away < 0 {
		return Result{}, fmt.Errorf("negative half score %v %v", h1, h2)
	}
	r := Result{Periods: []Score{h1, h2}, DecidedIn: "reg"}
	r.Reg = Score{Home: h1.Home + h2.Home, Away: h1.Away + h2.Away}
	r.Final = r.Reg
	if et != nil {
		if r.Reg.Home != r.Reg.Away {
			return Result{}, fmt.Errorf("extra time after decided regular time %v", r.Reg)
		}
		r.OT = et
		r.Final = Score{Home: r.Reg.Home + et.Home, Away: r.Reg.Away + et.Away}
		r.DecidedIn = "ot"
	}
	if shootout {
		if r.Final.Home != r.Final.Away {
			return Result{}, fmt.Errorf("shootout after a decided game %v", r.Final)
		}
		r.DecidedIn = "so"
		switch soWinner {
		case "home":
			r.Final.Home++
		case "away":
			r.Final.Away++
		case "":
		default:
			return Result{}, fmt.Errorf("shootout winner %q", soWinner)
		}
		r.SOWinner = soWinner
	}
	return r, nil
}

// Settlement is the stake fraction returned as profit units of one bet:
// 1 win, 0.5 half win, 0 push, -0.5 half loss, -1 loss (multiply wins by
// price-1 for the profit).
type Settlement float64

// QuarterLine reports whether a line is a quarter line (±0.25, ±0.75, ...),
// which settles as two half stakes on the neighbouring lines.
func QuarterLine(line float64) bool {
	f := math.Abs(line*4 - math.Round(line*2)*2)
	return f > 0.5
}

// splitLine returns the one or two lines a stake is spread over.
func splitLine(line float64) []float64 {
	if QuarterLine(line) {
		return []float64{line - 0.25, line + 0.25}
	}
	return []float64{line}
}

func settleMargin(m float64) Settlement {
	switch {
	case m > 0:
		return 1
	case m < 0:
		return -1
	}
	return 0
}

// SettleAH settles a home Asian handicap bet: line is added to the home
// score (−0.5 means home must win). Use the negated line for the away side.
func SettleAH(line float64, s Score) Settlement {
	parts := splitLine(line)
	var sum Settlement
	for _, l := range parts {
		sum += settleMargin(float64(s.Home-s.Away) + l)
	}
	return sum / Settlement(len(parts))
}

// SettleOver settles an over bet on total goals; the under is its negation.
func SettleOver(line float64, s Score) Settlement {
	parts := splitLine(line)
	var sum Settlement
	for _, l := range parts {
		sum += settleMargin(float64(s.Home+s.Away) - l)
	}
	return sum / Settlement(len(parts))
}
