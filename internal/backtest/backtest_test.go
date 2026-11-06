package backtest

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/periods"
)

func fitPeriods(train []canon.Match, asOf time.Time) (Predictor, error) {
	return periods.Fit(train, asOf, periods.DefaultConfig())
}

// End-to-end harness check without network: on a synthetic league the model
// beats base rates, and a "market" that knows the true probabilities beats the model.
func TestHarnessOnSyntheticLeague(t *testing.T) {
	sim := periods.DefaultSim()
	games, truth := periods.Simulate(sim)
	from := games[0].StartAt.AddDate(1, 0, 0) // first season is warm-up
	to := games[len(games)-1].StartAt.Add(24 * time.Hour)

	// Oracle market: true 1X2 from the generating parameters, priced with a 6 % margin.
	odds := map[int64][]canon.OddsQuote{}
	for _, g := range games {
		lh := math.Exp(sim.Mu + sim.HomeAdv + truth.Att[g.HomeTeam] - truth.Def[g.AwayTeam])
		la := math.Exp(sim.Mu + truth.Att[g.AwayTeam] - truth.Def[g.HomeTeam])
		mx := periods.Matrix(lh, la, 12, 0)
		var p [3]float64
		for h := range mx {
			for a, v := range mx[h] {
				switch {
				case h > a:
					p[0] += v
				case h == a:
					p[1] += v
				default:
					p[2] += v
				}
			}
		}
		for i, sel := range []string{"home", "draw", "away"} {
			odds[g.ID] = append(odds[g.ID], canon.OddsQuote{Book: "oracle", Market: "1x2", Selection: sel, Price: 1 / (p[i] * 1.06), CapturedAt: g.StartAt.Add(-time.Minute), Kind: "closing"})
		}
	}

	rows, err := Run(games, odds, fitPeriods, Config{From: from, To: to, MinTrain: 100})
	if err != nil {
		t.Fatal(err)
	}
	m, err := Summarise(rows)
	if err != nil {
		t.Fatal(err)
	}
	if m.N < 500 || m.NMarket1X2 != m.N {
		t.Fatalf("rows %d market %d", m.N, m.NMarket1X2)
	}
	if !(m.LogLoss1X2 < m.BaseLogLoss1X2) {
		t.Errorf("model %.4f should beat base rates %.4f", m.LogLoss1X2, m.BaseLogLoss1X2)
	}
	if !(m.MarketLogLoss1X2 <= m.ModelLLOnMarket) {
		t.Errorf("oracle market %.4f should beat the model %.4f", m.MarketLogLoss1X2, m.ModelLLOnMarket)
	}
	// Harness sanity (01 fáze 1 akceptace, by analogy): a correct model is close to the oracle.
	if gap := (m.ModelLLOnMarket - m.MarketLogLoss1X2) / m.MarketLogLoss1X2; gap > 0.02 {
		t.Errorf("model is %.1f %% worse than the oracle: harness or model bug", 100*gap)
	}
	// No look-ahead: every prediction was computed before its game.
	for _, r := range rows {
		if !r.Pred.ComputedAt.Before(r.Match.StartAt.Add(time.Nanosecond)) {
			t.Fatalf("prediction for %d computed at %v after start %v", r.Match.ID, r.Pred.ComputedAt, r.Match.StartAt)
		}
	}
}

func TestBenjaminiHochberg(t *testing.T) {
	got := BenjaminiHochberg([]float64{0.001, 0.04, 0.03, 0.5}, 0.05)
	want := []bool{true, false, false, false}
	// ranks: 0.001 (≤0.0125 ✓), 0.03 (≤0.025 ✗), 0.04 (≤0.0375 ✗), 0.5
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%v", got)
		}
	}
	got = BenjaminiHochberg([]float64{0.01, 0.02, 0.03}, 0.05)
	if !got[0] || !got[1] || !got[2] {
		t.Fatalf("all should pass: %v", got)
	}
}

func TestPairedDelta(t *testing.T) {
	sim := periods.DefaultSim()
	games, _ := periods.Simulate(sim)
	from := games[0].StartAt.AddDate(1, 0, 0)
	to := games[len(games)-1].StartAt.Add(24 * time.Hour)
	base, err := Run(games, nil, fitPeriods, Config{From: from, To: to, MinTrain: 100})
	if err != nil {
		t.Fatal(err)
	}
	same := PairedDelta(base, base)
	if d := same[0]; d.N != len(base) || d.MeanDelta1X2 != 0 || d.CILo != 0 {
		t.Fatalf("identical runs: %+v", d)
	}
	// A deliberately worse variant (no tie inflation, short memory) must show Δ > 0.
	worse := func(train []canon.Match, asOf time.Time) (Predictor, error) {
		c := periods.DefaultConfig()
		c.HalfLifeDays = 15
		c.Ridge = 0.01
		return periods.Fit(train, asOf, c)
	}
	v, err := Run(games, nil, worse, Config{From: from, To: to, MinTrain: 100})
	if err != nil {
		t.Fatal(err)
	}
	d := PairedDelta(base, v)[0]
	if !(d.MeanDelta1X2 > 0 && d.CILo > 0) {
		t.Fatalf("worse variant should be significantly worse: %+v", d)
	}
	if md := RenderAblation("x", PairedDelta(base, v), 2023); !strings.Contains(md, "(odložená)") {
		t.Fatal(md)
	}
}
