// Package report renders DATA_M0.md (coverage per league) from the database.
package report

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/db/dbq"
)

// Data is everything the report shows; filled by Load, rendered by Render.
type Data struct {
	GeneratedAt time.Time
	Coverage    []dbq.CoverageByLeagueSeasonRow
	Margins     []dbq.MarginByLeagueRow
	Markets     []dbq.MarketsByLeagueRow
	Joins       []dbq.JoinRateBySourceRow
	Unmatched   []dbq.UnmatchedFixturesRow
	NewTeams    []dbq.TeamsCreatedBySecondaryRow
}

// Load runs the report queries.
func Load(ctx context.Context, q *dbq.Queries, now time.Time) (Data, error) {
	d := Data{GeneratedAt: now}
	var err error
	if d.Coverage, err = q.CoverageByLeagueSeason(ctx); err != nil {
		return d, fmt.Errorf("coverage: %w", err)
	}
	if d.Margins, err = q.MarginByLeague(ctx); err != nil {
		return d, fmt.Errorf("margins: %w", err)
	}
	if d.Markets, err = q.MarketsByLeague(ctx); err != nil {
		return d, fmt.Errorf("markets: %w", err)
	}
	if d.Joins, err = q.JoinRateBySource(ctx); err != nil {
		return d, fmt.Errorf("joins: %w", err)
	}
	if d.Unmatched, err = q.UnmatchedFixtures(ctx); err != nil {
		return d, fmt.Errorf("unmatched: %w", err)
	}
	if d.NewTeams, err = q.TeamsCreatedBySecondary(ctx); err != nil {
		return d, fmt.Errorf("new teams: %w", err)
	}
	return d, nil
}

// Gates from the plan (§1.4.1, §6 M0).
const (
	MinSeasons  = 3
	MinBooks    = 5
	MinJoinRate = 0.98
)

// LeagueSummary aggregates one league over seasons.
type LeagueSummary struct {
	Slug, Name, Role string
	Seasons          int
	Games            int64
	OddsCov          float64
	GoalieCov        float64
	Margin1X2        float64 // NaN = no complete 1X2 books
	MarginML         float64
	Books            int64
	JoinRate         float64 // worst source; NaN = no secondary source
	Score            float64
	Gates            []string // failed gates
}

// Summarise computes per-league summaries and the M1 ranking score.
// Score (fringe leagues only): 0.4 odds coverage + 0.4 goalie coverage + 0.2 min(books/5, 1),
// zero when any hard gate fails. It is a sorting aid; the choice stays with the owner.
func Summarise(d Data) []LeagueSummary {
	by := map[string]*LeagueSummary{}
	var order []string
	for _, c := range d.Coverage {
		s := by[c.Slug]
		if s == nil {
			s = &LeagueSummary{Slug: c.Slug, Name: c.Name, Role: c.Role, Margin1X2: math.NaN(), MarginML: math.NaN(), JoinRate: math.NaN()}
			by[c.Slug] = s
			order = append(order, c.Slug)
		}
		s.Seasons++
		s.Games += c.Games
		s.OddsCov += float64(c.GamesWithOdds)
		s.GoalieCov += float64(c.GamesWithGoalies)
	}
	for _, m := range d.Margins {
		s := by[m.Slug]
		if s == nil {
			continue
		}
		switch m.Market {
		case "1x2":
			s.Margin1X2 = m.AvgMargin
		case "ml":
			s.MarginML = m.AvgMargin
		}
		if m.Books > s.Books {
			s.Books = m.Books
		}
	}
	for _, j := range d.Joins {
		s := by[j.Slug]
		if s == nil || j.Fixtures == 0 {
			continue
		}
		r := float64(j.Matched) / float64(j.Fixtures)
		if math.IsNaN(s.JoinRate) || r < s.JoinRate {
			s.JoinRate = r
		}
	}
	out := make([]LeagueSummary, 0, len(order))
	for _, slug := range order {
		s := by[slug]
		if s.Games > 0 {
			s.OddsCov /= float64(s.Games)
			s.GoalieCov /= float64(s.Games)
		}
		if s.Seasons < MinSeasons {
			s.Gates = append(s.Gates, fmt.Sprintf("sezon %d < %d", s.Seasons, MinSeasons))
		}
		if s.Books < MinBooks {
			s.Gates = append(s.Gates, fmt.Sprintf("knih %d < %d", s.Books, MinBooks))
		}
		if !math.IsNaN(s.JoinRate) && s.JoinRate < MinJoinRate {
			s.Gates = append(s.Gates, czDecimal(fmt.Sprintf("join %.1f %% < 98 %%", 100*s.JoinRate)))
		}
		if s.Role == "fringe" && len(s.Gates) == 0 {
			s.Score = 0.4*s.OddsCov + 0.4*s.GoalieCov + 0.2*math.Min(float64(s.Books)/MinBooks, 1)
		}
		out = append(out, *s)
	}
	return out
}

