package rating

import (
	"math"
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

func TestGoalieTimelineIsCausalAndShrinks(t *testing.T) {
	t0 := time.Date(2025, 9, 1, 18, 0, 0, 0, time.UTC)
	var ms []canon.Match
	for i := 0; i < 40; i++ {
		// Home goalie "Hot" saves everything; away goalie "Cold" saves 80 %.
		ms = append(ms, canon.Match{
			ID: int64(i + 1), StartAt: t0.Add(time.Duration(i) * 24 * time.Hour), HomeTeam: 1, AwayTeam: 2,
			Goalies: []canon.GoalieStart{
				{Side: canon.Home, Player: "Hot", Starter: true, ShotsAgainst: 30, Saves: 30},
				{Side: canon.Away, Player: "Cold", Starter: true, ShotsAgainst: 30, Saves: 24},
			},
		})
	}
	tl := GoalieTimeline(ms, DefaultGoalieConfig())
	first := tl[1][canon.Home]
	if first.StarterShot != 0 || math.Abs(first.StarterSV-0.900) > 1e-9 {
		t.Fatalf("first game must not see its own saves: %+v", first)
	}
	last := tl[40]
	h, a := last[canon.Home], last[canon.Away]
	if !(h.StarterSV > 0.9 && h.StarterSV < 1) || !(a.StarterSV < 0.9 && a.StarterSV > 0.8) {
		t.Fatalf("shrunk SV: hot %.4f cold %.4f", h.StarterSV, a.StarterSV)
	}
	// The only goalie of a team equals the team: no adjustment.
	if math.Abs(h.Adj()) > 1e-9 {
		t.Fatalf("single goalie adj %.4f", h.Adj())
	}
}
