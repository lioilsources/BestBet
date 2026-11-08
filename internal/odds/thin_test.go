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

func TestThinAtMarksProvisional(t *testing.T) {
	start := time.Date(2026, 11, 9, 1, 0, 0, 0, time.UTC)
	pts := []Point{
		{Key: "k", At: start.Add(-48 * time.Hour), I: 0},
		{Key: "k", At: start.Add(-30 * time.Hour), I: 1},
		{Key: "k", At: start.Add(-10 * time.Hour), I: 2},
	}
	got := ThinAt(pts, start, start.Add(-5*time.Hour))
	if got[0] != "opening" || got[1] != "t24h" || got[2] != "adhoc" {
		t.Fatalf("%v", got)
	}
	if got := ThinAt(pts, start, start.Add(time.Hour)); got[2] != "closing" {
		t.Fatalf("after start: %v", got)
	}
}
