package odds

import (
	"testing"
	"time"
)

func TestThinKeepsGateRoles(t *testing.T) {
	start := time.Date(2026, 11, 1, 18, 0, 0, 0, time.UTC)
	at := func(h float64) time.Time { return start.Add(time.Duration(h * float64(time.Hour))) }
	pts := []Point{
		{"a", at(-72), 0}, {"a", at(-48), 1}, {"a", at(-21), 2}, {"a", at(-10), 3},
		{"a", at(-1.5), 4}, {"a", at(-0.5), 5}, {"a", at(-0.1), 6}, {"a", at(0.5), 7},
		{"b", at(-30), 8},
	}
	got := Thin(pts, start)
	want := map[int]string{0: "opening", 2: "t24h", 4: "t1h", 6: "closing", 8: "closing"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i, k := range want {
		if got[i] != k {
			t.Fatalf("point %d: got %q want %q (%v)", i, got[i], k, got)
		}
	}
}
