package features

import (
	"math"
	"sort"

	"github.com/lioilsources/BestBet/internal/canon"
)

// PressureConfig fixes how a "race for the line" is read from the table
// (docs/DECISIONS.md 2026-11-08, M1b F1). Lines are 1-based ranks C: the line
// sits between rank C and C+1.
type PressureConfig struct {
	Lines      []int
	MinShare   float64 // a team must have played this share of its regular season
	PointsWin  int     // regulation win
	PointsOTW  int     // win in overtime or shootout
	PointsOTL  int     // loss in overtime or shootout
	PerGameGap float64 // in the race when |points − line| <= remaining games × this
}

// PressureConfigFor returns the fixed config of a league (nil = no layer).
func PressureConfigFor(league string) *PressureConfig {
	switch league {
	case "nhl":
		return &PressureConfig{Lines: []int{16}, MinShare: 0.6, PointsWin: 2, PointsOTW: 2, PointsOTL: 1, PerGameGap: 1}
	case "swe-allsvenskan":
		return &PressureConfig{Lines: []int{10, 12}, MinShare: 0.6, PointsWin: 3, PointsOTW: 2, PointsOTL: 1, PerGameGap: 1}
	}
	return nil
}

// PressureTimeline reports, for every regular-season match, whether each side
// was in a race for a table line before kick-off. The schedule (games per team)
// is known in advance; points only come from earlier results.
func PressureTimeline(matches []canon.Match, c PressureConfig) map[int64]map[canon.Side]bool {
	ms := append([]canon.Match(nil), matches...)
	canon.SortMatches(ms)
	type key struct {
		season int
		team   int32
	}
	scheduled := map[key]int{}
	for _, m := range ms {
		if m.Phase != "regular" {
			continue
		}
		scheduled[key{m.Season, m.HomeTeam}]++
		scheduled[key{m.Season, m.AwayTeam}]++
	}
	points := map[key]int{}
	played := map[key]int{}
	teams := map[int][]int32{}
	seen := map[key]bool{}
	for _, m := range ms {
		if m.Phase != "regular" {
			continue
		}
		for _, t := range []int32{m.HomeTeam, m.AwayTeam} {
			if k := (key{m.Season, t}); !seen[k] {
				seen[k] = true
				teams[m.Season] = append(teams[m.Season], t)
			}
		}
	}
	out := map[int64]map[canon.Side]bool{}
	for _, m := range ms {
		if m.Phase != "regular" {
			continue
		}
		pts := make([]float64, 0, len(teams[m.Season]))
		for _, t := range teams[m.Season] {
			pts = append(pts, float64(points[key{m.Season, t}]))
		}
		sort.Sort(sort.Reverse(sort.Float64Slice(pts)))
		st := map[canon.Side]bool{}
		for side, t := range map[canon.Side]int32{canon.Home: m.HomeTeam, canon.Away: m.AwayTeam} {
			k := key{m.Season, t}
			n := scheduled[k]
			if n == 0 || float64(played[k]) < c.MinShare*float64(n) {
				continue
			}
			remaining := float64(n - played[k])
			gap := math.Inf(1)
			for _, line := range c.Lines {
				if line < 1 || line >= len(pts) {
					continue
				}
				lp := (pts[line-1] + pts[line]) / 2
				gap = math.Min(gap, math.Abs(float64(points[k])-lp))
			}
			st[side] = gap <= remaining*c.PerGameGap
		}
		out[m.ID] = st
		if !m.Played() || m.Final == nil {
			continue
		}
		hp, ap := c.result(m)
		points[key{m.Season, m.HomeTeam}] += hp
		points[key{m.Season, m.AwayTeam}] += ap
		played[key{m.Season, m.HomeTeam}]++
		played[key{m.Season, m.AwayTeam}]++
	}
	return out
}

func (c PressureConfig) result(m canon.Match) (home, away int) {
	homeWon := m.Final.Home > m.Final.Away
	extra := m.Reg.Home == m.Reg.Away // decided after 60 minutes
	switch {
	case homeWon && !extra:
		return c.PointsWin, 0
	case homeWon:
		return c.PointsOTW, c.PointsOTL
	case !extra:
		return 0, c.PointsWin
	default:
		return c.PointsOTL, c.PointsOTW
	}
}
