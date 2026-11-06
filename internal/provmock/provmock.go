// Package provmock is a deterministic stand-in for the paid data providers
// (API-HOCKEY and OddsPapi) so the whole ingest → backtest pipeline runs without
// credentials. It serves the same JSON shapes the real clients parse:
//
//	<base>/apihockey/leagues|games|odds
//	<base>/oddspapi/tournaments|fixtures|markets|historical-odds
//
// The world is generated from the league config: 12 mock teams per league,
// a double round robin per season, results from a Poisson model with fixed team
// strengths, and prices that move from a noisy opening line toward the fair
// price as kick-off approaches (so open → close movement and CLV are testable).
//
// Mock data is never real: team names start with "Mock", books with "mock-",
// and cmd/ingest-hockey refuses mock mode unless the database name ends in _mock.
package provmock

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/config"
	"github.com/lioilsources/BestBet/internal/model/markets"
	"github.com/lioilsources/BestBet/internal/model/periods"
)

// Books are the mock bookmakers: a sharp low-margin book and two "CZ" books.
var Books = []Book{{"mock-sharp", 0.025, 0.6}, {"mock-cz1", 0.06, 1.0}, {"mock-cz2", 0.08, 1.2}}

// Book is a mock bookmaker: margin on the fair price and noise multiplier.
type Book struct {
	Slug   string
	Margin float64
	Noise  float64
}

// World is the generated universe.
type World struct {
	Seed    uint64
	Now     func() time.Time
	Leagues []League
}

// League is one mock league.
type League struct {
	Cfg          config.League
	APIID        int
	TournamentID int
	Teams        []Team
	Games        []Game
}

// Team is a mock team with fixed strengths (log scale).
type Team struct {
	ID       int
	Name     string
	Att, Def float64
}

// Game is a scheduled mock game; the result is drawn deterministically.
type Game struct {
	ID         int
	Season     int
	Start      time.Time
	Home, Away int // team index
}

// New builds the world for the configured leagues and seasons.
func New(cfg config.File, seed uint64, now func() time.Time) *World {
	w := &World{Seed: seed, Now: now}
	for li, lc := range cfg.Leagues {
		r := rand.New(rand.NewPCG(seed, uint64(li)+1))
		l := League{Cfg: lc, APIID: 900 + li, TournamentID: 9000 + li}
		for t := 0; t < 12; t++ {
			l.Teams = append(l.Teams, Team{ID: (li+1)*100 + t, Name: fmt.Sprintf("Mock %s %02d", lc.Name, t+1),
				Att: r.NormFloat64() * 0.15, Def: r.NormFloat64() * 0.15})
		}
		for _, s := range cfg.Seasons {
			l.Games = append(l.Games, schedule(li, s, len(l.Teams))...)
		}
		w.Leagues = append(w.Leagues, l)
	}
	return w
}

// schedule is a double round robin (circle method), two rounds a week.
func schedule(li, season, n int) []Game {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	start := time.Date(season, 9, 15, 17, 0, 0, 0, time.UTC).Add(time.Duration(li) * time.Hour)
	var out []Game
	k := 0
	for leg := 0; leg < 2; leg++ {
		for round := 0; round < n-1; round++ {
			day := start.Add(time.Duration(leg*(n-1)+round) * 84 * time.Hour) // 3.5 days
			for i := 0; i < n/2; i++ {
				h, a := idx[i], idx[n-1-i]
				if (round+leg)%2 == 1 {
					h, a = a, h
				}
				out = append(out, Game{ID: (li+1)*10_000_000 + season*1000 + k, Season: season, Start: day, Home: h, Away: a})
				k++
			}
			// rotate all but the first
			last := idx[n-1]
			copy(idx[2:], idx[1:n-1])
			idx[1] = last
		}
	}
	return out
}

func (l *League) lambdas(g Game) (float64, float64) {
	h, a := l.Teams[g.Home], l.Teams[g.Away]
	mu := math.Log(2.8)
	return math.Exp(mu + 0.08 + h.Att - a.Def), math.Exp(mu + a.Att - h.Def)
}

// Fair returns the true 1x2, ml and total 5.5 probabilities of a game.
func (l *League) Fair(g Game) canon.MarketProbs {
	lh, la := l.lambdas(g)
	reg := periods.Matrix(lh, la, 12, 0.3)
	p1 := periods.Matrix(lh*0.31, la*0.31, 12, 0)
	return markets.Derive(reg, p1, 0.52)
}

type result struct {
	periods [3][2]int
	ot, so  *[2]int
	short   string
	fin     [2]int
}

