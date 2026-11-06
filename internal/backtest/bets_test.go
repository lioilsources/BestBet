package backtest

import (
	"math"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/periods"
)

// simOdds prices every game: the sharp book closes at the true probabilities
// (2.5 % margin); a soft book quotes noisy prices 2 h before the start (6 %).
func simOdds(games []canon.Match, sim periods.SimParams, truth periods.SimTruth, seed uint64) map[int64][]canon.OddsQuote {
	r := rand.New(rand.NewPCG(seed, 3))
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
		var soft [3]float64
		tot := 0.0
		for i := range p {
			soft[i] = p[i] * math.Exp(0.15*r.NormFloat64())
			tot += soft[i]
		}
		for i, sel := range sels1x2 {
			odds[g.ID] = append(odds[g.ID],
				canon.OddsQuote{Book: "sharp", Market: "1x2", Selection: sel, Price: 1 / (p[i] * 1.025), CapturedAt: g.StartAt.Add(-5 * time.Minute)},
				canon.OddsQuote{Book: "soft", Market: "1x2", Selection: sel, Price: 1 / (soft[i] / tot * 1.06), CapturedAt: g.StartAt.Add(-2 * time.Hour)},
			)
		}
	}
	return odds
}

func TestBetsBeatTheCloseWhenTheModelIsRight(t *testing.T) {
	sim := periods.DefaultSim()
	games, truth := periods.Simulate(sim)
	odds := simOdds(games, sim, truth, 1)
	from := games[0].StartAt.AddDate(1, 0, 0)
	rows, err := Run(games, odds, fitPeriods, Config{From: from, To: games[len(games)-1].StartAt.Add(24 * time.Hour), MinTrain: 100})
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultBetConfig()
	cfg.Sharp, cfg.Books = "sharp", []string{"soft"}
	bets := SimulateBets(rows, odds, cfg)
	s := SummariseBets(bets, 1)
	if s.N < 100 || s.NCLV != s.N {
		t.Fatalf("bets %d clv %d", s.N, s.NCLV)
	}
	// The soft book's noise is what the model exploits: CLV against a true close is positive.
	if !(s.CLVLo > 0) {
		t.Fatalf("CLV %.4f (%.4f..%.4f) should be significantly positive", s.MeanCLV, s.CLVLo, s.CLVHi)
	}
	for _, b := range bets {
		if b.Stake > cfg.MaxStake+1e-12 || b.Stake <= 0 {
			t.Fatalf("stake %.4f outside (0, cap]", b.Stake)
		}
	}
	perDay := map[string]int{}
	for _, b := range bets {
		perDay[b.Match.StartAt.Format("2006-01-02")]++
	}
	for d, n := range perDay {
		if n > cfg.MaxPerDay {
			t.Fatalf("%s: %d bets", d, n)
		}
	}
	md := RenderGate(GateInput{Rows: rows, Bets: bets, Holdout: 2023, Sharp: "sharp", Odds: odds, Ablation: "test"})
	if !strings.Contains(md, "CLV") || strings.Contains(md, "chybí (žádné sázky") {
		t.Fatal(md)
	}
	if c, n := MarketMove(rows, odds, "sharp"); n != 0 || c != 0 {
		t.Fatalf("one sharp capture per game has no move: %.3f %d", c, n)
	}
}

func TestGateWithoutOddsSaysMissing(t *testing.T) {
	md := RenderGate(GateInput{Holdout: 2025, Sharp: "pinnacle"})
	if strings.Count(md, "chybí") != 3 {
		t.Fatal(md)
	}
}
