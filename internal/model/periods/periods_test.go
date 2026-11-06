package periods

import (
	"math"
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

func TestFitRecoversTruth(t *testing.T) {
	p := DefaultSim()
	p.Seasons = 4
	games, truth := Simulate(p)
	asOf := games[len(games)-1].StartAt.Add(24 * time.Hour)
	cfg := DefaultConfig()
	cfg.HalfLifeDays = 10000 // static truth: no decay
	cfg.Ridge = 1
	m, err := Fit(games, asOf, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(m.HomeAdv-p.HomeAdv) > 0.04 {
		t.Errorf("home adv %.3f want %.3f", m.HomeAdv, p.HomeAdv)
	}
	if math.Abs(m.Mu-p.Mu) > 0.05 {
		t.Errorf("mu %.3f want %.3f", m.Mu, p.Mu)
	}
	var se, n float64
	for id, a := range truth.Att {
		// Identifiable up to a shift absorbed by μ; truth is centred near 0.
		se += math.Pow(m.Att[id]-a, 2) + math.Pow(m.Def[id]-truth.Def[id], 2)
		n += 2
	}
	if rmse := math.Sqrt(se / n); rmse > 0.06 {
		t.Errorf("att/def rmse %.3f", rmse)
	}
	if m.Profile[2] < m.Profile[0] {
		t.Errorf("profile %v: third period should carry most goals", m.Profile)
	}
	// Simulated regulation ties are plain Poisson: δ should be ~0.
	if math.Abs(m.Delta) > 0.15 {
		t.Errorf("delta %.3f", m.Delta)
	}
	if m.OTBeta <= 0 {
		t.Errorf("stronger team should win OT more often: beta %.3f", m.OTBeta)
	}
}

func TestPredictConsistent(t *testing.T) {
	games, _ := Simulate(DefaultSim())
	asOf := games[len(games)-1].StartAt.Add(time.Hour)
	m, err := Fit(games, asOf, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	pred, err := m.Predict(canon.Match{ID: 99, HomeTeam: 1, AwayTeam: 2})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(pred.Matrix.Sum()-1) > 1e-9 {
		t.Fatal("matrix mass")
	}
	x := pred.Markets["1x2"]
	if math.Abs(x["home"]+x["draw"]+x["away"]-1) > 1e-9 || x["draw"] < 0.15 || x["draw"] > 0.3 {
		t.Fatalf("1x2 %v", x)
	}
	ml := pred.Markets["ml"]
	if math.Abs(ml["home"]+ml["away"]-1) > 1e-9 {
		t.Fatal("ml mass")
	}
	// Unknown team = league average.
	if _, err := m.Predict(canon.Match{HomeTeam: 500, AwayTeam: 501}); err != nil {
		t.Fatal(err)
	}
	if _, err := Fit(games, games[0].StartAt, DefaultConfig()); err != ErrNoData {
		t.Fatalf("no games before asOf: %v", err)
	}
}

func TestTieInflationFitsExtraDraws(t *testing.T) {
	games, _ := Simulate(DefaultSim())
	// Force extra ties: turn every 6th one-goal game into a tie.
	for i := range games {
		g := &games[i]
		if i%6 == 0 && g.Reg.Home-g.Reg.Away == 1 {
			g.Periods[2].Home--
			if g.Periods[2].Home < 0 {
				g.Periods[2].Home = 0
				continue
			}
			g.Reg.Home--
		}
	}
	m, err := Fit(games, games[len(games)-1].StartAt.Add(time.Hour), DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if m.Delta < 0.1 {
		t.Fatalf("delta should pick up extra ties: %.3f", m.Delta)
	}
}
