package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lioilsources/BetWave/internal/apihockey"
	"github.com/lioilsources/BetWave/internal/config"
	"github.com/lioilsources/BetWave/internal/db/dbq"
	"github.com/lioilsources/BetWave/internal/httpcache"
	"github.com/lioilsources/BetWave/internal/odds"
	"github.com/lioilsources/BetWave/internal/store"
)

func selectLeagues(cfg config.File, slug string) ([]config.League, error) {
	if slug == "" {
		return cfg.Leagues, nil
	}
	l, ok := cfg.League(slug)
	if !ok {
		return nil, fmt.Errorf("unknown league %q", slug)
	}
	return []config.League{l}, nil
}

func apiHockey(cacheDir string) (*apihockey.Client, error) {
	key := os.Getenv("APIHOCKEY_KEY")
	if key == "" {
		return nil, errors.New("APIHOCKEY_KEY is not set")
	}
	budget := 90
	if v := os.Getenv("APIHOCKEY_BUDGET"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("APIHOCKEY_BUDGET: %w", err)
		}
		budget = n
	}
	hc := &httpcache.Client{Dir: cacheDir + "/apihockey", MinGap: 7 * time.Second, Budget: budget, UserAgent: userAgent}
	return apihockey.New(key, hc), nil
}

// refresh re-reads a league row: `leagues` may have just stored its API ids.
func refresh(ctx context.Context, st *store.Store, rows map[string]dbq.League, slug string) dbq.League {
	if r, err := st.Q.GetLeagueBySlug(ctx, slug); err == nil {
		return r
	}
	return rows[slug]
}

// currentSeason: a hockey season starting in August belongs to that year.
func currentSeason(now time.Time) int {
	if now.Month() >= time.July {
		return now.Year()
	}
	return now.Year() - 1
}

func oddsKind(now, start time.Time) string { return odds.SnapshotKind(now, start) }

func pgTime(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
