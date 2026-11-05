package swehockey

import (
	"os"

	"github.com/lioilsources/BetWave/internal/hockey"

	"testing"
	"time"
)

func TestParseSchedule(t *testing.T) {
	page, err := os.ReadFile("testdata/schedule_18266.html")
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Europe/Stockholm")
	games, err := ParseSchedule(page, loc, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 6 {
		t.Fatalf("want 6 rows, got %d", len(games))
	}
	g := games[0]
	if g.GameID != 1005612 || g.Home != "MoDo Hockey" || g.Away != "Östersunds IK" {
		t.Fatalf("row 0: %+v", g)
	}
	if want := time.Date(2025, 9, 19, 17, 0, 0, 0, time.UTC); !g.StartAt.Equal(want) {
		t.Fatalf("start %v want %v", g.StartAt.UTC(), want)
	}
	if g.Result == nil || g.Result.DecidedIn != "reg" || g.Result.Reg.Home != 3 {
		t.Fatalf("row 0 result: %+v", g.Result)
	}
	// Second row has no date cell: the date carries over.
	if games[1].StartAt.Format("2006-01-02") != "2025-09-19" {
		t.Fatalf("carried date: %v", games[1].StartAt)
	}
	var so, ot int
	for _, g := range games {
		if g.ParseErr != "" {
			t.Errorf("game %d: %s", g.GameID, g.ParseErr)
			continue
		}
		switch g.Result.DecidedIn {
		case "so":
			so++
		case "ot":
			ot++
		}
	}
	if so == 0 || ot == 0 {
		t.Fatalf("fixture should contain OT and SO games: ot=%d so=%d", ot, so)
	}
}

func TestParseEvents(t *testing.T) {
	page, err := os.ReadFile("testdata/events_1005612.html")
	if err != nil {
		t.Fatal(err)
	}
	ev, err := ParseEvents(page)
	if err != nil {
		t.Fatal(err)
	}
	if ev.HomeAbbr != "MoDo" || ev.AwayAbbr != "ÖST" || ev.HomeName != "MoDo Hockey" || ev.AwayName != "Östersunds IK" {
		t.Fatalf("teams: %+v", ev)
	}
	if len(ev.Goalies) != 2 {
		t.Fatalf("goalies: %+v", ev.Goalies)
	}
	h := ev.Goalies[0]
	if h.Side != "home" || h.Name != "Adam Werner" || h.Jersey != 35 || h.Saves != 18 || h.ShotsAgainst != 19 || h.GoalsAgainst != 1 || !h.Starter {
		t.Fatalf("home goalie: %+v", h)
	}
	// Home goalie played the full 60 minutes.
	if h.TOISeconds != 60*60 {
		t.Fatalf("home toi %d", h.TOISeconds)
	}
	a := ev.Goalies[1]
	if a.Side != "away" || a.Name != "Carl Lidö" || a.TOISeconds != 55*60+45 /* pulled for the extra attacker */ || !a.Starter {
		t.Fatalf("away goalie: %+v", a)
	}
}

func TestParseSchedulePlayoff(t *testing.T) {
	page, err := os.ReadFile("testdata/schedule_playoff_19979.html")
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Europe/Stockholm")
	games, err := ParseSchedule(page, loc, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 3 {
		t.Fatalf("want 3, got %d", len(games))
	}
	g := games[0]
	if g.HasEvents || g.GameID != 90133001 || g.Home != "Mora IK" || g.Away != "Södertälje SK" || g.Label != "Åttondelsfinal 1" {
		t.Fatalf("%+v", g)
	}
	if g.Result == nil || g.Result.Final.Away != 2 || g.StartAt.Format("2006-01-02 15:04") != "2026-03-09 19:00" {
		t.Fatalf("result/time: %+v", g)
	}
	if games[1].Result == nil || games[1].Result.DecidedIn != "ot" {
		t.Fatalf("second game should be decided in OT: %+v", games[1])
	}
}

func TestResultFromPeriodsMultiOT(t *testing.T) {
	sc := func(h, a int) hockey.Score { return hockey.Score{Home: h, Away: a} }
	p := []hockey.Score{sc(1, 1), sc(0, 0), sc(1, 1), sc(0, 0), sc(0, 0), sc(1, 0)}
	r, err := resultFromPeriods(p, hockey.Score{Home: 3, Away: 2}, false)
	if err != nil || r.DecidedIn != "ot" || r.OT.Home != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := resultFromPeriods(p, hockey.Score{Home: 3, Away: 2}, true); err == nil {
		t.Fatal("6 scores with shootouts must fail")
	}
}