func (w *World) result(l *League, g Game) result {
	r := rand.New(rand.NewPCG(w.Seed^0xabcdef, uint64(g.ID)))
	lh, la := l.lambdas(g)
	var res result
	prof := [3]float64{0.31, 0.33, 0.36}
	for k := 0; k < 3; k++ {
		res.periods[k] = [2]int{poisson(r, lh*prof[k]), poisson(r, la*prof[k])}
		res.fin[0] += res.periods[k][0]
		res.fin[1] += res.periods[k][1]
	}
	res.short = "FT"
	if res.fin[0] == res.fin[1] {
		homeWins := r.Float64() < 0.52
		if r.Float64() < 0.6 {
			res.short = "AOT"
			ot := [2]int{0, 1}
			if homeWins {
				ot = [2]int{1, 0}
			}
			res.ot = &ot
		} else {
			res.short = "AP"
			ot, so := [2]int{0, 0}, [2]int{2, 3}
			if homeWins {
				so = [2]int{3, 2}
			}
			res.ot, res.so = &ot, &so
		}
		if homeWins {
			res.fin[0]++
		} else {
			res.fin[1]++
		}
	}
	return res
}

func poisson(r *rand.Rand, lambda float64) int {
	l, k, p := math.Exp(-lambda), 0, 1.0
	for {
		p *= r.Float64()
		if p <= l {
			return k
		}
		k++
	}
}

// noise is a deterministic N(0,1) draw keyed by the parts.
func (w *World) noise(parts ...any) float64 {
	h := fnv.New64a()
	fmt.Fprint(h, w.Seed)
	for _, p := range parts {
		fmt.Fprint(h, "|", p)
	}
	r := rand.New(rand.NewPCG(h.Sum64(), 1))
	return r.NormFloat64()
}

// Price is book b's price for one selection at time at. The opening error is
// fixed per game and book and shrinks linearly to a small residual at kick-off;
// a small per-capture jitter is added. Prices carry the book margin.
func (w *World) Price(l *League, g Game, b Book, market, sel string, at time.Time) float64 {
	fair := l.Fair(g)[market]
	hours := g.Start.Sub(at).Hours()
	shrink := math.Min(1, math.Max(hours, 0)/120)
	keys := sortedKeys(fair)
	p := map[string]float64{}
	tot := 0.0
	for _, k := range keys {
		e := b.Noise * (0.03 + 0.12*shrink) * w.noise("open", g.ID, b.Slug, market, k)
		e += 0.01 * w.noise("jit", g.ID, b.Slug, market, k, at.Truncate(time.Hour).Unix())
		p[k] = fair[k] * math.Exp(e)
		tot += p[k]
	}
	q := p[sel] / tot
	return math.Round(100/(q*(1+b.Margin))) / 100
}

func sortedKeys(m map[string]float64) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// Handler serves both providers.
func (w *World) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/apihockey/leagues", w.ahLeagues)
	mux.HandleFunc("/apihockey/games", w.ahGames)
	mux.HandleFunc("/apihockey/odds", w.ahOdds)
	mux.HandleFunc("/oddspapi/tournaments", w.opTournaments)
	mux.HandleFunc("/oddspapi/fixtures", w.opFixtures)
	mux.HandleFunc("/oddspapi/markets", w.opMarkets)
	mux.HandleFunc("/oddspapi/historical-odds", w.opHistorical)
	return mux
}

func writeJSON(rw http.ResponseWriter, v any) {
	rw.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(rw).Encode(v)
}

func envelope(rw http.ResponseWriter, resp any, n int) {
	rw.Header().Set("x-ratelimit-requests-remaining", "99")
	writeJSON(rw, map[string]any{"errors": []any{}, "results": n, "paging": map[string]int{"current": 1, "total": 1}, "response": resp})
}

func (w *World) league(apiID int) *League {
	for i := range w.Leagues {
		if w.Leagues[i].APIID == apiID || w.Leagues[i].TournamentID == apiID {
			return &w.Leagues[i]
		}
	}
	return nil
}

func (w *World) ahLeagues(rw http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("search"))
	var out []map[string]any
	for _, l := range w.Leagues {
		names := append([]string{l.Cfg.APIHockey.Search}, l.Cfg.APIHockey.Alt...)
		hit := false
		for _, n := range names {
			if strings.Contains(strings.ToLower(n), q) || strings.Contains(q, strings.ToLower(n)) {
				hit = true
			}
		}
		if !hit {
			continue
		}
		var seasons []map[string]any
		for _, g := range seasonsOf(l) {
			seasons = append(seasons, map[string]any{"season": g, "current": g == seasonAt(w.Now()), "start": fmt.Sprintf("%d-09-01", g), "end": fmt.Sprintf("%d-04-30", g+1)})
		}
		out = append(out, map[string]any{"id": l.APIID, "name": l.Cfg.APIHockey.Search, "type": "league",
			"country": map[string]string{"name": l.Cfg.APIHockey.Country, "code": "XX"}, "seasons": seasons})
	}
	envelope(rw, out, len(out))
}

