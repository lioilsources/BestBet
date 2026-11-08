package markets

import (
	"math"
	"testing"

	"github.com/lioilsources/BestBet/internal/canon"
)

func approx(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %.12f, want %.12f", name, got, want)
	}
}

func TestDerive(t *testing.T) {
	// Toy 60-min distribution over 0..2 goals.
	reg := canon.ScoreMatrix{
		{0.10, 0.05, 0.05},
		{0.15, 0.10, 0.05},
		{0.20, 0.10, 0.20},
	}
	m := Derive(reg, reg, 0.6)
	home := 0.15 + 0.20 + 0.10
	draw := 0.10 + 0.10 + 0.20
	away := 0.05 + 0.05 + 0.05
	approx(t, "home", m[OneX2]["home"], home)
	approx(t, "draw", m[OneX2]["draw"], draw)
	approx(t, "ml home", m[ML]["home"], home+0.6*draw)
	approx(t, "ml sum", m[ML]["home"]+m[ML]["away"], 1)
	approx(t, "dnb", m[DNB]["home"], home/(home+away))
	approx(t, "over 3.5", m[Total]["over@3.5"], 0.20)
	approx(t, "under 3.5", m[Total]["under@3.5"], 0.80)
	// Home -1.5 needs a 2-goal margin: only 2-0.
	approx(t, "home -1.5", m[PuckLine]["home@-1.5"], 0.20)
	approx(t, "away -1.5 (home +1.5)", m[PuckLine]["away@1.5"], 0.05) // away by 2+: only 0-2
	approx(t, "home +1.5", m[PuckLine]["home@1.5"], 1-0.05)
	approx(t, "cs 2-2", m[CorrectScore]["2-2"], 0.20)
	approx(t, "btts", m[BTTS]["yes"], 0.10+0.05+0.10+0.20)
	approx(t, "p1 total over 0.5", m[P1Total]["over@0.5"], 0.9)
	if Key("over", 5.5) != "over@5.5" || Key("home", -1) != "home@-1" {
		t.Error("keys")
	}
}