// Render produces DATA_M0.md (Czech, user-facing).
func Render(d Data) string {
	var b strings.Builder
	sums := Summarise(d)
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }

	w("# DATA_M0 — pokrytí dat (hokej)\n\n")
	w("Vygenerováno `ingest-hockey report` %s. Soubor je generovaný, ručně ho neupravuj; komentáře patří do `docs/M0.md`.\n\n", d.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC"))

	w("## Souhrn per liga\n\n")
	w("| Liga | Role | Sezon | Zápasů | S kurzy | S brankáři | Marže 1X2 | Marže ML | Knih | Join | Skóre M1 | Neprošlo |\n")
	w("|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, s := range sums {
		w("| %s | %s | %d | %d | %s | %s | %s | %s | %d | %s | %s | %s |\n",
			s.Name, s.Role, s.Seasons, s.Games, pct(s.OddsCov), pct(s.GoalieCov),
			pctNaN(s.Margin1X2), pctNaN(s.MarginML), s.Books, pctNaN(s.JoinRate), score(s), strings.Join(s.Gates, "; "))
	}
	if len(sums) == 0 {
		w("| — | | | | | | | | | | | zatím žádná data |\n")
	}

	w("\n## Doporučení pro M1\n\n")
	var cands []LeagueSummary
	for _, s := range sums {
		if s.Score > 0 {
			cands = append(cands, s)
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].Score > cands[j].Score })
	switch {
	case len(cands) >= 2:
		w("Heuristika (0,4 × pokrytí kurzy + 0,4 × pokrytí brankáři + 0,2 × knihy/5, jen okrajové ligy, které prošly branami) vybírá **%s** a **%s**. Rozhoduje Olin.\n", cands[0].Name, cands[1].Name)
	case len(cands) == 1:
		w("Branami prošla jen **%s**. Druhou ligu zatím nelze doporučit; viz sloupec „Neprošlo“.\n", cands[0].Name)
	default:
		w("Žádná okrajová liga zatím neprošla branami (≥ %d sezony, ≥ %d knih, join ≥ 98 %%). Viz sloupec „Neprošlo“.\n", MinSeasons, MinBooks)
	}

	w("\n## Sezony\n\n| Liga | Sezona | Zápasů | S kurzy | S brankáři | S bruslaři | Mimo základní část | Po 60 min nerozhodnuto |\n|---|---|---|---|---|---|---|---|\n")
	for _, c := range d.Coverage {
		w("| %s | %d/%02d | %d | %d | %d | %d | %d | %d |\n", c.Name, c.Season, (c.Season+1)%100, c.Games, c.GamesWithOdds, c.GamesWithGoalies, c.GamesWithSkaters, c.GamesNonRegular, c.GamesPast60)
	}

	w("\n## Trhy v kurzech\n\n| Liga | Trh | Zápasů | Knih |\n|---|---|---|---|\n")
	for _, m := range d.Markets {
		w("| %s | %s | %d | %d |\n", m.Slug, m.Market, m.Games, m.Books)
	}

	w("\n## Join klíče per zdroj\n\n| Liga | Zdroj | Fixtures | Spárováno | Podíl |\n|---|---|---|---|---|\n")
	for _, j := range d.Joins {
		r := math.NaN()
		if j.Fixtures > 0 {
			r = float64(j.Matched) / float64(j.Fixtures)
		}
		w("| %s | %s | %d | %d | %s |\n", j.Slug, j.Source, j.Fixtures, j.Matched, pctNaN(r))
	}

	if len(d.NewTeams) > 0 {
		w("\n## Týmy známé jen sekundárnímu zdroji\n\nNejspíš chybí alias v `config/leagues.yaml`.\n\n")
		for _, t := range d.NewTeams {
			w("- %s: %s (%s)\n", t.Slug, t.Name, t.CreatedBy)
		}
	}
	if len(d.Unmatched) > 0 {
		w("\n## Nespárované fixtures (max 200)\n\n")
		for _, u := range d.Unmatched {
			w("- %s / %s `%s`: %s\n", u.Slug, u.Source, u.SourceFixtureID, u.Detail)
		}
	}
	return b.String()
}

func pct(v float64) string { return fmt.Sprintf("%.0f %%", 100*v) }

func czDecimal(s string) string { return strings.Replace(s, ".", ",", 1) }

func pctNaN(v float64) string {
	if math.IsNaN(v) {
		return "—"
	}
	return czDecimal(fmt.Sprintf("%.1f %%", 100*v))
}

func score(s LeagueSummary) string {
	if s.Role != "fringe" {
		return "kontrola"
	}
	if s.Score == 0 {
		return "—"
	}
	return czDecimal(fmt.Sprintf("%.2f", s.Score))
}
