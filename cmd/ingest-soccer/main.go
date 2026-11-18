// Command ingest-soccer fills the soccer leagues of config/leagues.yaml (plan 05).
// The schema is shared with hockey; run `ingest-hockey migrate` first.
//
//	ingest-soccer asa    [-league s] [-season y]   # American Soccer Analysis: games, halves (no key)
//	ingest-soccer fdcouk [-league s]               # football-data.co.uk closing 1X2 (no key)
//
// Environment: DATABASE_URL, CACHE_DIR (data/cache).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/lioilsources/BestBet/internal/asa"
	"github.com/lioilsources/BestBet/internal/config"
	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/ingest"
	"github.com/lioilsources/BestBet/internal/store"
)

const userAgent = "BestBet-M0/0.1 (research; low rate)"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ingest-soccer <asa|fdcouk> [flags]")
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
	leagueFlag := fs.String("league", "", "league slug (default: all soccer leagues)")
	seasonFlag := fs.Int("season", 0, "season (default: the league's seasons)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set (see .env.example)")
	}
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return err
	}
	leagues := cfg.Of(config.SportSoccer)
	if *leagueFlag != "" {
		l, ok := cfg.League(*leagueFlag)
		if !ok || l.Sport != config.SportSoccer {
			return fmt.Errorf("no soccer league %q", *leagueFlag)
		}
		leagues = []config.League{l}
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
	cacheDir := os.Getenv("CACHE_DIR")
	if cacheDir == "" {
		cacheDir = "data/cache"
	}

	switch cmd {
	case "asa":
		hc := &httpcache.Client{Dir: cacheDir + "/asa", MinGap: time.Second, UserAgent: userAgent}
		for _, l := range leagues {
			if l.ASA == "" {
				continue
			}
			c := &asa.Client{HTTP: hc, League: l.ASA}
			seasons := l.Seasons
			if *seasonFlag != 0 {
				seasons = []int{*seasonFlag}
			}
			for _, s := range seasons {
				if _, err := ingest.ASA(ctx, env, c, l, rows[l.Slug], s); err != nil {
					return fmt.Errorf("%s %d: %w", l.Slug, s, err)
				}
			}
		}
		return nil
	case "fdcouk":
		// The site serves an empty body to some agents; the default Go agent works.
		hc := &httpcache.Client{Dir: cacheDir + "/fdcouk", MinGap: time.Second}
		for _, l := range leagues {
			if l.FDCOUK.URL == "" {
				continue
			}
			b, _, err := hc.Get(ctx, httpcache.Request{URL: l.FDCOUK.URL, Key: l.FDCOUK.URL, TTL: 12 * time.Hour})
			if err != nil {
				return err
			}
			if _, err := ingest.FDCOUK(ctx, env, b, l, rows[l.Slug], l.FDCOUK.League); err != nil {
				return fmt.Errorf("%s: %w", l.Slug, err)
			}
		}
		return nil
	}
	return fmt.Errorf("unknown command %q", cmd)
}
