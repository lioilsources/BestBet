package hockey

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Phase values match games.phase.
const (
	PhaseRegular    = "regular"
	PhasePrePlayoff = "pre_playoff"
	PhasePlayoff    = "playoff"
	PhaseRelegation = "relegation_playoff"
	PhasePromotion  = "promotion_qual"
)

// DetectPhase classifies a free-text stage label from a feed ("Regular Season",
// "Play Offs", "Quarter-finals", "Slutspel", "Play Out", "Kvalserien", "Baráž" ...).
// pre_playoff cannot be read from a label; it needs standings and is set in M1.
func DetectPhase(stage string) string {
	s := strings.ToLower(Fold(stage))
	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(s, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("relegation", "play out", "playout", "play-out", "baraz", "nedre", "playdown", "play-down"):
		return PhaseRelegation
	case has("kval", "qualification", "promotion", "qualifier"):
		return PhasePromotion
	case has("play off", "playoff", "play-off", "slutspel", "final", "semi", "quarter", "1/4", "1/2", "round of", "pre-playoff", "predkolo"):
		return PhasePlayoff
	default:
		return PhaseRegular
	}
}

// SeriesGame is the input for DeriveSeries.
type SeriesGame struct {
	ID         int64
	HomeTeamID int32
	AwayTeamID int32
	StartAt    time.Time
	HomeScore  *int32
	AwayScore  *int32
}

// SeriesState is the derived series information for one game.
type SeriesState struct {
	SeriesID    string
	GameNo      int
	ScoreBefore string // home team's wins - away team's wins before this game
}

// DeriveSeries groups non-regular games by unordered team pair and numbers them.
// A pair that meets again after a gap of more than maxGap starts a new series
// (e.g. a play-out after a play-off). Unfinished games count as played but add no win.
func DeriveSeries(prefix string, games []SeriesGame, maxGap time.Duration) map[int64]SeriesState {
	sorted := append([]SeriesGame(nil), games...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].StartAt.Before(sorted[j].StartAt) })

	type series struct {
		id   string
		n    int
		wins map[int32]int
		last time.Time
	}
	open := map[[2]int32]*series{}
	seq := map[[2]int32]int{}
	out := make(map[int64]SeriesState, len(sorted))
	for _, g := range sorted {
		a, b := g.HomeTeamID, g.AwayTeamID
		if a > b {
			a, b = b, a
		}
		key := [2]int32{a, b}
		s := open[key]
		if s == nil || g.StartAt.Sub(s.last) > maxGap {
			seq[key]++
			s = &series{id: fmt.Sprintf("%s:%d-%d:%d", prefix, a, b, seq[key]), wins: map[int32]int{}}
			open[key] = s
		}
		s.n++
		out[g.ID] = SeriesState{
			SeriesID:    s.id,
			GameNo:      s.n,
			ScoreBefore: fmt.Sprintf("%d-%d", s.wins[g.HomeTeamID], s.wins[g.AwayTeamID]),
		}
		s.last = g.StartAt
		if g.HomeScore != nil && g.AwayScore != nil {
			switch {
			case *g.HomeScore > *g.AwayScore:
				s.wins[g.HomeTeamID]++
			case *g.AwayScore > *g.HomeScore:
				s.wins[g.AwayTeamID]++
			}
		}
	}
	return out
}
