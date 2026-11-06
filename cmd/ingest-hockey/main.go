// Command ingest-hockey fills the M0 tables (plan §1.4.5) and writes DATA_M0.md.
//
//	ingest-hockey migrate                      # creates the database if missing
//	ingest-hockey drop-mock                    # drops DATABASE_URL, only *_mock
//	ingest-hockey leagues                      # sync config, resolve API ids
//	ingest-hockey games    [-league s] [-season y]
//	ingest-hockey swehockey [-league s] [-season y] [-goalies=true]
//	ingest-hockey nhl      [-season y] [-goalies=true]  # public NHL API, no key
//	ingest-hockey odds     [-league s]         # one API-HOCKEY odds snapshot now
//	ingest-hockey snapshot                     # cron: odds for games entering T-24h/T-1h/closing
//	ingest-hockey oddspapi -league s -from 2026-01-01 -to 2026-06-30 [-books pinnacle,bet365,unibet]
//	ingest-hockey report   [-out DATA_M0.md]
//
// Environment: DATABASE_URL, APIHOCKEY_KEY, ODDSPAPI_KEY, CACHE_DIR (data/cache),
// APIHOCKEY_BUDGET (90 requests per run; the free tier has 100 per day).
// MOCK_PROVIDERS=1 replaces both paid APIs with internal/provmock (no keys); it
// only runs against a database whose name ends in _mock (make mock-run).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/config"
	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/ingest"
	"github.com/lioilsources/BestBet/internal/oddspapi"
	"github.com/lioilsources/BestBet/internal/report"
	"github.com/lioilsources/BestBet/internal/store"
)

