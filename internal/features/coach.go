package features

import (
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

// CoachTenure is a head coach's run with a team (dates of first and last game).
type CoachTenure struct {
	Team     int32
	Coach    string
	From, To time.Time
}

// NewCoachGames is the window after an in-season coaching change (fixed,
// docs/DECISIONS.md 2026-11-08).
const NewCoachGames = 10

// CoachTimeline flags, per match and side, a head coach who took over during the
// season and is within his first NewCoachGames games. A tenure that starts with
// the team's first game of a season (summer change) is not flagged. The change
// itself is public before the game, so the flag is causal.
func CoachTimeline(matches []canon.Match, tenures []CoachTenure) map[int64]map[canon.Side]bool {
	ms := append([]canon.Match(nil), matches...)
	canon.SortMatches(ms)
	byTeam := map[int32][]CoachTenure{}
	for _, t := range tenures {
		byTeam[t.Team] = append(byTeam[t.Team], t)
	}
	type key struct {
		season int
		team   int32
	}
	first := map[key]time.Time{}
	for _, m := range ms {
		for _, t := range []int32{m.HomeTeam, m.AwayTeam} {
			if _, ok := first[key{m.Season, t}]; !ok {
				first[key{m.Season, t}] = m.StartAt
			}
		}
	}
	day := func(t time.Time) time.Time { return t.UTC().Truncate(24 * time.Hour) }
	games := map[[2]any]int{} // (team, tenure start) -> games so far
	out := map[int64]map[canon.Side]bool{}
	for _, m := range ms {
		st := map[canon.Side]bool{}
		for side, team := range map[canon.Side]int32{canon.Home: m.HomeTeam, canon.Away: m.AwayTeam} {
			d := day(m.StartAt)
			for _, t := range byTeam[team] {
				if d.Before(day(t.From)) || d.After(day(t.To)) {
					continue
				}
				k := [2]any{team, t.From}
				inSeason := day(t.From).After(day(first[key{m.Season, team}]))
				st[side] = inSeason && games[k] < NewCoachGames
				games[k]++
				break
			}
		}
		out[m.ID] = st
	}
	return out
}
