package ingest

import (
	"testing"
	"time"
)

func TestTenures(t *testing.T) {
	d := func(n int) time.Time { return time.Date(2025, 10, n, 0, 0, 0, 0, time.UTC) }
	ts := Tenures([]CoachGame{
		{1, d(3), "B"}, {1, d(1), "A"}, {1, d(2), "A"}, {1, d(4), ""}, {1, d(5), "B"},
		{2, d(1), "C"},
	})
	if len(ts) != 3 {
		t.Fatalf("got %d tenures: %+v", len(ts), ts)
	}
	if ts[0].Coach != "A" || !ts[0].To.Equal(d(2)) || ts[1].Coach != "B" || !ts[1].From.Equal(d(3)) || !ts[1].To.Equal(d(5)) || ts[2].Team != 2 {
		t.Fatalf("unexpected tenures: %+v", ts)
	}
}
