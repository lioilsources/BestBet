package asa

import (
	"os"
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/soccer"
)

func load(t *testing.T) (map[string]Game, map[string][]Shot) {
	t.Helper()
	b, err := os.ReadFile("testdata/games.json")
	if err != nil {
		t.Fatal(err)
	}
	gs, err := ParseGames(b)
	if err != nil {
		t.Fatal(err)
	}
	games := map[string]Game{}
	for _, g := range gs {
		games[g.ID] = g
	}
	b, err = os.ReadFile("testdata/shots.json")
	if err != nil {
		t.Fatal(err)
	}
	ss, err := ParseShots(b)
	if err != nil {
		t.Fatal(err)
	}
	shots := map[string][]Shot{}
	for _, s := range ss {
		shots[s.GameID] = append(shots[s.GameID], s)
	}
	return games, shots
}

func TestParse(t *testing.T) {
	games, shots := load(t)
	g := games["Vj589JGmq8"]
	if !g.StartAt.Equal(time.Date(2025, 12, 6, 19, 30, 0, 0, time.UTC)) || !g.Knockout || !g.Finished() || g.Referee == "" {
		t.Fatalf("game: %+v", g)
	}
	if y, err := g.SeasonYear(); err != nil || y != 2025 {
		t.Fatalf("season %d %v", y, err)
	}
	if len(shots["Vj589JGmq8"]) != 18 {
		t.Fatalf("shots: %d", len(shots["Vj589JGmq8"]))
	}
	b, err := os.ReadFile("testdata/teams.json")
	if err != nil {
		t.Fatal(err)
	}
	ts, err := ParseTeams(b)
	if err != nil || len(ts) != 3 || ts[0].Name == "" {
		t.Fatalf("teams %v %v", ts, err)
	}
	if _, err := ParseGames([]byte(`{"error":"500 - Internal server error"}`)); err == nil {
		t.Fatal("want error body to fail")
	}
}

func TestResult(t *testing.T) {
	games, shots := load(t)
	sc := func(h, a int) soccer.Score { return soccer.Score{Home: h, Away: a} }
	cases := []struct {
		id       string
		h1, h2   soccer.Score
		et       *soccer.Score
		decided  string
		otType   string
		finalSum soccer.Score
	}{
		// MLS Cup 2025: first-half goal is not a shot (own goal), seen in the score fields.
		{id: "Vj589JGmq8", h1: sc(1, 0), h2: sc(2, 1), decided: "reg", otType: soccer.OTNone, finalSum: sc(3, 1)},
		// Knockout 2-2 after extra time, then penalties with an unknown winner.
		{id: "vzqoYE1ZMa", h1: sc(2, 0), h2: sc(0, 2), et: &soccer.Score{}, decided: "so", otType: soccer.OTExtra, finalSum: sc(2, 2)},
		// Regular season: an own goal is missing from the shots; the game score wins.
		{id: "7vQ7bgGzqD", h1: sc(1, 0), h2: sc(1, 2), decided: "reg", otType: soccer.OTNone, finalSum: sc(2, 2)},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			r, ot, ok, err := Result(games[c.id], shots[c.id])
			if err != nil || !ok {
				t.Fatalf("ok=%v err=%v", ok, err)
			}
			if r.Periods[0] != c.h1 || r.Periods[1] != c.h2 || r.DecidedIn != c.decided || ot != c.otType || r.Final != c.finalSum {
				t.Fatalf("got %+v ot=%s", r, ot)
			}
			if (c.et == nil) != (r.OT == nil) || (c.et != nil && *r.OT != *c.et) {
				t.Fatalf("et %v want %v", r.OT, c.et)
			}
		})
	}
}

func TestResultInconsistent(t *testing.T) {
	g := Game{ID: "x", HomeTeam: "h", AwayTeam: "a", HomeScore: 0, AwayScore: 1, Status: "FullTime"}
	// Shots claim 1-0 at half time, the game ended 0-1: halves cannot be reconciled.
	ss := []Shot{
		{GameID: "x", Period: 1, ExpandedMinute: 10, TeamID: "h", Goal: 1},
		{GameID: "x", Period: 2, ExpandedMinute: 50, TeamID: "a", HomeScore: 1},
	}
	if _, _, ok, err := Result(g, ss); ok || err != nil {
		t.Fatalf("want not ok, got ok=%v err=%v", ok, err)
	}
	// No shots at all in a goalless game: 0-0 halves.
	g.AwayScore = 0
	r, _, ok, err := Result(g, nil)
	if !ok || err != nil || r.Periods[0] != (soccer.Score{}) || r.Periods[1] != (soccer.Score{}) {
		t.Fatalf("goalless: %+v ok=%v err=%v", r, ok, err)
	}
}
