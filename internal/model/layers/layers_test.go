package layers

import (
	"math"
	"testing"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/periods"
	"github.com/lioilsources/BestBet/internal/rating"
)

func TestGoalieLayerFindsEffect(t *testing.T) {
	sim := periods.DefaultSim()
	sim.Seasons = 3
	sim.GoalieGap = 0.12 // starter vs backup: ~0.6 goals per game in total
	games, _ := periods.Simulate(sim)
	tl := rating.GoalieTimeline(games, rating.DefaultGoalieConfig())
	asOf := games[len(games)-1].StartAt
	m, err := Fit(games, asOf, Config{Base: periods.DefaultConfig(), Goalie: true}, tl)
	if err != nil {
		t.Fatal(err)
	}
	// Shrinkage dilutes adj, so the fitted β is > 1; it must be clearly positive.
	if m.GoalieBeta < 0.5 || m.GoalieN < 500 {
		t.Fatalf("beta %.3f n %d", m.GoalieBeta, m.GoalieN)
	}
	// Prediction with a backup in goal for the home side gives the away side more goals.
	var backupGame canon.Match
	for _, g := range games[len(games)/2:] {
		if g.Goalies[0].Player[len(g.Goalies[0].Player)-1] == '2' {
			backupGame = g
			break
		}
	}
	_, la, adjH, _ := m.Lambdas(backupGame)
	_, laBase := m.Base.Lambdas(backupGame.HomeTeam, backupGame.AwayTeam)
	if !(adjH > 0 && la > laBase) {
		t.Fatalf("backup should raise away λ: adj %.3f la %.3f base %.3f", adjH, la, laBase)
	}
	p, err := m.Predict(backupGame)
	if err != nil || len(p.Contributions) != 2 || p.Contributions[0].PP >= 0 {
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
	tl := rating.GoalieTimeline(games, rating.DefaultGoalieConfig())
	m, err := Fit(games, games[len(games)-1].StartAt, Config{Base: periods.DefaultConfig(), Goalie: true}, tl)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(m.GoalieBeta) > 1.5 {
		t.Fatalf("null effect beta %.3f", m.GoalieBeta)
	}
}