const userAgent = "BestBet-M0/0.1 (research; low rate)"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ingest-hockey <migrate|drop-mock|leagues|games|swehockey|nhl|odds|snapshot|oddspapi|report> [flags]")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1], os.Args[2:]); err != nil {
		slog.Error("failed", "cmd", os.Args[1], "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd string, args []string) error {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	cfgPath := fs.String("config", "config/leagues.yaml", "league config")
	leagueFlag := fs.String("league", "", "league slug (default: all)")
	seasonFlag := fs.Int("season", 0, "season start year (default: config seasons)")
	goalies := fs.Bool("goalies", true, "swehockey, nhl: also fetch per-game goalies")
	from := fs.String("from", "", "oddspapi: from date YYYY-MM-DD")
	to := fs.String("to", "", "oddspapi: to date YYYY-MM-DD")
	books := fs.String("books", "pinnacle,bet365,unibet", "oddspapi: up to 3 bookmaker slugs")
	out := fs.String("out", "DATA_M0.md", "report: output file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set (see .env.example)")
	}
	if cmd == "drop-mock" {
		return store.DropMockDatabase(ctx, dsn)
	}
	if cmd == "migrate" {
		created, err := store.EnsureDatabase(ctx, dsn)
		if err != nil {
			return err
		}
		if created {
			slog.Info("database created")
		}
		if err := store.Migrate(dsn); err != nil {
			return err
		}
		slog.Info("migrated")
		return nil
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return err
	}
	if os.Getenv("MOCK_PROVIDERS") == "1" {
		if err := startMock(dsn, cfg); err != nil {
			return err
		}
	}
	st, err := store.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer st.Close()
	rows, err := st.SyncLeagues(ctx, cfg)
	if err != nil {
		return err
	}
	env := ingest.Env{Store: st, Cfg: cfg, Now: time.Now}
	leagues, err := selectLeagues(cfg, *leagueFlag)
	if err != nil {
		return err
	}
	seasons := cfg.Seasons
	if *seasonFlag != 0 {
		seasons = []int{*seasonFlag}
	}
	cacheDir := envOr("CACHE_DIR", "data/cache")

	switch cmd {
	case "leagues":
		return cmdLeagues(ctx, st, leagues, cacheDir)
	case "games":
		c, err := apiHockey(cacheDir)
		if err != nil {
			return err
		}
		for _, l := range leagues {
			for _, s := range seasons {
				if _, err := ingest.APIHockeyGames(ctx, env, c, l, refresh(ctx, st, rows, l.Slug), s); err != nil {
					if errors.Is(err, httpcache.ErrBudget) {
						return fmt.Errorf("stopped at %s %d: %w (cached responses are kept, rerun tomorrow)", l.Slug, s, err)
					}
					slog.Error("games", "league", l.Slug, "season", s, "err", err)
				}
			}
		}
		return nil
	case "swehockey":
		hc := &httpcache.Client{Dir: cacheDir + "/swehockey", MinGap: 1500 * time.Millisecond, UserAgent: userAgent}
		for _, l := range leagues {
			for _, sch := range l.Swehockey {
				if *seasonFlag != 0 && sch.Season != *seasonFlag {
					continue
				}
				if _, err := ingest.Swehockey(ctx, env, hc, l, refresh(ctx, st, rows, l.Slug), sch, *goalies); err != nil {
					return fmt.Errorf("swehockey %s %d: %w", l.Slug, sch.ScheduleID, err)
				}
			}
		}
		return nil
	case "odds":
		c, err := apiHockey(cacheDir)
		if err != nil {
			return err
		}
		season := currentSeason(time.Now())
		for _, l := range leagues {
			if _, err := ingest.APIHockeyOdds(ctx, env, c, l, refresh(ctx, st, rows, l.Slug), season); err != nil {
				slog.Error("odds", "league", l.Slug, "err", err)
			}
		}
		return nil
	case "nhl":
		hc := &httpcache.Client{Dir: cacheDir + "/nhl", MinGap: time.Second, UserAgent: userAgent}
		l, ok := cfg.League("nhl")
		if !ok {
			return errors.New("league nhl is not configured")
		}
		for _, s := range seasons {
			if _, err := ingest.NHL(ctx, env, hc, l, refresh(ctx, st, rows, l.Slug), s, *goalies); err != nil {
				return fmt.Errorf("nhl %d: %w", s, err)
			}
		}
		return nil
	case "snapshot":
		return cmdSnapshot(ctx, env, rows, leagues, cacheDir)
	case "oddspapi":
		c, err := oddsPapi(cacheDir)
		if err != nil {
			return err
		}
		f, err := time.Parse("2006-01-02", *from)
		if err != nil {
			return fmt.Errorf("-from: %w", err)
		}
		t, err := time.Parse("2006-01-02", *to)
		if err != nil {
			return fmt.Errorf("-to: %w", err)
		}
		for _, l := range leagues {
			if _, err := ingest.OddsPapi(ctx, env, c, l, refresh(ctx, st, rows, l.Slug), f, t, strings.Split(*books, ",")); err != nil {
				slog.Error("oddspapi", "league", l.Slug, "err", err)
			}
		}
		return nil
	case "report":
		d, err := report.Load(ctx, st.Q, time.Now())
		if err != nil {
			return err
		}
		if err := os.WriteFile(*out, []byte(report.Render(d)), 0o644); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		slog.Info("report written", "file", *out)
		return nil
	}
	return fmt.Errorf("unknown command %q", cmd)
}