func seasonsOf(l League) []int {
	seen := map[int]bool{}
	var out []int
	for _, g := range l.Games {
		if !seen[g.Season] {
			seen[g.Season] = true
			out = append(out, g.Season)
		}
	}
	return out
}

func seasonAt(t time.Time) int {
	if t.Month() >= time.July {
		return t.Year()
	}
	return t.Year() - 1
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func (w *World) ahGames(rw http.ResponseWriter, r *http.Request) {
	l := w.league(atoi(r.URL.Query().Get("league")))
	season := atoi(r.URL.Query().Get("season"))
	if l == nil {
		envelope(rw, []any{}, 0)
		return
	}
	now := w.Now()
	var out []map[string]any
	for _, g := range l.Games {
		if g.Season != season {
			continue
		}
		h, a := l.Teams[g.Home], l.Teams[g.Away]
		item := map[string]any{
			"id": g.ID, "date": g.Start.Format(time.RFC3339), "timestamp": g.Start.Unix(), "week": "Regular Season",
			"league": map[string]any{"id": l.APIID, "name": l.Cfg.APIHockey.Search, "season": season},
			"teams":  map[string]any{"home": map[string]any{"id": h.ID, "name": h.Name}, "away": map[string]any{"id": a.ID, "name": a.Name}},
		}
		switch {
		case g.Start.After(now):
			item["status"] = map[string]string{"long": "Not Started", "short": "NS"}
			item["scores"] = map[string]any{"home": nil, "away": nil}
			item["periods"] = map[string]any{"first": nil, "second": nil, "third": nil, "overtime": nil, "penalties": nil}
		case g.Start.Add(3 * time.Hour).After(now):
			item["status"] = map[string]string{"long": "Second Period", "short": "P2"}
			item["scores"] = map[string]any{"home": nil, "away": nil}
			item["periods"] = map[string]any{"first": nil, "second": nil, "third": nil, "overtime": nil, "penalties": nil}
		default:
			res := w.result(l, g)
			sc := func(p [2]int) string { return fmt.Sprintf("%d-%d", p[0], p[1]) }
			per := map[string]any{"first": sc(res.periods[0]), "second": sc(res.periods[1]), "third": sc(res.periods[2]), "overtime": nil, "penalties": nil}
			if res.ot != nil {
				per["overtime"] = sc(*res.ot)
			}
			if res.so != nil {
				per["penalties"] = sc(*res.so)
			}
			item["status"] = map[string]string{"long": "Finished", "short": res.short}
			item["scores"] = map[string]any{"home": res.fin[0], "away": res.fin[1]}
			item["periods"] = per
		}
		out = append(out, item)
	}
	envelope(rw, out, len(out))
}

// apiHockeyBets maps our markets onto API-HOCKEY bet names and values.
var apiHockeyBets = []struct {
	name, market string
	values       map[string]string // our selection -> API value
}{
	{"3Way Result", "1x2", map[string]string{"home": "Home", "draw": "Draw", "away": "Away"}},
	{"Home/Away", "ml", map[string]string{"home": "Home", "away": "Away"}},
	{"Over/Under", "total", map[string]string{"over@5.5": "Over 5.5", "under@5.5": "Under 5.5"}},
}

func (w *World) ahOdds(rw http.ResponseWriter, r *http.Request) {
	l := w.league(atoi(r.URL.Query().Get("league")))
	if l == nil {
		envelope(rw, []any{}, 0)
		return
	}
	now := w.Now()
	var out []map[string]any
	for _, g := range l.Games {
		if !g.Start.After(now) || g.Start.After(now.Add(7*24*time.Hour)) {
			continue
		}
		var books []map[string]any
		for bi, b := range Books {
			var bets []map[string]any
			for i, bet := range apiHockeyBets {
				var vals []map[string]string
				for _, sel := range sortedSel(bet.values) {
					vals = append(vals, map[string]string{"value": bet.values[sel], "odd": strconv.FormatFloat(w.Price(l, g, b, bet.market, sel, now), 'f', 2, 64)})
				}
				bets = append(bets, map[string]any{"id": i + 1, "name": bet.name, "values": vals})
			}
			books = append(books, map[string]any{"id": bi + 1, "name": b.Slug, "bets": bets})
		}
		out = append(out, map[string]any{"game": map[string]int{"id": g.ID}, "update": now.Format(time.RFC3339), "bookmakers": books})
	}
	envelope(rw, out, len(out))
}

func sortedSel(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func (w *World) opTournaments(rw http.ResponseWriter, _ *http.Request) {
	var out []map[string]any
	for _, l := range w.Leagues {
		out = append(out, map[string]any{"tournamentId": l.TournamentID, "tournamentSlug": l.Cfg.Slug,
			"tournamentName": l.Cfg.OddsPapi.Search, "categoryName": l.Cfg.OddsPapi.Category})
	}
	writeJSON(rw, out)
}

func (w *World) opFixtures(rw http.ResponseWriter, r *http.Request) {
	l := w.league(atoi(r.URL.Query().Get("tournamentId")))
	from, _ := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	to, _ := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	out := []map[string]any{}
	if l != nil {
		for _, g := range l.Games {
			if g.Start.Before(from) || !g.Start.Before(to.Add(24*time.Hour)) {
				continue
			}
			status := 0
			if g.Start.Add(3 * time.Hour).Before(w.Now()) {
				status = 2
			}
			out = append(out, map[string]any{"fixtureId": fmt.Sprintf("mock-%d", g.ID), "participant1Name": l.Teams[g.Home].Name,
				"participant2Name": l.Teams[g.Away].Name, "startTime": g.Start.Format(time.RFC3339), "statusId": status})
		}
	}
	writeJSON(rw, out)
}

// OddsPapi catalogue ids used by the mock.
var opMarkets = []struct {
	id, name string
	handicap *float64
	market   string
	outcomes [][2]string // outcomeId, outcomeName / our selection
	sels     []string
}{
	{"101", "1X2", nil, "1x2", [][2]string{{"101", "1"}, {"102", "X"}, {"103", "2"}}, []string{"home", "draw", "away"}},
	{"1010", "Over/Under", ptr(5.5), "total", [][2]string{{"1", "Over"}, {"2", "Under"}}, []string{"over@5.5", "under@5.5"}},
}

func ptr(f float64) *float64 { return &f }

func (w *World) opMarkets(rw http.ResponseWriter, _ *http.Request) {
	var out []map[string]any
	for _, m := range opMarkets {
		var outs []map[string]string
		for _, o := range m.outcomes {
			outs = append(outs, map[string]string{"outcomeId": o[0], "outcomeName": o[1]})
		}
		item := map[string]any{"marketId": m.id, "marketName": m.name, "outcomes": outs}
		if m.handicap != nil {
			item["handicap"] = *m.handicap
		}
		out = append(out, item)
	}
	writeJSON(rw, out)
}

// History returns the capture times of a game's price history: every 12 h from
// T−5 d, then T−1 h and T−5 min, never after now.
func History(start, now time.Time) []time.Time {
	var ts []time.Time
	for t := start.Add(-120 * time.Hour); t.Before(start.Add(-time.Hour)); t = t.Add(12 * time.Hour) {
		ts = append(ts, t)
	}
	ts = append(ts, start.Add(-time.Hour), start.Add(-5*time.Minute))
	var out []time.Time
	for _, t := range ts {
		if !t.After(now) {
			out = append(out, t)
		}
	}
	return out
}

func (w *World) opHistorical(rw http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Query().Get("fixtureId"), "mock-")
	gid := atoi(id)
	var l *League
	var g Game
	for i := range w.Leagues {
		for _, x := range w.Leagues[i].Games {
			if x.ID == gid {
				l, g = &w.Leagues[i], x
			}
		}
	}
	books := map[string]any{}
	if l != nil {
		for _, slug := range strings.Split(r.URL.Query().Get("bookmakers"), ",") {
			b := bookBySlug(slug)
			mk := map[string]any{}
			for _, m := range opMarkets {
				outs := map[string]any{}
				for i, o := range m.outcomes {
					var pts []map[string]any
					for _, t := range History(g.Start, w.Now()) {
						pts = append(pts, map[string]any{"createdAt": t.Format(time.RFC3339), "price": w.Price(l, g, b, m.market, m.sels[i], t), "active": true})
					}
					outs[o[0]] = map[string]any{"players": map[string]any{"0": pts}}
				}
				mk[m.id] = map[string]any{"outcomes": outs}
			}
			books[b.Slug] = map[string]any{"markets": mk}
		}
	}
	writeJSON(rw, map[string]any{"fixtureId": "mock-" + id, "bookmakers": books})
}

// bookBySlug returns the mock book; unknown slugs (real book names passed on
// the command line) are mapped onto mock books so nothing real is implied.
func bookBySlug(slug string) Book {
	for _, b := range Books {
		if b.Slug == slug {
			return b
		}
	}
	h := fnv.New32a()
	h.Write([]byte(slug))
	b := Books[int(h.Sum32())%len(Books)]
	b.Slug = "mock-" + slug
	return b
}
