package features

import (
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

// Rest is one side's schedule load before a game. It only uses kick-off times of
// earlier games, which are known in advance, so it is causal.
type Rest struct {
	DaysSinceLast float64 // 0 when the team has no earlier game in the data
	BackToBack    bool    // previous game started < 36 h earlier (consecutive days)
	GamesLast7    int     // games in the 7 days before kick-off
	Known         bool    // the team has an earlier game in the same season
}

// RestTimeline returns each side's rest before every match (any phase counts).
func RestTimeline(matches []canon.Match) map[int64]map[canon.Side]Rest {
	ms := append([]canon.Match(nil), matches...)
	canon.SortMatches(ms)
	history := map[int32][]canon.Match{}
	out := map[int64]map[canon.Side]Rest{}
	for _, m := range ms {
		st := map[canon.Side]Rest{}
		for side, team := range map[canon.Side]int32{canon.Home: m.HomeTeam, canon.Away: m.AwayTeam} {
			st[side] = restBefore(history[team], m)
		}
		out[m.ID] = st
		history[m.HomeTeam] = append(history[m.HomeTeam], m)
		history[m.AwayTeam] = append(history[m.AwayTeam], m)
	}
	return out
}

func restBefore(prev []canon.Match, m canon.Match) Rest {
	var r Rest
	for i := len(prev) - 1; i >= 0; i-- {
		p := prev[i]
		if p.Season != m.Season || !p.StartAt.Before(m.StartAt) {
			continue
		}
		gap := m.StartAt.Sub(p.StartAt)
		if !r.Known {
			r.Known = true
			r.DaysSinceLast = gap.Hours() / 24
			r.BackToBack = gap < 36*time.Hour
		}
		if gap <= 7*24*time.Hour {
			r.GamesLast7++
		} else {
			break
		}
	}
	return r
}
