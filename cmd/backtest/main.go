// Command backtest runs the walk-forward evaluation of a model over one league
// and writes a results report (plan 01 §3, §5 fáze 1).
//
//	backtest -league swe-allsvenskan -from 2024 -to 2026 -holdout 2025 -out docs/RESULTS_PHASE1.md
//
// Seasons before -from are training warm-up only. Seasons >= -holdout are marked
// as held out; hyperparameters are fixed in docs/DECISIONS.md, never tuned on them.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lioilsources/BestBet/internal/backtest"
	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/periods"
	"github.com/lioilsources/BestBet/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("backtest failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	league := flag.String("league", "swe-allsvenskan", "league slug")
	from := flag.Int("from", 2024, "first evaluated season (start year)")
	to := flag.Int("to", 2026, "last evaluated season (start year)")
	holdout := flag.Int("holdout", 2025, "first held-out season (0 = none)")
	regularOnly := flag.Bool("regular-only", true, "evaluate regular season only (play-off has its own layer, 00 §1.4.7)")
	out := flag.String("out", "", "write the report here (default stdout)")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer st.Close()
	matches, odds, err := st.LoadMatches(ctx, *league)
	if err != nil {
		return err
	}
	var eval []canon.Match
	var start, end time.Time
	for _, m := range matches {
		if m.Season < *from || m.Season > *to || !m.Played() {
			continue
		}
		if start.IsZero() || m.StartAt.Before(start) {
			start = m.StartAt
		}
		if m.StartAt.After(end) {
			end = m.StartAt
		}
		eval = append(eval, m)
	}
	if len(eval) == 0 {
		return fmt.Errorf("no played games in seasons %d-%d", *from, *to)
	}
	cfg := periods.DefaultConfig()
	fit := func(train []canon.Match, asOf time.Time) (backtest.Predictor, error) {
		return periods.Fit(train, asOf, cfg)
	}
	rows, err := backtest.Run(matches, odds, fit, backtest.Config{From: start.Truncate(24 * time.Hour), To: end.Add(24 * time.Hour), MinTrain: 150})
	if err != nil {
		return err
	}
	if *regularOnly {
		kept := rows[:0]
		for _, r := range rows {
			if r.Match.Phase == "regular" {
				kept = append(kept, r)
			}
		}
		rows = kept
	}
	md, err := backtest.RenderResults(backtest.ReportInput{
		Title:        "Fáze 1 — baseline (bez vlny, bez brankáře)",
		League:       *league,
		ModelVersion: periods.Version,
		Config:       fmt.Sprintf("poločas vážení %.0f dní, ridge %.0f, navýšení remíz %v, max %d gólů", cfg.HalfLifeDays, cfg.Ridge, cfg.TieInflation, cfg.MaxGoals),
		Generated:    time.Now(),
		Holdout:      *holdout,
		Rows:         rows,
	})
	if err != nil {
		return err
	}
	if *out == "" {
		fmt.Print(md)
		return nil
	}
	if err := os.WriteFile(*out, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	slog.Info("report written", "file", *out, "rows", len(rows))
	return nil
}
