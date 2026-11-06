// Command predict is the cron step of plan 01 fáze 4: for every configured
// league, fit the current model on games played so far and store live
// predictions for games starting within the horizon.
//
//	predict [-league s] [-horizon 72h]
//
// Environment: DATABASE_URL. Re-running is idempotent (same inputs hash).
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/lioilsources/BestBet/internal/config"
	"github.com/lioilsources/BestBet/internal/live"
	"github.com/lioilsources/BestBet/internal/model/layers"
	"github.com/lioilsources/BestBet/internal/model/periods"
	"github.com/lioilsources/BestBet/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("predict failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := flag.String("config", "config/leagues.yaml", "league config")
	league := flag.String("league", "", "league slug (default: all)")
	horizon := flag.Duration("horizon", 72*time.Hour, "predict games starting within this window")
	flag.Parse()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return err
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer st.Close()
	now := time.Now().UTC()
	for _, l := range cfg.Leagues {
		if *league != "" && l.Slug != *league {
			continue
		}
		ms, err := st.LeagueMatches(ctx, l.Slug)
		if err != nil {
			return err
		}
		ps, err := live.Predict(ms, now, *horizon, layers.Config{Base: periods.DefaultConfig(), Layers: l.Layers})
		if err != nil {
			slog.Error("predict", "league", l.Slug, "err", err)
			continue
		}
		for _, p := range ps {
			if err := st.SavePrediction(ctx, "live", p); err != nil {
				return err
			}
		}
		slog.Info("predict", "league", l.Slug, "model", layers.Version(l.Layers), "games", len(ms), "predicted", len(ps))
	}
	return nil
}
