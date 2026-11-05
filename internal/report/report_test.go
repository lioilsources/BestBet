package report

import (
	"strings"
	"testing"
	"time"

	"github.com/lioilsources/BetWave/internal/db/dbq"
)

func TestRenderRanksFringeLeagues(t *testing.T) {
	cov := func(slug, name, role string, season int32, games, odds, goalies int64) dbq.CoverageByLeagueSeasonRow {
		return dbq.CoverageByLeagueSeasonRow{Slug: slug, Name: name, Role: role, Season: season, Games: games, GamesWithOdds: odds, GamesWithGoalies: goalies}
	}
	d := Data{GeneratedAt: time.Date(2026, 11, 5, 6, 0, 0, 0, time.UTC)}
	for _, s := range []int32{2023, 2024, 2025} {
		d.Coverage = append(d.Coverage,
			cov("swe2", "HockeyAllsvenskan", "fringe", s, 364, 300, 360),
			cov("cze2", "Maxa liga", "fringe", s, 400, 200, 0),
			cov("ger2", "DEL2", "fringe", s, 364, 360, 0),
			cov("nhl", "NHL", "control", s, 1312, 1312, 1300),
		)
	}
	d.Coverage = append(d.Coverage, cov("svk", "Tipos extraliga", "fringe", 2025, 300, 300, 300))
	for _, slug := range []string{"swe2", "cze2", "ger2", "nhl", "svk"} {
		d.Margins = append(d.Margins, dbq.MarginByLeagueRow{Slug: slug, Market: "1x2", AvgMargin: 0.08, Books: 6, Games: 100})
	}
	d.Joins = []dbq.JoinRateBySourceRow{{Slug: "ger2", Source: "oddspapi", Fixtures: 100, Matched: 90}}

	sums := Summarise(d)
	got := map[string]LeagueSummary{}
	for _, s := range sums {
		got[s.Slug] = s
	}
	if got["ger2"].Score != 0 || len(got["ger2"].Gates) != 1 {
		t.Errorf("ger2 must fail the join gate: %+v", got["ger2"])
	}
	if got["svk"].Score != 0 {
		t.Errorf("svk has one season only: %+v", got["svk"])
	}
	if got["nhl"].Score != 0 {
		t.Errorf("control league is never ranked")
	}
	md := Render(d)
	if !strings.Contains(md, "vybírá **HockeyAllsvenskan** a **Maxa liga**") {
		t.Fatalf("recommendation missing:\n%s", md)
	}
	if !strings.Contains(md, "2025/26") {
		t.Errorf("season label missing")
	}
}

func TestRenderEmpty(t *testing.T) {
	md := Render(Data{GeneratedAt: time.Unix(0, 0)})
	if !strings.Contains(md, "zatím žádná data") || !strings.Contains(md, "Žádná okrajová liga") {
		t.Fatal(md)
	}
}
