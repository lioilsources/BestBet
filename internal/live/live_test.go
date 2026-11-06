package live

import (
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/model/periods"
)

func TestPredictOnlyUpcomingAndCausal(t *testing.T) {
	games, _ := periods.Simulate(periods.DefaultSim())
	now := games[len(games)/2].StartAt
	// Pretend everything from now on is unplayed.
	for i := range games {
		if !games[i].StartAt.Before(now) {
			games[i].Reg, games[i].Final, games[i].Periods = nil, nil, nil
		}
	}
	ps, err := Predict(games, now, 48*time.Hour, periods.DefaultConfig())
	if err != nil || len(ps) == 0 {
		t.Fatalf("%d %v", len(ps), err)
	}
	starts := map[int64]time.Time{}
	for _, g := range games {
		starts[g.ID] = g.StartAt
	}
	for _, p := range ps {
		s := starts[p.MatchID]
		if s.Before(now) || !s.Before(now.Add(48*time.Hour)) || !p.ComputedAt.Equal(now) {
			t.Fatalf("prediction %d at %v for start %v", p.MatchID, p.ComputedAt, s)
		}
	}
	// Too little history: nothing.
	if ps, _ := Predict(games[:50], games[49].StartAt, 48*time.Hour, periods.DefaultConfig()); ps != nil {
		t.Fatal("expected no predictions with 50 games")
	}
}
