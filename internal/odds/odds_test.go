package odds

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestNormalizeAPIHockey(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	cases := []struct {
		bet, val string
		market   string
		sel      string
		line     *float64
	}{
		{"3Way Result", "Home", OneX2, "home", nil},
		{"3Way Result", "Draw", OneX2, "draw", nil},
		{"Home/Away", "Away", ML, "away", nil},
		{"Over/Under", "Over 5.5", Total, "over", f(5.5)},
		{"Over/Under 1st Period", "Under 1.5", P1Total, "under", f(1.5)},
		{"3Way Result 1st Period", "Away", P1OneX2, "away", nil},
		{"Asian Handicap", "Home -1.5", PuckLine, "home", f(-1.5)},
		{"Asian Handicap", "Away +1.5", PuckLine, "away", f(-1.5)},
		{"Correct Score", "3:2", CorrectScore, "3-2", nil},
		{"Total - Home", "Over 2.5", Other, "Over 2.5", nil},
		{"Over/Under 2nd Period", "Over 1.5", Other, "Over 1.5", nil},
		{"Highest Scoring Period", "1st Period", Other, "1st Period", nil},
		{"3Way Result", "Nobody", Other, "Nobody", nil},
		// OddsPapi names.
		{"Regular Time Result fulltime", "Home", OneX2, "home", nil},
		{"First Period Result", "Draw", P1OneX2, "draw", nil},
		{"Winner (incl. overtime and penalties) result", "Away", ML, "away", nil},
		{"Total (incl. overtime and penalties)", "Over 5.5", Other, "Over 5.5", nil},
		{"Correct Score (incl. overtime and penalties)", "3:2", Other, "3:2", nil},
	}
	for _, c := range cases {
		q := NormalizeAPIHockey(c.bet, c.val, 2)
		if q.Market != c.market || q.Selection != c.sel || (q.Line == nil) != (c.line == nil) || (q.Line != nil && *q.Line != *c.line) {
			t.Errorf("%s / %s: got %s %s %v", c.bet, c.val, q.Market, q.Selection, q.Line)
		}
	}
}

func TestParsePrice(t *testing.T) {
	for raw, want := range map[string]float64{`"1.85"`: 1.85, `2.1`: 2.1, `"3,40"`: 3.4} {
		got, err := ParsePrice(json.RawMessage(raw))
		if err != nil || got != want {
			t.Errorf("%s: %v %v", raw, got, err)
		}
	}
	if _, err := ParsePrice(json.RawMessage(`"1.00"`)); err == nil {
		t.Error("1.00 must be rejected")
	}
}

func TestMargin(t *testing.T) {
	if m := Margin(2, 2); math.Abs(m) > 1e-9 {
		t.Fatal(m)
	}
	if m := Margin(2.2, 4.0, 2.7); math.Abs(m-(1/2.2+1/4.0+1/2.7-1)) > 1e-12 || m < 0.07 {
		t.Fatal(m)
	}
}

func TestSnapshotKind(t *testing.T) {
	start := time.Date(2026, 11, 5, 19, 0, 0, 0, time.UTC)
	cases := map[time.Duration]string{24 * time.Hour: "t24h", time.Hour: "t1h", 5 * time.Minute: "closing", 6 * time.Hour: "adhoc", -time.Minute: "adhoc"}
	for before, want := range cases {
		if got := SnapshotKind(start.Add(-before), start); got != want {
			t.Errorf("%v before: %s want %s", before, got, want)
		}
	}
}
