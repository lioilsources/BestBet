package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/hint"
	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/odds"
	"github.com/lioilsources/BestBet/internal/oddspapi"
	"github.com/lioilsources/BestBet/internal/store"
	"github.com/lioilsources/BestBet/internal/timing"
)

// timingBooks is the bookmaker triple the OddsPapi histories were fetched
// with; it is part of the cache key.
var timingBooks = []string{"pinnacle", "betano.cz", "merkurxtip.cz"}

type timingLeague struct {
	Slug     string
	Fixtures int
	Cached   int
	Results  []timing.Result
}

// runTiming reads the full OddsPapi histories from the HTTP cache (offline,
// no request) and writes the plan 04 L6 report.
func runTiming(ctx context.Context, st *store.Store, leagues []string, cacheDir string, generated time.Time) (string, error) {
	c := &oddspapi.Client{Base: oddspapi.DefaultBase, HTTP: &httpcache.Client{Dir: cacheDir + "/oddspapi", Offline: true}}
	markets, err := c.Markets(ctx)
	if err != nil {
		return "", fmt.Errorf("markets (cache): %w", err)
	}
	rule := hint.Default()
	var out []timingLeague
	for _, slug := range leagues {
		rows, err := st.Q.ListOddsFixtures(ctx, dbq.ListOddsFixturesParams{Source: "oddspapi", Slug: slug})
		if err != nil {
			return "", err
		}
		tl := timingLeague{Slug: slug, Fixtures: len(rows)}
		for _, r := range rows {
			pts, err := c.HistoricalOdds(ctx, r.SourceFixtureID, timingBooks, true)
			if errors.Is(err, httpcache.ErrOffline) {
				continue
			}
			if err != nil {
				slog.Warn("timing history", "fixture", r.SourceFixtureID, "err", err)
				continue
			}
			tl.Cached++
			f := timing.Fixture{GameID: r.GameID, Start: r.StartAt.Time.UTC()}
			for _, p := range pts {
				def, ok := markets[p.MarketID]
				if !ok {
					continue
				}
				q := oddspapi.Normalize(def, p.OutcomeID, p.Price)
				if q.Market != odds.OneX2 {
					continue
				}
				f.Quotes = append(f.Quotes, canon.OddsQuote{Book: p.Book, Market: q.Market, Selection: q.Selection, Price: q.Price, CapturedAt: p.CreatedAt})
			}
			tl.Results = append(tl.Results, timing.Analyze(f, rule))
		}
		slog.Info("timing", "league", slug, "fixtures", tl.Fixtures, "cached", tl.Cached)
		out = append(out, tl)
	}
	return timingReport(out, rule, generated), nil
}

