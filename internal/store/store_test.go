package store

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lioilsources/BestBet/internal/config"
	"github.com/lioilsources/BestBet/internal/hockey"
)

// testStore creates a throw-away database on the server in TEST_DATABASE_URL
// (e.g. the compose.yml Postgres) and drops it afterwards. Skipped when unset.
func testStore(t *testing.T) *Store {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("bestbet_test_%d", os.Getpid())
	if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(base)
	u.Path = "/" + name
	if err := Migrate(u.String()); err != nil {
		t.Fatal(err)
	}
	s, err := Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		s.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		admin.Close(ctx)
	})
	return s
}

func TestSwehockeyOwnsGameWithoutAPI(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	cfg := config.File{Leagues: []config.League{{Slug: "swe2", Name: "HockeyAllsvenskan", Country: "Sweden", Role: "fringe", TZ: "Europe/Stockholm"}}}
	leagues, err := s.SyncLeagues(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	lg := leagues["swe2"]
	teams, _ := s.LoadTeams(ctx, lg.ID, "swehockey", nil)
	h, _ := teams.Resolve(ctx, "AIK", 0)
	a, _ := teams.Resolve(ctx, "Mora IK", 0)
	start := time.Date(2026, 11, 6, 18, 0, 0, 0, time.UTC)
	in := GameInput{LeagueID: lg.ID, Season: 2026, StartAt: start, Status: "scheduled", HomeTeamID: h, AwayTeamID: a, Phase: "regular"}
	if _, _, err := s.UpsertSwehockeyGame(ctx, 77, in); err != nil {
		t.Fatal(err)
	}
	res, _ := hockey.BuildResult([]hockey.Score{{Home: 1, Away: 0}, {Home: 0, Away: 0}, {Home: 0, Away: 0}}, nil, nil)
	in.Status, in.Result, in.OTType = "finished", &res, "none"
	if _, method, err := s.UpsertSwehockeyGame(ctx, 77, in); err != nil || method != "external_id" {
		t.Fatal(err, method)
	}
	var status string
	var hs int
	if err := s.Pool.QueryRow(ctx, "SELECT status, home_score FROM games WHERE swehockey_id = 77").Scan(&status, &hs); err != nil || status != "finished" || hs != 1 {
		t.Fatalf("status=%s home=%d err=%v", status, hs, err)
	}
}

func TestSecondaryThenPrimaryMerge(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	cfg := config.File{Leagues: []config.League{{Slug: "swe2", Name: "HockeyAllsvenskan", Country: "Sweden", Role: "fringe", TZ: "Europe/Stockholm"}}}
	leagues, err := s.SyncLeagues(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	lg := leagues["swe2"]
	start := time.Date(2025, 9, 19, 17, 0, 0, 0, time.UTC)
	res, _ := hockey.BuildResult([]hockey.Score{{Home: 0, Away: 0}, {Home: 1, Away: 1}, {Home: 2, Away: 0}}, nil, nil)

	// 1) swehockey first: creates teams and the game.
	swe, err := s.LoadTeams(ctx, lg.ID, "swehockey", nil)
	if err != nil {
		t.Fatal(err)
	}
	h, _ := swe.Resolve(ctx, "MoDo Hockey", 0)
	a, _ := swe.Resolve(ctx, "Östersunds IK", 0)
	sweGame, method, err := s.UpsertSwehockeyGame(ctx, 1005612, GameInput{LeagueID: lg.ID, Season: 2025, StartAt: start, Status: "finished", HomeTeamID: h, AwayTeamID: a, Result: &res, Phase: "regular", OTType: "none"})
	if err != nil || method != "new" {
		t.Fatalf("swehockey insert: %v %s", err, method)
	}

	// 2) API-HOCKEY with other spellings and a kick-off one hour off: same teams, same game.
	api, err := s.LoadTeams(ctx, lg.ID, "apihockey", nil)
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := api.Resolve(ctx, "MoDo", 501)
	a2, _ := api.Resolve(ctx, "Ostersund", 502)
	if h2 != h || a2 != a || len(api.Created) != 0 {
		t.Fatalf("teams not merged: %d/%d vs %d/%d created=%v", h2, a2, h, a, api.Created)
	}
	apiGame, method, err := s.UpsertAPIHockeyGame(ctx, 9001, GameInput{LeagueID: lg.ID, Season: 2025, StartAt: start.Add(time.Hour), Status: "finished", HomeTeamID: h2, AwayTeamID: a2, Result: &res, Phase: "regular", OTType: "none"})
	if err != nil || method != "date_teams" || apiGame != sweGame {
		t.Fatalf("api attach: id %d vs %d, %s, %v", apiGame, sweGame, method, err)
	}

	// 3) Re-running both sources is idempotent.
	if _, method, _ = s.UpsertAPIHockeyGame(ctx, 9001, GameInput{LeagueID: lg.ID, Season: 2025, StartAt: start, Status: "finished", HomeTeamID: h, AwayTeamID: a, Result: &res, Phase: "regular"}); method != "external_id" {
		t.Fatalf("rerun api: %s", method)
	}
	if id, method, _ := s.UpsertSwehockeyGame(ctx, 1005612, GameInput{LeagueID: lg.ID, Season: 2025, StartAt: start, Status: "finished", HomeTeamID: h, AwayTeamID: a, Result: &res, Phase: "regular"}); method != "external_id" || id != sweGame {
		t.Fatalf("rerun swe: %s %d", method, id)
	}
	var n int
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM games").Scan(&n); err != nil || n != 1 {
		t.Fatalf("games: %d %v", n, err)
	}
	var h60 int
	if err := s.Pool.QueryRow(ctx, "SELECT home_60 FROM games").Scan(&h60); err != nil || h60 != 3 {
		t.Fatalf("home_60: %d %v", h60, err)
	}
}
