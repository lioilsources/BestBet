package layers

import (
	"math"
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/periods"
)

func TestGoalieLayerFindsEffect(t *testing.T) {
	sim := periods.DefaultSim()
	sim.Seasons = 3
	sim.GoalieGap = 0.12 // starter vs backup: ~0.6 goals per game in total
	games, _ := periods.Simulate(sim)
	in := BuildInputs(games)
	m := walk(t, games, Config{Base: periods.DefaultConfig(), Layers: []string{Goalie}}, in)
	// Shrinkage dilutes adj, so the fitted β is > 1; it must be clearly positive.
	if m.Beta[Goalie] < 0.5 || m.N[Goalie] < 500 {
		t.Fatalf("beta %.3f n %d", m.Beta[Goalie], m.N[Goalie])
	}
	// Prediction with a backup in goal for the home side gives the away side more goals.
	var backupGame canon.Match
	for _, g := range games[len(games)/2:] {
		if g.Goalies[0].Player[len(g.Goalies[0].Player)-1] == '2' && g.Goalies[1].Player[len(g.Goalies[1].Player)-1] == '1' {
			backupGame = g
			break
		}
	}
	_, la := m.Lambdas(backupGame)
	_, laBase := m.Base.Lambdas(backupGame.HomeTeam, backupGame.AwayTeam)
	if adjH := in.Goalie[backupGame.ID][canon.Home].Adj(); !(adjH > 0 && la > laBase) {
		t.Fatalf("backup should raise away λ: adj %.3f la %.3f base %.3f", adjH, la, laBase)
	}
	p, err := m.Predict(backupGame)
	var goalie *canon.Contribution
	for i := range p.Contributions {
		if p.Contributions[i].Input == Goalie {
			goalie = &p.Contributions[i]
		}
	}
	if err != nil || len(p.Contributions) != 6 || goalie == nil || goalie.PP >= 0 || p.ModelVersion != "periods-v1+goalie" {
		t.Fatalf("home backup must cost the home side: %+v %v", p.Contributions, err)
	}
}

func TestGoalieLayerNullEffect(t *testing.T) {
	sim := periods.DefaultSim()
	games, _ := periods.Simulate(sim)
	// Attach goalie lines with no real effect: β must stay near 0.
	for i := range games {
		g := &games[i]
		name := "A"
		if i%3 == 0 {
			name = "B"
		}
		g.Goalies = []canon.GoalieStart{
			{Side: canon.Home, Player: name, Starter: true, ShotsAgainst: 30, Saves: 30 - g.Reg.Away},
			{Side: canon.Away, Player: name, Starter: true, ShotsAgainst: 30, Saves: 30 - g.Reg.Home},
		}
	}
	m := walk(t, games, Config{Base: periods.DefaultConfig(), Layers: All}, BuildInputs(games))
	// No layer has a real effect in this league: every β stays small.
	for l, b := range m.Beta {
		if math.Abs(b) > 1.5 {
			t.Fatalf("null effect %s beta %.3f", l, b)
		}
	}
}

// In a drifting league the Kalman level knows recent form the 180-day baseline
// lags behind: the wave-level layer gets a positive β and better likelihood.
func TestWaveLevelFindsDrift(t *testing.T) {
	sim := periods.DefaultSim()
	sim.DriftSD = 0.01
	games, _ := periods.Simulate(sim)
	m := walk(t, games, Config{Base: periods.DefaultConfig(), Layers: []string{WaveLevel}}, BuildInputs(games))
	if m.Beta[WaveLevel] < 0.05 || m.N[WaveLevel] < 500 {
		t.Fatalf("wave-level beta %.3f n %d", m.Beta[WaveLevel], m.N[WaveLevel])
	}
}

// walk predicts week by week (filling the out-of-sample cache like the
// backtest does) and returns the model fitted on everything at the end.
func walk(t *testing.T, games []canon.Match, cfg Config, in Inputs) *Model {
	t.Helper()
	canon.SortMatches(games)
	start := games[0].StartAt.AddDate(0, 3, 0)
	end := games[len(games)-1].StartAt.Add(time.Hour)
	for d := start; d.Before(end); d = d.Add(7 * 24 * time.Hour) {
		var train []canon.Match
		for _, g := range games {
			if g.StartAt.Before(d) {
				train = append(train, g)
			}
		}
		m, err := Fit(train, d, cfg, in)
		if err != nil {
			t.Fatal(err)
		}
		for _, g := range games {
			if !g.StartAt.Before(d) && g.StartAt.Before(d.Add(7*24*time.Hour)) {
				if _, err := m.Predict(g); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	m, err := Fit(games, end, cfg, in)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