func quartiles(xs []float64) (q1, q2, q3 float64) {
	if len(xs) == 0 {
		return 0, 0, 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	at := func(f float64) float64 { return s[int(f*float64(len(s)-1)+0.5)] }
	return at(0.25), at(0.5), at(0.75)
}

func minutes(xs []float64) string {
	if len(xs) == 0 {
		return "–"
	}
	q1, q2, q3 := quartiles(xs)
	return fmt.Sprintf("%.0f (%.0f–%.0f)", q2, q1, q3)
}

func share(n, d int) string {
	if d == 0 {
		return "–"
	}
	return fmt.Sprintf("%.0f %%", 100*float64(n)/float64(d))
}

func timingReport(ls []timingLeague, rule hint.Rule, generated time.Time) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# M3b L6: načasování CZ knih proti ostrému trhu\n\n")
	w("Vygenerováno %s příkazem `backtest -timing`. Zdroj: plná historie cen OddsPapi z HTTP cache (knihy %s), bez nového requestu. Plán 04 L6, prahy pevně předem.\n\n", generated.Format("2006-01-02 15:04 MST"), strings.Join(timingBooks, ", "))
	w("- **Pohyb ostrého trhu:** Shin pravděpodobnost Pinnaclu (domácí nebo hosté) se změní o ≥ %.0f p. b. během %d min. Další pohyb nejdřív %d min po předchozím.\n", 100*timing.MoveMin, int(timing.MoveWindow.Minutes()), int(timing.MoveWindow.Minutes()))
	w("- **Reakce knihy:** její Shin pravděpodobnost téhož výběru se posune ≥ %.0f p. b. stejným směrem do začátku zápasu. **Vedla:** kniha se tak posunula už v 30 min před pohybem Pinnaclu.\n", 100*timing.ReactMin)
	w("- **Okno hintu:** souvislý úsek, kdy pravidlo `%s` dává hint (hrana ≥ %.0f p. b., EV > 0, kurz ≤ %.0f). CLV prvního hintu okna proti poslední ceně Pinnaclu před začátkem.\n", hint.Version, 100*rule.MinEdge, rule.MaxPrice)
	w("- **Šance zachytit:** cron s periodou P a náhodnou fází okno délky d uvidí s pravděpodobností min(1, d/P).\n\n")
	w("Jen popis, bez brány. Pravidlo hintu se podle toho nemění (plán 04 §1).\n")
	for _, l := range ls {
		w("\n## %s\n\n", l.Slug)
		w("Odehraných zápasů s fixture OddsPapi: %d, s historií v cache: %d.\n\n", l.Fixtures, l.Cached)
		events := 0
		type agg struct {
			n, led, reacted, within2h int
			delays                    []float64
		}
		per := map[string]*agg{}
		var toStart []float64
		for _, r := range l.Results {
			for _, e := range r.Events {
				events++
				toStart = append(toStart, e.ToStart.Hours())
				for _, rc := range e.Reactions {
					a := per[rc.Book]
					if a == nil {
						a = &agg{}
						per[rc.Book] = a
					}
					a.n++
					if rc.Led {
						a.led++
					}
					if rc.Reacted {
						a.reacted++
						a.delays = append(a.delays, rc.Delay.Minutes())
						if rc.Delay <= 2*time.Hour {
							a.within2h++
						}
					}
				}
			}
		}
		_, medStart, _ := quartiles(toStart)
		w("### Pohyby Pinnaclu a reakce knih\n\nPohybů: %d (%.2f na zápas), medián %.1f h před začátkem.\n\n", events, float64(events)/float64(max(1, l.Cached)), medStart)
		w("| Kniha | Pohybů s cenou knihy | Vedla | Zareagovala | Do 2 h | Zpoždění min, medián (kvartily) |\n|---|---|---|---|---|---|\n")
		for _, bk := range rule.Soft {
			a := per[bk]
			if a == nil {
				w("| %s | 0 | – | – | – | – |\n", bk)
				continue
			}
			w("| %s | %d | %s | %s | %s | %s |\n", bk, a.n, share(a.led, a.n), share(a.reacted, a.n), share(a.within2h, a.n), minutes(a.delays))
		}
		var wins, after []float64
		var durs []float64
		perBook := map[string]int{}
		catch := map[time.Duration]float64{}
		polls := []time.Duration{10 * time.Minute, 30 * time.Minute, time.Hour, 2 * time.Hour}
		for _, r := range l.Results {
			for _, x := range r.Windows {
				d := x.To.Sub(x.From)
				durs = append(durs, d.Minutes())
				perBook[x.First.Book]++
				for _, p := range polls {
					catch[p] += timing.CatchProb(d, p)
				}
				if !x.HasClose {
					continue
				}
				wins = append(wins, x.CLV)
				if x.AfterMove {
					after = append(after, x.CLV)
				}
			}
		}
		n := len(durs)
		w("\n### Okna hintu\n\nOken: %d (%.2f na zápas). Betano %d, MerkurXtip %d. Délka v min, medián (kvartily): %s.\n\n", n, float64(n)/float64(max(1, l.Cached)), perBook["betano.cz"], perBook["merkurxtip.cz"], minutes(durs))
		w("| Perioda cronu | Očekávaný podíl zachycených oken |\n|---|---|\n")
		for _, p := range polls {
			v := "–"
			if n > 0 {
				v = fmt.Sprintf("%.0f %%", 100*catch[p]/float64(n))
			}
			w("| %s | %s |\n", p, v)
		}
		all, am := meanCI(wins), meanCI(after)
		w("\n| Okna | n | CLV průměr | 95%% interval |\n|---|---|---|---|\n")
		w("| všechna s uzavřením | %d | %+.2f %% | %+.2f až %+.2f %% |\n", all.N, 100*all.MeanDelta1X2, 100*all.CILo, 100*all.CIHi)
		w("| otevřená do 2 h po pohybu Pinnaclu | %d | %+.2f %% | %+.2f až %+.2f %% |\n", am.N, 100*am.MeanDelta1X2, 100*am.CILo, 100*am.CIHi)
	}
	w("\n## Co z toho plyne pro provoz\n\nViz `docs/DECISIONS.md` (záznam k L6). Čísla nahoře jsou jediný podklad; interpretace se do reportu nepíše, aby šel přegenerovat.\n")
	return b.String()
}

// write prints the report or stores it in a file.
func write(path, md string) error {
	if path == "" {
		fmt.Print(md)
		return nil
	}
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	slog.Info("report written", "file", path)
	return nil
}
