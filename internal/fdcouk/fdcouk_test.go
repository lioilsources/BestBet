package fdcouk

import (
	"os"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	b, err := os.ReadFile("testdata/usa.csv")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows: %d", len(rows))
	}
	r := rows[0]
	if r.League != "MLS" || r.Season != 2012 || r.Home != "Colorado Rapids" || r.HG != 2 || r.AG != 0 ||
		!r.StartAt.Equal(time.Date(2012, 3, 10, 23, 0, 0, 0, time.UTC)) {
		t.Fatalf("row 0: %+v", r)
	}
	if p := r.Closing["pinnacle"]; p != [3]float64{2.06, 3.4, 4.11} {
		t.Fatalf("pinnacle: %v", p)
	}
	if _, ok := r.Closing["betfair_ex"]; ok {
		t.Fatal("2012 has no exchange prices")
	}
	if len(rows[1].Closing) != 5 {
		t.Fatalf("2025 final: %v", rows[1].Closing)
	}
	if _, ok := rows[2].Closing["pinnacle"]; ok || len(rows[2].Closing) != 4 {
		t.Fatalf("missing Pinnacle row: %v", rows[2].Closing)
	}
	if rows[1].FixtureID() != "2025-11-23T02:30|Vancouver Whitecaps|Los Angeles FC" {
		t.Fatal(rows[1].FixtureID())
	}
}
