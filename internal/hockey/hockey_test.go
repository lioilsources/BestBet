package hockey

import (
	"testing"
	"time"
)

func sc(h, a int) Score   { return Score{h, a} }
func psc(h, a int) *Score { s := Score{h, a}; return &s }

func TestBuildResult(t *testing.T) {
	cases := []struct {
		name      string
		periods   []Score
		ot, so    *Score
		final     Score
		reg       Score
		decided   string
		soWinner  string
		wantError bool
	}{
		{name: "regulation", periods: []Score{sc(0, 0), sc(1, 1), sc(2, 0)}, final: sc(3, 1), reg: sc(3, 1), decided: "reg"},
		{name: "ot goal", periods: []Score{sc(1, 0), sc(0, 1), sc(0, 0)}, ot: psc(0, 1), final: sc(1, 2), reg: sc(1, 1), decided: "ot"},
		{name: "shootout", periods: []Score{sc(0, 0), sc(0, 1), sc(1, 0)}, ot: psc(0, 0), so: psc(0, 1), final: sc(1, 2), reg: sc(1, 1), decided: "so", soWinner: "away"},
		{name: "shootout tally", periods: []Score{sc(2, 2), sc(0, 0), sc(0, 0)}, so: psc(3, 2), final: sc(3, 2), reg: sc(2, 2), decided: "so", soWinner: "home"},
		{name: "ot after decided", periods: []Score{sc(1, 0), sc(0, 0), sc(0, 0)}, ot: psc(1, 0), wantError: true},
		{name: "two periods", periods: []Score{sc(1, 0), sc(0, 0)}, wantError: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := BuildResult(c.periods, c.ot, c.so)
			if c.wantError {
				if err == nil {
					t.Fatalf("want error, got %+v", r)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if r.Final != c.final || r.Reg != c.reg || r.DecidedIn != c.decided || r.SOWinner != c.soWinner {
				t.Fatalf("got final=%v reg=%v decided=%s so=%s", r.Final, r.Reg, r.DecidedIn, r.SOWinner)
			}
		})
	}
}

func TestOTType(t *testing.T) {
	f := OTFormat{Regular: "3v3_5", Playoff: "5v5_20"}
	r, _ := BuildResult([]Score{sc(1, 1), sc(0, 0), sc(0, 0)}, psc(1, 0), nil)
	if got := OTType(r, PhaseRegular, f); got != "3v3_5" {
		t.Fatalf("regular ot: %s", got)
	}
	if got := OTType(r, PhasePlayoff, f); got != "5v5_20" {
		t.Fatalf("playoff ot: %s", got)
	}
	if got := OTType(r, PhasePlayoff, OTFormat{}); got != "unknown" {
		t.Fatalf("no format: %s", got)
	}
}

func TestDetectPhase(t *testing.T) {
	cases := map[string]string{
		"Regular Season":       PhaseRegular,
		"":                     PhaseRegular,
		"Play Offs":            PhasePlayoff,
		"Quarter-finals":       PhasePlayoff,
		"Slutspel HockeyAllsv": PhasePlayoff,
		"Play Out":             PhaseRelegation,
		"Baráž o extraligu":    PhaseRelegation,
		"Kvalserien":           PhasePromotion,
	}
	for in, want := range cases {
		if got := DetectPhase(in); got != want {
			t.Errorf("DetectPhase(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestDeriveSeries(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 3, d, 19, 0, 0, 0, time.UTC) }
	i := func(v int32) *int32 { return &v }
	games := []SeriesGame{
		{ID: 2, HomeTeamID: 7, AwayTeamID: 3, StartAt: day(4), HomeScore: i(1), AwayScore: i(2)},
		{ID: 1, HomeTeamID: 3, AwayTeamID: 7, StartAt: day(2), HomeScore: i(4), AwayScore: i(1)},
		{ID: 3, HomeTeamID: 3, AwayTeamID: 7, StartAt: day(6)},
		{ID: 9, HomeTeamID: 5, AwayTeamID: 6, StartAt: day(2), HomeScore: i(0), AwayScore: i(1)},
		// Same pair weeks later: a new series.
		{ID: 4, HomeTeamID: 7, AwayTeamID: 3, StartAt: day(28), HomeScore: i(3), AwayScore: i(0)},
	}
	got := DeriveSeries("swe2:2025", games, 10*24*time.Hour)
	want := map[int64]SeriesState{
		1: {"swe2:2025:3-7:1", 1, "0-0"},
		2: {"swe2:2025:3-7:1", 2, "0-1"}, // home team 7 has 0 wins, away 3 has 1
		3: {"swe2:2025:3-7:1", 3, "2-0"},
		9: {"swe2:2025:5-6:1", 1, "0-0"},
		4: {"swe2:2025:3-7:2", 1, "0-0"},
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("game %d: got %+v want %+v", id, got[id], w)
		}
	}
}

func TestFold(t *testing.T) {
	if got := Fold("Östersunds IK Baráž Vålerenga"); got != "Ostersunds IK Baraz Valerenga" {
		t.Fatal(got)
	}
}
