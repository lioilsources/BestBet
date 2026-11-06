package rating

import (
	"math"
	"testing"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/periods"
)

// The filter must be causal and track the true strength gap of a drifting league.
func TestWaveTimelineTracksDrift(t *testing.T) {
	sim := periods.DefaultSim()
	sim.DriftSD = 0.02
	games, truth := periods.Simulate(sim)
	tl := WaveTimeline(games, DefaultWaveConfig())
	first := tl[games[0].ID]
	if first[canon.Home].Level != 0 || first[canon.Home].Games != 0 {
		t.Fatalf("first game must see a fresh filter: %+v", first)
	}
	// Truth at the end: strength in goal units ≈ 2.8·(att+def) per team.
	last := map[int32]float64{}
	for _, g := range games {
		st := tl[g.ID]
		last[g.HomeTeam] = st[canon.Home].Level
		last[g.AwayTeam] = st[canon.Away].Level
	}
	var xs, ys []float64
	for id, lv := range last {
		xs = append(xs, lv)
		ys = append(ys, truth.Att[id]+truth.Def[id])
	}
	if c := corr(xs, ys); c < 0.7 {
		t.Fatalf("Kalman level vs truth correlation %.2f", c)
	}
}

func corr(x, y []float64) float64 {
	n := float64(len(x))
	var mx, my float64
	for i := range x {
		mx += x[i] / n
		my += y[i] / n
	}
	var sxy, sxx, syy float64
	for i := range x {
		sxy += (x[i] - mx) * (y[i] - my)
		sxx += (x[i] - mx) * (x[i] - mx)
		syy += (y[i] - my) * (y[i] - my)
	}
	return sxy / math.Sqrt(sxx*syy)
}
