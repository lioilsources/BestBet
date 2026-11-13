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
	"sort"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/backtest"
	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/layers"
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
	ablate := flag.String("ablation", "", "layers to compare against the baseline: all or a comma list of "+strings.Join(layers.All, ","))
	title := flag.String("title", "Fáze 1 — baseline (bez vlny, bez brankáře)", "report title")
	calibrate := flag.Bool("calib", false, "per-market calibration fitted on -tune, scored on held-out seasons")
	tune := flag.Int("tune", 2024, "calib: tuning season (never a held-out one)")
	gate := flag.Bool("gate", false, "simulate 1X2 bets (01 §2.3) and render brána G1 (01 fáze 3)")
	sharp := flag.String("sharp", "pinnacle", "gate: reference book for the close")
	ablNote := flag.String("gate-ablation", "žádná vrstva nezlepšila (`docs/RESULTS_PHASE2.md`)", "gate: ablation result for this league (text)")
	books := flag.String("books", "tipsport,fortuna,chance,betano", "gate: books we may bet at (CZ licensed)")
	absence := flag.Bool("absence", false, "plan 04 L4: H7 and F-abs, absences against the market (replaces the report)")
	m1b := flag.Bool("m1b", false, "plan 02 F1: market-prior model, every layer against the market (replaces the report)")
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
	// The walk starts with the first game of any season so that layers have
	// out-of-sample baseline predictions to fit on; rows before -from are dropped.
	var eval []canon.Match
	var start, end time.Time
	for _, m := range matches {
		if m.Season > *to || !m.Played() {
			continue
		}
		if start.IsZero() || m.StartAt.Before(start) {
			start = m.StartAt
		}
		if m.StartAt.After(end) {
			end = m.StartAt
		}
		if m.Season >= *from {
			eval = append(eval, m)
		}
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
		var kept []backtest.Row
		for _, r := range rows {
			if r.Match.Season >= *from && (!*regularOnly || r.Match.Phase == "regular") {
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
		Title: *title, League: *league, ModelVersion: periods.Version,
		Config: cfgLine, Generated: time.Now(), Holdout: *holdout, Rows: rows,
	}
	md, err := backtest.RenderResults(in)
	if err != nil {
		return err
	}
	if *ablate != "" {
		names := layers.All
		if *ablate != "all" {
			names = strings.Split(*ablate, ",")
		}
		in.Title = "Fáze 2 — ablace vrstev"
		md, err = ablation(matches, odds, cfg, walk, keep, in, rows, names)
		if err != nil {
			return err
		}
	}
	if *calibrate {
		if *holdout != 0 && *tune >= *holdout {
			return fmt.Errorf("-tune %d is a held-out season", *tune)
		}
		md += "\n## Kalibrace po trzích (01 fáze 3)\n\n" + backtest.RenderCalibration(rows, *tune, *holdout)
	}
	if *gate {
		md += gateSection(rows, odds, *holdout, *sharp, strings.Split(*books, ","), *ablNote)
	}
	if *m1b {
		tenures, err := st.LoadCoachTenures(ctx, *league)
		if err != nil {
			return err
		}
		md = m1bReport(m1bInput{League: *league, Matches: matches, Odds: odds, Base: rows, Tenures: tenures,
			Sharp: *sharp, Books: strings.Split(*books, ","), Holdout: *holdout}, time.Now())
	}
	if *absence {
		if err := st.AttachLineups(ctx, *league, matches); err != nil {
			return err
		}
		md = absReport(absInput{League: *league, Matches: matches, Odds: odds, Base: rows, Sharp: *sharp, Holdout: *holdout}, time.Now())
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

// ablation runs the baseline with each layer alone and with all of them, on the
// same walk, and renders the phase 2 table (01 §5 fáze 2). Layers must improve
// log-loss to stay; p-values are Benjamini–Hochberg corrected over the layers.
func ablation(matches []canon.Match, odds map[int64][]canon.OddsQuote, cfg periods.Config, walk backtest.Config,
	keep func([]backtest.Row) []backtest.Row, baseIn backtest.ReportInput, baseRows []backtest.Row, names []string) (string, error) {
	type variant struct {
		label  string
		layers []string
		rows   []backtest.Row
		betas  map[string][]float64
		lastN  map[string]int
		delta  map[int]backtest.Delta
	}
	var vs []*variant
	for _, l := range names {
		vs = append(vs, &variant{label: l, layers: []string{l}})
	}
	if len(names) > 1 {
		vs = append(vs, &variant{label: "vše", layers: names})
	}
	for _, v := range vs {
		in := layers.BuildInputs(matches)
		v.betas, v.lastN = map[string][]float64{}, map[string]int{}
		lc := layers.Config{Base: cfg, Layers: v.layers}
		fit := func(train []canon.Match, asOf time.Time) (backtest.Predictor, error) {
			m, err := layers.Fit(train, asOf, lc, in)
			if err == nil {
				for l, b := range m.Beta {
					v.betas[l] = append(v.betas[l], b)
					v.lastN[l] = m.N[l]
				}
			}
			return m, err
		}
		rows, err := backtest.Run(matches, odds, fit, walk)
		if err != nil {
			return "", fmt.Errorf("ablation %s: %w", v.label, err)
		}
		v.rows = keep(rows)
		v.delta = backtest.PairedDelta(baseRows, v.rows)
	}
	var pvals []float64
	for _, v := range vs[:len(names)] {
		pvals = append(pvals, v.delta[0].P)
	}
	sig := backtest.BenjaminiHochberg(pvals, 0.10)

	var b strings.Builder
	base, err := backtest.RenderResults(baseIn)
	if err != nil {
		return "", err
	}
	b.WriteString(base)
	b.WriteString("\n## Ablace\n\n")
	b.WriteString("Každá varianta běží na stejných oknech jako baseline; Δ je párový rozdíl log-loss 1X2 (varianta − baseline), záporné = vrstva pomáhá. ")
	b.WriteString("Koeficienty vrstev se fitují out-of-sample: tréninkový zápas vstupuje s intenzitami baseline, které pro něj byly předpovězené před zápasem. ")
	b.WriteString("p-hodnoty jsou korigované Benjamini–Hochberg (q = 0,10) přes jednotlivé vrstvy.\n\n")
	b.WriteString("| Vrstva | Zápasů | Δ celkem | 95% interval | p | BH | Δ odložené | medián β | hodnocených s vrstvou | tréninkových s vrstvou (poslední okno) |\n|---|---|---|---|---|---|---|---|---|---|\n")
	for i, v := range vs {
		d := v.delta[0]
		bh := "—"
		if i < len(names) {
			bh = "ne"
			if sig[i] && d.MeanDelta1X2 < 0 {
				bh = "**ano**"
			}
		}
		var beta, ns []string
		for _, l := range v.layers {
			ns = append(ns, fmt.Sprint(v.lastN[l]))
			bs := append([]float64(nil), v.betas[l]...)
			sort.Float64s(bs)
			med := 0.0
			if len(bs) > 0 {
				med = bs[len(bs)/2]
			}
			beta = append(beta, cz("%.2f", med))
		}
		ho := holdoutDelta(baseRows, v.rows, baseIn.Holdout)
		fmt.Fprintf(&b, "| %s | %d | %s | %s až %s | %s | %s | %s | %s | %d | %s |\n", v.label, d.N, cz("%+.4f", d.MeanDelta1X2),
			cz("%+.4f", d.CILo), cz("%+.4f", d.CIHi), cz("%.3f", d.P), bh, ho, strings.Join(beta, " / "), active(v.rows), strings.Join(ns, " / "))
	}
	b.WriteString("\nVrstva s méně než 100 tréninkovými zápasy, kde je aktivní, se nefituje (β = 0) a nic nemění.\n")
	b.WriteString("\nVrstva zůstává jen tehdy, když Δ je záporné celkem i na odložených sezonách a projde BH. Ostatní jdou ven a zůstávají jako informace v UI.\n")
	for _, v := range vs {
		fmt.Fprintf(&b, "\n### %s\n\n", v.label)
		b.WriteString(backtest.RenderAblation(v.label, v.delta, baseIn.Holdout))
	}
	b.WriteString("\n## Kopíruje vlna trh?\n\nKorelace rezidua modelu s pohybem kurzu od otevření k uzavření (01 §5 fáze 2) vyžaduje otevírací i uzavírací kurzy. Pro Allsvenskan 2023–2026 je nemáme, takže test čeká na data.\n")
	return b.String(), nil
}

// gateSection simulates the staking policy on the baseline and renders G1.
func gateSection(rows []backtest.Row, odds map[int64][]canon.OddsQuote, holdout int, sharp string, books []string, ablNote string) string {
	cfg := backtest.DefaultBetConfig()
	cfg.Sharp, cfg.Books = sharp, books
	bets := backtest.SimulateBets(rows, odds, cfg)
	var b strings.Builder
	fmt.Fprintf(&b, "\n## Sázky (01 §2.3)\n\nBaseline, trh 1X2, sázka u %s s posledním kurzem nejméně 1 h před začátkem, když p_model − p_implied ≥ 3 p. b. (Shin) a EV > 0. Kelly 1/4, strop 2 %% bankrollu, nejvýš 5 sázek denně. CLV = log(vzatý kurz × p uzavíracího %s po odmaržování).\n\n",
		strings.Join(books, ", "), sharp)
	b.WriteString("| Sezona | Sázek | Yield | CLV | 95% interval CLV | Poráží close | Max drawdown | Bankroll na konci |\n|---|---|---|---|---|---|---|---|\n")
	by := map[int][]backtest.Bet{}
	for _, x := range bets {
		by[x.Match.Season] = append(by[x.Match.Season], x)
	}
	var ss []int
	for k := range by {
		ss = append(ss, k)
	}
	sort.Ints(ss)
	line := func(label string, xs []backtest.Bet) {
		s := backtest.SummariseBets(xs, 1)
		clv, ci, beat := "chybí", "—", "—"
		if s.NCLV > 0 {
			clv = cz("%+.2f %%", 100*s.MeanCLV)
			ci = cz("%+.2f", 100*s.CLVLo) + " až " + cz("%+.2f %%", 100*s.CLVHi)
			beat = cz("%.0f %%", 100*s.BeatClose)
		}
		fmt.Fprintf(&b, "| %s | %d | %s | %s | %s | %s | %s | %s |\n", label, s.N, cz("%+.1f %%", 100*s.Yield), clv, ci, beat, cz("%.1f %%", 100*s.MaxDD), cz("%.3f", s.EndBank))
	}
	for _, k := range ss {
		label := fmt.Sprintf("%d/%02d", k, (k+1)%100)
		if holdout != 0 && k >= holdout {
			label += " (odložená)"
		}
		line(label, by[k])
	}
	if len(bets) == 0 {
		b.WriteString("| **celkem** | 0 | — | chybí | — | — | — | — |\n\nŽádná sázka: pro tyto zápasy nejsou kurzy knih, u kterých smíme sázet.\n")
	} else {
		line("**celkem**", bets)
	}
	if c, n := backtest.MarketMove(rows, odds, sharp); n > 0 {
		fmt.Fprintf(&b, "\nModel vs. pohyb trhu: korelace (p_model − p_open) s (p_close − p_open) pro výhru domácích = %s na %d zápasech. Kladná korelace znamená, že model předjímá, kam se trh do začátku posune.\n", cz("%.2f", c), n)
	} else {
		fmt.Fprintf(&b, "\nModel vs. pohyb trhu: chybí (nejsou otevírací a uzavírací kurzy %s).\n", sharp)
	}
	b.WriteString("\n## Brána G1 (01 fáze 3)\n\n")
	b.WriteString(backtest.RenderGate(backtest.GateInput{Rows: rows, Bets: bets, Holdout: holdout, Sharp: sharp, Odds: odds,
		Ablation: ablNote, AblationOK: false}))
	return b.String()
}

func holdoutDelta(base, variant []backtest.Row, holdout int) string {
	if holdout == 0 {
		return "—"
	}
	var bh, vh []backtest.Row
	for _, r := range base {
		if r.Match.Season >= holdout {
			bh = append(bh, r)
		}
	}
	for _, r := range variant {
		if r.Match.Season >= holdout {
			vh = append(vh, r)
		}
	}
	d, ok := backtest.PairedDelta(bh, vh)[0]
	if !ok {
		return "—"
	}
	return cz("%+.4f", d.MeanDelta1X2)
}

// active counts rows where at least one layer moved the prediction.
func active(rows []backtest.Row) int {
	n := 0
	for _, r := range rows {
		for _, c := range r.Pred.Contributions {
			if c.PP != 0 {
				n++
				break
			}
		}
	}
	return n
}

func cz(format string, v float64) string { return strings.Replace(fmt.Sprintf(format, v), ".", ",", 1) }
