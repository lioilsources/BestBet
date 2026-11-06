package provmock

import (
	"context"
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/apihockey"
	"github.com/lioilsources/BestBet/internal/config"
	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/oddspapi"
)

func world(now time.Time) *World {
	var l config.League
	l.Slug, l.Name, l.Country = "swe-allsvenskan", "HockeyAllsvenskan", "Sweden"
	l.APIHockey.Search, l.APIHockey.Country = "HockeyAllsvenskan", "Sweden"
	l.OddsPapi.Search, l.OddsPapi.Category = "Allsvenskan", "Sweden"
	return New(config.File{Seasons: []int{2025}, Leagues: []config.League{l}}, 1, func() time.Time { return now })
}

func TestMockServesWhatTheClientsParse(t *testing.T) {
	now := time.Date(2025, 11, 20, 12, 0, 0, 0, time.UTC)
	w := world(now)
	srv := httptest.NewServer(w.Handler())
	defer srv.Close()
	ctx := context.Background()

	ah := apihockey.New("mock", &httpcache.Client{Dir: t.TempDir()})
	ah.Base = srv.URL + "/apihockey"
	ls, err := ah.Leagues(ctx, "HockeyAllsvenskan")
	if err != nil || len(ls) != 1 || ls[0].ID != 900 {
		t.Fatalf("leagues %+v %v", ls, err)
	}
	games, err := ah.Games(ctx, 900, 2025, -1)
	if err != nil || len(games) != 132 {
		t.Fatalf("games %d %v", len(games), err)
	}
	finished, upcoming := 0, 0
	for _, g := range games {
		switch apihockey.Status(g.Status.Short) {
		case "finished":
			r, err := g.Result()
			if err != nil {
				t.Fatalf("result %d: %v", g.ID, err)
			}
			if r.Final.Home != *g.Scores.Home || r.Final.Away != *g.Scores.Away {
				t.Fatalf("final disagrees: %+v", g)
			}
			finished++
		case "scheduled":
			upcoming++
		}
	}
	if finished == 0 || upcoming == 0 {
		t.Fatalf("finished %d upcoming %d", finished, upcoming)
	}
	items, _, err := ah.Odds(ctx, 900, 2025, 1)
	if err != nil || len(items) == 0 || len(items[0].Bookmakers) != len(Books) {
		t.Fatalf("odds %d %v", len(items), err)
	}

	op := &oddspapi.Client{Base: srv.URL + "/oddspapi", Key: "mock", HTTP: &httpcache.Client{Dir: t.TempDir()}}
	ts, err := op.Tournaments(ctx)
	if err != nil || len(ts) != 1 || ts[0].TournamentID != 9000 {
		t.Fatalf("tournaments %+v %v", ts, err)
	}
	fx, err := op.Fixtures(ctx, 9000, now.AddDate(0, 0, -14), now, -1)
	if err != nil || len(fx) == 0 {
		t.Fatalf("fixtures %d %v", len(fx), err)
	}
	mk, err := op.Markets(ctx)
	if err != nil || len(mk) != 2 {
		t.Fatalf("markets %v", err)
	}
	pts, err := op.HistoricalOdds(ctx, fx[0].FixtureID, []string{"mock-sharp", "pinnacle"}, true)
	if err != nil || len(pts) == 0 {
		t.Fatalf("history %d %v", len(pts), err)
	}
	books := map[string]bool{}
	for _, p := range pts {
		books[p.Book] = true
	}
	if !books["mock-sharp"] || !books["mock-pinnacle"] || books["pinnacle"] {
		t.Fatalf("real book names must be prefixed: %v", books)
	}
}

// Closing prices are closer to the fair price than opening ones (the market
// learns), so CLV and open→close movement have something to measure.
func TestPricesConvergeToFair(t *testing.T) {
	w := world(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	l := &w.Leagues[0]
	var openErr, closeErr float64
	for _, g := range l.Games[:100] {
		fair := l.Fair(g)["1x2"]["home"]
		b := Books[0]
		po := 1 / (w.Price(l, g, b, "1x2", "home", g.Start.Add(-120*time.Hour)) * (1 + b.Margin))
		pc := 1 / (w.Price(l, g, b, "1x2", "home", g.Start.Add(-5*time.Minute)) * (1 + b.Margin))
		openErr += math.Abs(po - fair)
		closeErr += math.Abs(pc - fair)
	}
	if !(closeErr < 0.6*openErr) {
		t.Fatalf("close error %.3f should be well below open %.3f", closeErr, openErr)
	}
}
