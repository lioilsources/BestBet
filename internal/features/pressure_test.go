package features

import (
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

func game(id int64, day int, h, a int32, hs, as int) canon.Match {
	m := canon.Match{ID: id, Season: 2025, Phase: "regular", HomeTeam: h, AwayTeam: a,
		StartAt: time.Date(2025, 10, 1, 18, 0, 0, 0, time.UTC).AddDate(0, 0, day)}
	if hs >= 0 {
		m.Reg = &canon.Score{Home: hs, Away: as}
		m.Final = &canon.Score{Home: hs, Away: as}
		m.Periods = []canon.Score{{}, {}, {Home: hs, Away: as}}
	}
	return m
}

func TestPressure(t *testing.T) {
	// Four teams, line after rank 2, ten games each scheduled; team 1 wins all.
	var ms []canon.Match
	id := int64(1)
	pairs := [][2]int32{{1, 2}, {3, 4}, {1, 3}, {2, 4}, {1, 4}, {2, 3}}
	for r := 0; r < 20; r++ {
		p := pairs[r%len(pairs)]
		hs, as := 3, 1
		if p[1] == 1 {
			hs, as = 1, 3
		}
		if r >= 16 {
			hs = -1 // scheduled, not played
		}
		ms = append(ms, game(id, r, p[0], p[1], hs, as))
		id++
	}
	c := PressureConfig{Lines: []int{2}, MinShare: 0.6, PointsWin: 2, PointsOTW: 2, PointsOTL: 1, PerGameGap: 1}
	tl := PressureTimeline(ms, c)
	if tl[1][canon.Home] || tl[1][canon.Away] {
		t.Fatalf("first game must not be in a race (too early): %+v", tl[1])
	}
	// Late in the season someone is in the race.
	any := false
	for _, m := range ms[12:] {
		if tl[m.ID][canon.Home] || tl[m.ID][canon.Away] {
			any = true
		}
	}
	if !any {
		t.Fatal("expected a race late in the season")
	}
}

func TestCoachTimeline(t *testing.T) {
	var ms []canon.Match
	for i := 0; i < 30; i++ {
		ms = append(ms, game(int64(i+1), i, 1, 2, 1, 0))
	}
	d := func(i int) time.Time { return ms[i].StartAt }
	tl := CoachTimeline(ms, []CoachTenure{
		{Team: 1, Coach: "A", From: d(0), To: d(14)},
		{Team: 1, Coach: "B", From: d(15), To: d(29)},
		{Team: 2, Coach: "C", From: d(0), To: d(29)},
	})
	if tl[1][canon.Home] {
		t.Fatal("season-opening tenure must not count as a change")
	}
	if !tl[16][canon.Home] || !tl[25][canon.Home] || tl[26][canon.Home] {
		t.Fatalf("new coach window wrong: 16=%v 25=%v 26=%v", tl[16][canon.Home], tl[25][canon.Home], tl[26][canon.Home])
	}
	if tl[16][canon.Away] {
		t.Fatal("away coach never changed")
	}
}
