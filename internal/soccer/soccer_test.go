package soccer

import "testing"

func sc(h, a int) Score { return Score{Home: h, Away: a} }

func TestBuildResult(t *testing.T) {
	et := sc(1, 0)
	cases := []struct {
		name     string
		h1, h2   Score
		et       *Score
		shootout bool
		winner   string
		reg, fin Score
		decided  string
		wantErr  bool
	}{
		{name: "regular", h1: sc(1, 0), h2: sc(1, 2), reg: sc(2, 2), fin: sc(2, 2), decided: "reg"},
		{name: "extra time", h1: sc(0, 1), h2: sc(1, 0), et: &et, reg: sc(1, 1), fin: sc(2, 1), decided: "ot"},
		{name: "straight to penalties, winner unknown", h1: sc(0, 0), h2: sc(2, 2), shootout: true, reg: sc(2, 2), fin: sc(2, 2), decided: "so"},
		{name: "penalties, away wins", h1: sc(0, 0), h2: sc(0, 0), shootout: true, winner: "away", reg: sc(0, 0), fin: sc(0, 1), decided: "so"},
		{name: "extra time after a winner", h1: sc(1, 0), h2: sc(0, 0), et: &et, wantErr: true},
		{name: "penalties after a winner", h1: sc(1, 0), h2: sc(0, 0), shootout: true, wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := BuildResult(c.h1, c.h2, c.et, c.shootout, c.winner)
			if c.wantErr {
				if err == nil {
					t.Fatalf("want error, got %+v", r)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if r.Reg != c.reg || r.Final != c.fin || r.DecidedIn != c.decided || len(r.Periods) != 2 {
				t.Fatalf("got reg=%v final=%v decided=%s periods=%d", r.Reg, r.Final, r.DecidedIn, len(r.Periods))
			}
		})
	}
}

func TestSettleAH(t *testing.T) {
	cases := []struct {
		line float64
		s    Score
		want Settlement
	}{
		{0, sc(1, 1), 0},
		{0, sc(2, 1), 1},
		{-0.5, sc(1, 1), -1},
		{-0.25, sc(1, 1), -0.5}, // half on 0 (push), half on -0.5 (loss)
		{-0.25, sc(2, 1), 1},
		{-0.75, sc(2, 1), 0.5}, // half on -0.5 (win), half on -1 (push)
		{-0.75, sc(3, 1), 1},
		{-1, sc(2, 1), 0},
		{+0.25, sc(1, 1), 0.5},
		{+0.75, sc(0, 1), -0.5},
		{-1.25, sc(2, 1), -0.5},
	}
	for _, c := range cases {
		if got := SettleAH(c.line, c.s); got != c.want {
			t.Errorf("AH %v on %v: got %v want %v", c.line, c.s, got, c.want)
		}
	}
}

func TestSettleOver(t *testing.T) {
	cases := []struct {
		line float64
		s    Score
		want Settlement
	}{
		{2.5, sc(2, 1), 1},
		{2.5, sc(1, 1), -1},
		{2.25, sc(1, 1), -0.5}, // half on 2 (push), half on 2.5 (loss)
		{2.75, sc(2, 1), 0.5},  // half on 2.5 (win), half on 3 (push)
		{3, sc(2, 1), 0},
	}
	for _, c := range cases {
		if got := SettleOver(c.line, c.s); got != c.want {
			t.Errorf("over %v on %v: got %v want %v", c.line, c.s, got, c.want)
		}
	}
	for _, l := range []float64{0.25, -0.75, 2.25, 2.75} {
		if !QuarterLine(l) {
			t.Errorf("%v is a quarter line", l)
		}
	}
	for _, l := range []float64{0, 0.5, -1, 2.5, 3} {
		if QuarterLine(l) {
			t.Errorf("%v is not a quarter line", l)
		}
	}
}
