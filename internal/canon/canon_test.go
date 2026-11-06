package canon

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestJSONRoundTrip(t *testing.T) {
	line := 5.5
	start := time.Date(2026, 2, 1, 18, 0, 0, 0, time.UTC)
	values := []any{
		&Match{
			ID: 1, League: "swe-allsvenskan", Season: 2025, StartAt: start, HomeTeam: 3, AwayTeam: 7, Phase: "regular",
			Periods: []Score{{1, 0}, {0, 1}, {1, 1}}, Reg: &Score{2, 2}, Final: &Score{3, 2}, DecidedIn: "so",
			Goalies: []GoalieStart{{Side: Home, Player: "Adam Werner", Starter: true, ShotsAgainst: 30, Saves: 28, TOISeconds: 3900, Confirm: "CONFIRMED"}},
			Lineups: []Lineup{{Side: Away, Players: []PlayerMinutes{{PlayerID: 9, Position: "C", LineNo: 1}}, Absent: []int32{12}}},
		},
		&OddsQuote{Book: "pinnacle", Market: "total", Selection: "over", Line: &line, Price: 1.91, CapturedAt: start, Kind: "closing"},
		&Prediction{
			MatchID: 1, ModelVersion: "periods-v1", InputsHash: "abc", ComputedAt: start, LambdaHome: 3.1, LambdaAway: 2.6,
			Matrix: ScoreMatrix{{0.1, 0.2}, {0.3, 0.4}}, OTHomeWin: 0.52,
			Markets:       MarketProbs{"1x2": {"home": 0.45, "draw": 0.22, "away": 0.33}},
			Contributions: []Contribution{{Input: "goalie", Side: Home, PP: 2.3}},
		},
	}
	for _, v := range values {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		back := reflect.New(reflect.TypeOf(v).Elem()).Interface()
		if err := json.Unmarshal(b, back); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(v, back) {
			t.Errorf("round trip changed %T:\n%+v\n%+v", v, v, back)
		}
	}
}

func TestHashInputsStable(t *testing.T) {
	a, _ := HashInputs(map[string]any{"b": 1, "a": []int{1, 2}})
	b, _ := HashInputs(map[string]any{"a": []int{1, 2}, "b": 1})
	c, _ := HashInputs(map[string]any{"a": []int{2, 1}, "b": 1})
	if a != b || a == c || len(a) != 24 {
		t.Fatalf("%s %s %s", a, b, c)
	}
}

func TestPlayedAndSort(t *testing.T) {
	t0 := time.Unix(100, 0)
	ms := []Match{{ID: 2, StartAt: t0}, {ID: 1, StartAt: t0}, {ID: 3, StartAt: t0.Add(-time.Hour)}}
	SortMatches(ms)
	if ms[0].ID != 3 || ms[1].ID != 1 || ms[2].ID != 2 {
		t.Fatal(ms)
	}
	if (Match{}).Played() {
		t.Fatal("empty match is not played")
	}
}
