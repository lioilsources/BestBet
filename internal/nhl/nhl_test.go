package nhl

import (
	"os"
	"testing"
)

func TestParseScore(t *testing.T) {
	b, err := os.ReadFile("testdata/score_2025-11-01.json")
	if err != nil {
		t.Fatal(err)
	}
	games, err := ParseScore(b)
	if err != nil || len(games) != 13 {
		t.Fatalf("%d %v", len(games), err)
	}
	want := map[int64]struct {
		decided  string
		rh, ra   int
		fh, fa   int
		soWinner string
	}{
		2025020181: {"reg", 2, 1, 2, 1, ""},
		2025020184: {"ot", 2, 2, 3, 2, ""},
		2025020185: {"so", 3, 3, 4, 3, "home"},
	}
	for _, g := range games {
		if g.Status() != "finished" || g.Season != 2025 {
			t.Fatalf("%d status %s season %d", g.ID, g.Status(), g.Season)
		}
		if ph, ok := g.Phase(); !ok || ph != "regular" {
			t.Fatalf("phase %s", ph)
		}
		r, err := g.Result()
		if err != nil {
			t.Fatal(err)
		}
		w, ok := want[g.ID]
		if !ok {
			continue
		}
		if r.DecidedIn != w.decided || r.Reg.Home != w.rh || r.Reg.Away != w.ra || r.Final.Home != w.fh || r.Final.Away != w.fa || r.SOWinner != w.soWinner {
			t.Errorf("%d: %+v", g.ID, r)
		}
		if len(r.Periods) != 3 {
			t.Errorf("%d periods %v", g.ID, r.Periods)
		}
	}
	if games[0].Home.Name == "" || games[0].Home.Abbrev != "BOS" {
		t.Fatalf("team %+v", games[0].Home)
	}
}

func TestParseBoxscore(t *testing.T) {
	b, err := os.ReadFile("testdata/boxscore_2025020180.json")
	if err != nil {
		t.Fatal(err)
	}
	gs, err := ParseBoxscore(b)
	if err != nil {
		t.Fatal(err)
	}
	starters := 0
	for _, g := range gs {
		if g.Starter {
			starters++
			if g.ShotsAgainst == 0 || g.TOISeconds < 3000 || g.Name == "" {
				t.Fatalf("starter %+v", g)
			}
		}
	}
	if starters != 2 {
		t.Fatalf("%+v", gs)
	}
}
