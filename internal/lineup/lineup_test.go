package lineup

import (
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/features"
)

func TestBuild(t *testing.T) {
	var prev []features.TeamGame
	for d := 1; d <= 8; d++ {
		g := features.TeamGame{StartAt: time.Date(2025, 10, d, 0, 0, 0, 0, time.UTC), Season: 2025}
		for id, toi := range map[int32]int{1: 1200, 2: 1100, 3: 1000, 4: 500, 5: 400} {
			if id == 1 && d == 8 {
				continue // the top player missed the last game
			}
			g.Players = append(g.Players, canon.PlayerMinutes{PlayerID: id, Position: "C", TOISeconds: toi})
		}
		prev = append(prev, g)
	}
	info := map[int32]Info{1: {Name: "A. Star", Games: 7, Points: 14}}
	est := Build(prev, info, 2025, nil)
	if est.Status != Estimate || len(est.Missing) != 1 || est.Missing[0].Name != "A. Star" || !est.Missing[0].Key || est.KeyMissing != 1 {
		t.Fatalf("estimate %+v", est)
	}
	if est.Missing[0].TOIPerGame != 20 || est.Missing[0].PointsPerGame != 2 {
		t.Fatalf("player %+v", est.Missing[0])
	}
	at := time.Date(2025, 10, 9, 0, 0, 0, 0, time.UTC)
	conf := Build(prev, info, 2025, map[string]Published{Confirmed: {Players: []int32{1, 2, 3, 4, 5}, At: at}})
	if conf.Status != Confirmed || len(conf.Missing) != 0 || conf.ConfirmedAt == nil || !conf.ConfirmedAt.Equal(at) {
		t.Fatalf("confirmed %+v", conf)
	}
	// The available list without the star: a certain absence; the confirmed
	// lineup wins when both exist.
	av := Build(prev, info, 2025, map[string]Published{Available: {Players: []int32{2, 3, 4, 5, 6}, At: at}})
	if av.Status != Available || len(av.Missing) != 1 || av.Listed != 5 {
		t.Fatalf("available %+v", av)
	}
	both := Build(prev, info, 2025, map[string]Published{Available: {Players: []int32{2, 3}, At: at}, Confirmed: {Players: []int32{1, 2, 3, 4, 5}, At: at}})
	if both.Status != Confirmed {
		t.Fatalf("both %+v", both)
	}
	if u := Build(prev[:2], info, 2025, nil); u.Status != Unknown {
		t.Fatalf("unknown %+v", u)
	}
}
