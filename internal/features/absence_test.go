package features

import (
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

// teamGame builds a team game where players 1..n dressed with the given TOI.
func teamGame(day int, toi map[int32]int) TeamGame {
	g := TeamGame{StartAt: time.Date(2025, 10, day, 0, 0, 0, 0, time.UTC), Season: 2025}
	for id, s := range toi {
		g.Players = append(g.Players, canon.PlayerMinutes{PlayerID: id, Position: "C", TOISeconds: s})
	}
	g.Players = append(g.Players, canon.PlayerMinutes{PlayerID: 99, Position: "G", TOISeconds: 3600})
	return g
}

func TestAbsenceBefore(t *testing.T) {
	// Players 1-3 are heavy-minute regulars, 4-6 depth. Player 3 misses the
	// last 4 games; player 6 plays rarely.
	var prev []TeamGame
	for d := 1; d <= 10; d++ {
		toi := map[int32]int{1: 1200, 2: 1100, 4: 600, 5: 500}
		if d <= 6 {
			toi[3] = 1000
		}
		if d%5 == 0 {
			toi[6] = 900
		}
		prev = append(prev, teamGame(d, toi))
	}
	if a := AbsenceBefore(prev[:4], 2025, nil); a.Known {
		t.Fatalf("too few games: %+v", a)
	}
	// Estimate: last game's lineup (no player 3).
	a := AbsenceBefore(prev, 2025, prev[len(prev)-1].Lineup())
	if !a.Known || a.Window != 10 {
		t.Fatalf("%+v", a)
	}
	// Player 3 played 6/10 = 0.6 ≥ share and TOI above the median: regular, missing 4.
	if len(a.Missing) != 1 || a.Missing[0].PlayerID != 3 || a.Missing[0].Missed != 4 {
		t.Fatalf("missing %+v", a.Missing)
	}
	if a.Share <= 0 || a.Share >= 0.5 {
		t.Fatalf("share %v", a.Share)
	}
	// Confirmed lineup with player 3 back: a return after 4 missed games.
	back := append(prev[len(prev)-1].Lineup(), 3)
	b := AbsenceBefore(prev, 2025, back)
	if len(b.Missing) != 0 || b.Share != 0 {
		t.Fatalf("missing after return %+v", b)
	}
	if len(b.Returning) != 0 {
		// Still a regular in the current window, so not counted as a return.
		t.Fatalf("returning %+v", b.Returning)
	}
	// A longer absence drops player 3 out of the regulars; then he is a return.
	for d := 11; d <= 16; d++ {
		prev = append(prev, teamGame(d, map[int32]int{1: 1200, 2: 1100, 4: 600, 5: 500}))
	}
	c := AbsenceBefore(prev, 2025, append(prev[len(prev)-1].Lineup(), 3))
	if len(c.Returning) != 1 || c.Returning[0].PlayerID != 3 || c.Returning[0].Missed != 10 {
		t.Fatalf("returning %+v", c.Returning)
	}
	// Goalies never count.
	for _, p := range append(a.Missing, c.Returning...) {
		if p.PlayerID == 99 {
			t.Fatal("goalie counted")
		}
	}
}

func TestAbsenceOtherSeasonIgnored(t *testing.T) {
	var prev []TeamGame
	for d := 1; d <= 6; d++ {
		g := teamGame(d, map[int32]int{1: 1200, 2: 1100, 3: 1000})
		g.Season = 2024
		prev = append(prev, g)
	}
	if a := AbsenceBefore(prev, 2025, []int32{1}); a.Known {
		t.Fatalf("previous season counted: %+v", a)
	}
}
