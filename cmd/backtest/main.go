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
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/backtest"
	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/layers"
	"github.com/lioilsources/BestBet/internal/model/periods"
	"github.com/lioilsources/BestBet/internal/rating"
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
	ablation := flag.String("ablation", "", "layer to compare against the baseline: goalie")
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
	walk := backtest.Config{From: start.Truncate(24 * time.Hour), To: end.Add(24 * time.Hour), MinTrain: 150}
	keep := func(rows []backtest.Row) []backtest.Row {
		if !*regularOnly {
			return rows
		}
		var kept []backtest.Row
		for _, r := range rows {
			if r.Match.Phase == "regular" {
				kept = append(kept, r)
			}
		}
		return kept
	}
	rows, err := backtest.Run(matches, odds, fit, walk)
	if err != nil {
		return err
	}
	rows = keep(rows)
	cfgLine := fmt.Sprintf("poločas vážení %.0f dní, ridge %.0f, navýšení remíz %v, max %d gólů", cfg.HalfLifeDays, cfg.Ridge, cfg.TieInflation, cfg.MaxGoals)
	in := backtest.ReportInput{
		Title: "Fáze 1 — baseline (bez vlny, bez brankáře)", League: *league, ModelVersion: periods.Version,
		Config: cfgLine, Generated: time.Now(), Holdout: *holdout, Rows: rows,
	}
	md, err := backtest.RenderResults(in)
	if err != nil {
		return err
	}
	switch *ablation {
	case "":
	case "goalie":
		md, err = goalieAblation(matches, odds, cfg, walk, keep, in, rows)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown -ablation %q", *ablation)
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

func goalieAblation(matches []canon.Match, odds map[int64][]canon.OddsQuote, cfg periods.Config, walk backtest.Config,
	keep func([]backtest.Row) []backtest.Row, baseIn backtest.ReportInput, baseRows []backtest.Row) (string, error) {
	gcfg := rating.DefaultGoalieConfig()
	tl := rating.GoalieTimeline(matches, gcfg)
	var betas []float64
	fit := func(train []canon.Match, asOf time.Time) (backtest.Predictor, error) {
		m, err := layers.Fit(train, asOf, layers.Config{Base: cfg, Goalie: true}, tl)
		if err == nil {
			betas = append(betas, m.GoalieBeta)
		}
		return m, err
	}
	rows, err := backtest.Run(matches, odds, fit, walk)
	if err != nil {
		return "", err
	}
	rows = keep(rows)
	known := 0
	var adjs []float64
	for _, r := range rows {
		if st, ok := tl[r.Match.ID]; ok && st[canon.Home].Known && st[canon.Away].Known {
			known++
			adjs = append(adjs, st[canon.Home].Adj(), st[canon.Away].Adj())
		}
	}
	adjSD, adjBig := 0.0, 0
	for _, a := range adjs {
		adjSD += a * a
		if math.Abs(a) > 0.05 {
			adjBig++
		}
	}
	if len(adjs) > 0 {
		adjSD = math.Sqrt(adjSD / float64(len(adjs)))
	}
	sort.Float64s(betas)
	in := baseIn
	in.Title = "Fáze 2 — vrstva brankář"
	in.ModelVersion = layers.Version
	in.Rows = rows
	variant, err := backtest.RenderResults(in)
	if err != nil {
		return "", err
	}
	deltas := backtest.PairedDelta(baseRows, rows)
	all := deltas[0]
	verdict := "vrstva log-loss **nezlepšuje** (interval obsahuje nulu nebo je kladný)"
	if all.CIHi < 0 {
		verdict = "vrstva log-loss **zlepšuje** (celý 95% interval je pod nulou)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n## Ablace: brankář vs. baseline\n\n", variant)
	fmt.Fprintf(&b, "**Závěr ablace:** %s; Δ = %s na %d zápasech. Srovnání s trhem chybí, dokud nejsou kurzy.\n\n", verdict, cz("%+.4f", all.MeanDelta1X2), all.N)
	b.WriteString(backtest.RenderAblation("brankář", deltas, baseIn.Holdout))
	fmt.Fprintf(&b, "\nFitované β (%d oken walk-forward): min %s, medián %s, max %s. β = 1 znamená, že rozdíl ve stažené úspěšnosti zákroků se promítá do gólů jedna ku jedné; β kolem 0 znamená, že vrstva nic nenese. Shrinkage: prior %.0f střel, poločas %.0f dní.\n",
		len(betas), cz("%.2f", betas[0]), cz("%.2f", betas[len(betas)/2]), cz("%.2f", betas[len(betas)-1]), gcfg.PriorShots, gcfg.HalfLifeDays)
	fmt.Fprintf(&b, "\nSignál na vstupu: směrodatná odchylka adj = %s (log násobek gólů soupeře), |adj| > 0,05 u %d z %d startů. Malé adj znamená, že startér se od týmového průměru skoro neliší (jednička chytá většinu zápasů).\n",
		cz("%.4f", adjSD), adjBig, len(adjs))
	fmt.Fprintf(&b, "\nStartující brankáři známí u obou týmů: %d z %d hodnocených zápasů. Ve zbytku vrstva nic nemění.\n", known, len(rows))
	return b.String(), nil
}

func cz(format string, v float64) string { return strings.Replace(fmt.Sprintf(format, v), ".", ",", 1) }
