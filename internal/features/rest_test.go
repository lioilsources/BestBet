package features

import (
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

func TestRestTimeline(t *testing.T) {
	d := func(day, hour int) time.Time { return time.Date(2025, 10, day, hour, 0, 0, 0, time.UTC) }
	ms := []canon.Match{
		{ID: 1, Season: 2025, StartAt: d(1, 17), HomeTeam: 1, AwayTeam: 2},
		{ID: 2, Season: 2025, StartAt: d(2, 17), HomeTeam: 3, AwayTeam: 1}, // team 1 back-to-back
		{ID: 3, Season: 2025, StartAt: d(6, 17), HomeTeam: 2, AwayTeam: 1},
		{ID: 4, Season: 2026, StartAt: d(7, 17), HomeTeam: 1, AwayTeam: 2}, // new season: unknown
	}
	tl := RestTimeline(ms)
	if r := tl[1][canon.Home]; r.Known {
		t.Fatalf("first game: %+v", r)
	}
	if r := tl[2][canon.Away]; !r.BackToBack || r.DaysSinceLast != 1 || r.GamesLast7 != 1 {
		t.Fatalf("b2b: %+v", r)
	}
	if r := tl[3][canon.Away]; r.BackToBack || r.DaysSinceLast != 4 || r.GamesLast7 != 2 {
		t.Fatalf("rest 4 days: %+v", r)
	}
	if r := tl[3][canon.Home]; r.DaysSinceLast != 5 {
		t.Fatalf("team 2: %+v", r)
	}
	if r := tl[4][canon.Home]; r.Known {
		t.Fatalf("season boundary: %+v", r)
	}
}