func cmdLeagues(ctx context.Context, st *store.Store, leagues []config.League, cacheDir string) error {
	if c, err := apiHockey(cacheDir); err == nil {
		for _, l := range leagues {
			best, all, err := ingest.ResolveAPIHockeyLeague(ctx, c, l)
			if err != nil {
				return err
			}
			if best == nil {
				fmt.Printf("%-16s api-hockey: no unique match; candidates:\n", l.Slug)
				for _, a := range all {
					fmt.Printf("    id=%d %q (%s) seasons=%d\n", a.ID, a.Name, a.Country.Name, len(a.Seasons))
				}
				continue
			}
			id := int32(best.ID)
			if _, err := st.Q.UpsertLeague(ctx, dbq.UpsertLeagueParams{Slug: l.Slug, Name: l.Name, Country: l.Country, Role: l.Role, ApihockeyID: &id}); err != nil {
				return err
			}
			var ss []string
			for _, s := range best.Seasons {
				ss = append(ss, strconv.Itoa(s.Season))
			}
			fmt.Printf("%-16s api-hockey id=%d %q seasons=%s  -> put `id: %d` into config\n", l.Slug, best.ID, best.Name, strings.Join(ss, ","), best.ID)
		}
	} else {
		slog.Warn("skip api-hockey", "err", err)
	}
	if c, err := oddsPapi(cacheDir); err == nil {
		ts, err := c.Tournaments(ctx)
		if err != nil {
			return err
		}
		for _, l := range leagues {
			fmt.Printf("%-16s oddspapi candidates:", l.Slug)
			var hits []oddspapi.Tournament
			for _, t := range ts {
				if strings.Contains(strings.ToLower(t.TournamentName), strings.ToLower(l.OddsPapi.Search)) &&
					strings.Contains(strings.ToLower(t.CategoryName), strings.ToLower(l.OddsPapi.Category)) {
					fmt.Printf(" [%d %q / %s]", t.TournamentID, t.TournamentName, t.CategoryName)
					hits = append(hits, t)
				}
			}
			if len(hits) != 1 {
				fmt.Println("  -> put `tournament_id` into config")
				continue
			}
			// A unique candidate is stored like the API-HOCKEY id (COALESCE keeps the rest).
			id := int32(hits[0].TournamentID)
			if _, err := st.Q.UpsertLeague(ctx, dbq.UpsertLeagueParams{Slug: l.Slug, Name: l.Name, Country: l.Country, Role: l.Role, OddspapiTournamentID: &id}); err != nil {
				return err
			}
			fmt.Printf("  -> stored; put `tournament_id: %d` into config\n", id)
		}
	} else {
		slog.Warn("skip oddspapi", "err", err)
	}
	return nil
}

// cmdSnapshot is the cron entry: for each league with a game entering a capture
// window that has no snapshot of that kind yet, take one API-HOCKEY odds snapshot.
// One request per league per window keeps the free tier alive.
func cmdSnapshot(ctx context.Context, env ingest.Env, rows map[string]dbq.League, leagues []config.League, cacheDir string) error {
	now := env.Now()
	games, err := env.Store.Q.ListGamesStartingBetween(ctx, dbq.ListGamesStartingBetweenParams{FromTs: pgTime(now), ToTs: pgTime(now.Add(28 * time.Hour))})
	if err != nil {
		return fmt.Errorf("upcoming games: %w", err)
	}
	due := map[string]bool{}
	for _, g := range games {
		kind := snapshotKind(now, g.StartAt.Time)
		if kind == "" {
			continue
		}
		has, err := env.Store.Q.HasSnapshot(ctx, dbq.HasSnapshotParams{GameID: &g.ID, SnapshotKind: kind})
		if err != nil {
			return fmt.Errorf("has snapshot: %w", err)
		}
		if !has {
			due[g.LeagueSlug] = true
		}
	}
	if len(due) == 0 {
		slog.Info("snapshot: nothing due")
		return nil
	}
	c, err := apiHockey(cacheDir)
	if err != nil {
		return err
	}
	for _, l := range leagues {
		if !due[l.Slug] {
			continue
		}
		if _, err := ingest.APIHockeyOdds(ctx, env, c, l, rows[l.Slug], currentSeason(now)); err != nil {
			slog.Error("snapshot", "league", l.Slug, "err", err)
		}
	}
	return nil
}

func snapshotKind(now, start time.Time) string {
	switch k := oddsKind(now, start); k {
	case "t24h", "t1h", "closing":
		return k
	}
	return ""
}
