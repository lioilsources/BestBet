package oddspapi

import (
	"testing"
	"time"
)

func TestParseHistorical(t *testing.T) {
	b := []byte(`{"fixtureId":"id1","bookmakers":{"pinnacle":{"markets":{"101":{"outcomes":{
		"101":{"players":{"0":[{"createdAt":"2026-02-01T10:00:00Z","price":2.1,"active":true},{"createdAt":"2026-02-01T09:00:00Z","price":"2.05"}]}},
		"103":{"players":{"0":[{"createdAt":"2026-02-01T09:00:00Z","price":1}]}}}}}}}}`)
	pts, err := ParseHistorical(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 2 || pts[0].Price != 2.05 || pts[1].Price != 2.1 || pts[0].Book != "pinnacle" || pts[0].MarketID != "101" {
		t.Fatalf("%+v", pts)
	}
}

func TestParseFixturesAndNormalize(t *testing.T) {
	fx, err := ParseFixtures([]byte(`[{"fixtureId":"id9","participant1Name":"MoDo Hockey","participant2Name":"Östersunds IK","startTime":"2026-02-01T18:00:00Z","statusId":2}]`))
	if err != nil || len(fx) != 1 || fx[0].Home != "MoDo Hockey" || fx[0].StatusID != 2 {
		t.Fatalf("%+v %v", fx, err)
	}
	mk, err := ParseMarkets([]byte(`[
		{"marketId":101,"marketName":"1X2","outcomes":[{"outcomeId":101,"outcomeName":"1"},{"outcomeId":102,"outcomeName":"X"},{"outcomeId":103,"outcomeName":"2"}]},
		{"marketId":1010,"marketName":"Over/Under","handicap":5.5,"outcomes":[{"outcomeId":1,"outcomeName":"Over"},{"outcomeId":2,"outcomeName":"Under"}]},
		{"marketId":1020,"marketName":"Asian Handicap","handicap":-1.5,"outcomes":[{"outcomeId":1,"outcomeName":"1"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if q := Normalize(mk["101"], "102", 3.9); q.Market != "1x2" || q.Selection != "draw" {
		t.Errorf("1x2: %+v", q)
	}
	if q := Normalize(mk["1010"], "1", 1.9); q.Market != "total" || q.Selection != "over" || *q.Line != 5.5 {
		t.Errorf("total: %+v", q)
	}
	if q := Normalize(mk["1020"], "1", 2.6); q.Market != "puck_line" || q.Selection != "home" || *q.Line != -1.5 {
		t.Errorf("puck line: %+v", q)
	}
}

func TestRetryAfter(t *testing.T) {
	if d := retryAfter(`"details":"Please wait 3.95 seconds before making another request"`); d != 4950*time.Millisecond {
		t.Fatalf("got %v", d)
	}
	if d := retryAfter("http 429"); d != 5*time.Second {
		t.Fatalf("got %v", d)
	}
}
