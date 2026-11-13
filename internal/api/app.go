package api

import (
	"context"
	"io/fs"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/hint"
	"github.com/lioilsources/BestBet/internal/lineup"
)

// AppSource is what the app endpoints (plan 03) read on top of Source.
type AppSource interface {
	OddsHistory(ctx context.Context, gameID int64) ([]canon.OddsQuote, error)
	Hints(ctx context.Context, rule string) ([]hint.Entry, error)
	Coverage(ctx context.Context) (any, error) // slow: full-table counts
	// Lineups: missing regulars of both sides (estimate or published lineup);
	// nil when the league has no player data.
	Lineups(ctx context.Context, g canon.GameInfo) (*lineup.Game, error)
}

// App holds the app endpoints' configuration.
type App struct {
	Src     AppSource
	Rule    hint.Rule
	Reports fs.FS // RESULTS_*.md and DECISIONS.md

	mu       sync.Mutex
	coverage any
	covAt    time.Time
	covBusy  bool
}

func (s *Server) appRoutes(mux *http.ServeMux) {
	if s.App == nil {
		return
	}
	mux.HandleFunc("GET /api/board", s.board)
	mux.HandleFunc("GET /api/games/{id}", s.gameDetail)
	mux.HandleFunc("GET /api/eval", s.eval)
	mux.HandleFunc("GET /api/data", s.data)
	mux.HandleFunc("GET /api/reports", s.reports)
	mux.HandleFunc("GET /api/reports/{name}", s.report)
}

// BookView is one allowed book's 1X2 for a game.
type BookView struct {
	Book   string             `json:"book"`
	Prices map[string]float64 `json:"prices"`
	Fair   *[3]float64        `json:"fair,omitempty"` // Shin, when all three prices exist
	Margin float64            `json:"margin,omitempty"`
	At     time.Time          `json:"at"`
}

// BestPrice is the highest allowed price of one selection.
type BestPrice struct {
	Book  string  `json:"book"`
	Price float64 `json:"price"`
}

// BoardRow is one game of the board.
type BoardRow struct {
	canon.GameInfo
	ModelVersion string               `json:"model_version,omitempty"`
	PModel       *[3]float64          `json:"p_model,omitempty"`
	Sharp        *[3]float64          `json:"sharp,omitempty"`      // sharp market's fair 1X2, latest
	SharpOpen    *[3]float64          `json:"sharp_open,omitempty"` // at the opening price
	SharpAt      *time.Time           `json:"sharp_at,omitempty"`
	Books        []BookView           `json:"books"`
	Best         map[string]BestPrice `json:"best,omitempty"`
	Hint         *hint.Hint           `json:"hint,omitempty"`     // the rule on the latest prices
	Recorded     *hint.Entry          `json:"recorded,omitempty"` // first qualifying price (ledger)
	Lineup       *lineup.Game         `json:"lineup,omitempty"`   // upcoming games only
}

func (s *Server) boardRow(ctx context.Context, g canon.GameInfo, recorded map[int64]hint.Entry) (BoardRow, []canon.OddsQuote, error) {
	row := BoardRow{GameInfo: g, Books: []BookView{}}
	p, err := s.Src.LatestPrediction(ctx, g.ID)
	if err != nil {
		return row, nil, err
	}
	if p != nil {
		x := p.Markets["1x2"]
		row.ModelVersion, row.PModel = p.ModelVersion, &[3]float64{x["home"], x["draw"], x["away"]}
	}
	hist, err := s.App.Src.OddsHistory(ctx, g.ID)
	if err != nil {
		return row, nil, err
	}
	if g.Status == "scheduled" {
		if row.Lineup, err = s.App.Src.Lineups(ctx, g); err != nil {
			return row, nil, err
		}
	}
	latest := hint.Books1X2(hist)
	if f, ok := hint.Fair(latest[s.App.Rule.Sharp]); ok {
		row.Sharp = &f
		t := latest[s.App.Rule.Sharp]["home"].CapturedAt
		row.SharpAt = &t
	}
	var open []canon.OddsQuote
	for _, q := range hist {
		if q.Book == s.App.Rule.Sharp && q.Kind == "opening" {
			open = append(open, q)
		}
	}
	if f, ok := hint.Fair(hint.Books1X2(open)[s.App.Rule.Sharp]); ok {
		row.SharpOpen = &f
	}
	books := make([]string, 0, len(latest))
	for b := range latest {
		if s.Books[b] {
			books = append(books, b)
		}
	}
	sort.Strings(books)
	row.Best = map[string]BestPrice{}
	for _, b := range books {
		v := BookView{Book: b, Prices: map[string]float64{}}
		inv := 0.0
		for sel, q := range latest[b] {
			v.Prices[sel] = q.Price
			inv += 1 / q.Price
			if q.CapturedAt.After(v.At) {
				v.At = q.CapturedAt
			}
			if cur, ok := row.Best[sel]; !ok || q.Price > cur.Price {
				row.Best[sel] = BestPrice{Book: b, Price: q.Price}
			}
		}
		if f, ok := hint.Fair(latest[b]); ok {
			v.Fair, v.Margin = &f, inv-1
		}
		row.Books = append(row.Books, v)
	}
	if g.Status == "scheduled" && g.StartAt.After(s.Now()) {
		row.Hint = s.App.Rule.Find(hist)
	}
	if e, ok := recorded[g.ID]; ok {
		row.Recorded = &e
	}
	return row, hist, nil
}

func (s *Server) recordedHints(ctx context.Context) (map[int64]hint.Entry, []hint.Entry, error) {
	es, err := s.App.Src.Hints(ctx, hint.Version)
	if err != nil {
		return nil, nil, err
	}
	m := make(map[int64]hint.Entry, len(es))
	for _, e := range es {
		m[e.GameID] = e
	}
	return m, es, nil
}

func (s *Server) board(w http.ResponseWriter, r *http.Request) {
	hours := 48
	if h, err := strconv.Atoi(r.URL.Query().Get("hours")); err == nil && h > 0 && h <= 168 {
		hours = h
	}
	now := s.Now()
	games, err := s.Src.GamesBetween(r.Context(), now.Add(-3*time.Hour), now.Add(time.Duration(hours)*time.Hour))
	if err != nil {
		fail(w, http.StatusInternalServerError, "games", err)
		return
	}
	rec, _, err := s.recordedHints(r.Context())
	if err != nil {
		fail(w, http.StatusInternalServerError, "hints", err)
		return
	}
	out := make([]BoardRow, 0, len(games))
	for _, g := range games {
		row, _, err := s.boardRow(r.Context(), g, rec)
		if err != nil {
			fail(w, http.StatusInternalServerError, "board", err)
			return
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"now": now, "games": out, "rule": s.ruleInfo(), "notice": s.notice()})
}

func (s *Server) ruleInfo() map[string]any {
	return map[string]any{"version": hint.Version, "min_edge": s.App.Rule.MinEdge, "max_price": s.App.Rule.MaxPrice,
		"soft": s.App.Rule.Soft, "status": "ve zkoušce"}
}

// Point is one moment of a book's 1X2 history (carried forward per selection).
type Point struct {
	At     time.Time          `json:"at"`
	Fair   [3]float64         `json:"fair"`
	Prices map[string]float64 `json:"prices,omitempty"` // allowed books only
}

// history turns a game's quotes into per-book fair-probability series.
func (s *Server) history(hist []canon.OddsQuote) map[string][]Point {
	var qs []canon.OddsQuote
	for _, q := range hist {
		if q.Market == "1x2" && (q.Book == s.App.Rule.Sharp || s.Books[q.Book]) {
			qs = append(qs, q)
		}
	}
	sort.SliceStable(qs, func(i, j int) bool {
		if !qs[i].CapturedAt.Equal(qs[j].CapturedAt) {
			return qs[i].CapturedAt.Before(qs[j].CapturedAt)
		}
		return qs[i].Book < qs[j].Book
	})
	out := map[string][]Point{}
	cur := map[string]map[string]canon.OddsQuote{}
	for i, q := range qs {
		if cur[q.Book] == nil {
			cur[q.Book] = map[string]canon.OddsQuote{}
		}
		cur[q.Book][q.Selection] = q
		// Emit once per book and timestamp, after the last quote of that moment.
		if i+1 < len(qs) && qs[i+1].Book == q.Book && qs[i+1].CapturedAt.Equal(q.CapturedAt) {
			continue
		}
		f, ok := hint.Fair(cur[q.Book])
		if !ok {
			continue
		}
		p := Point{At: q.CapturedAt, Fair: f}
		if s.Books[q.Book] {
			p.Prices = map[string]float64{}
			for sel, x := range cur[q.Book] {
				p.Prices[sel] = x.Price
			}
		}
		ps := out[q.Book]
		if n := len(ps); n > 0 && ps[n-1].Fair == f {
			continue
		}
		out[q.Book] = append(ps, p)
	}
	name := map[string][]Point{}
	for b, ps := range out {
		if b == s.App.Rule.Sharp {
			b = "sharp"
		}
		name[b] = ps
	}
	return name
}

func (s *Server) gameDetail(w http.ResponseWriter, r *http.Request) {
	g, ok := s.game(w, r)
	if !ok {
		return
	}
	rec, _, err := s.recordedHints(r.Context())
	if err != nil {
		fail(w, http.StatusInternalServerError, "hints", err)
		return
	}
	row, hist, err := s.boardRow(r.Context(), g, rec)
	if err != nil {
		fail(w, http.StatusInternalServerError, "game", err)
		return
	}
	out := map[string]any{"game": row, "history": s.history(hist), "rule": s.ruleInfo(), "notice": s.notice()}
	p, err := s.Src.LatestPrediction(r.Context(), g.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "prediction", err)
		return
	}
	if p != nil {
		out["prediction"] = map[string]any{"model_version": p.ModelVersion, "computed_at": p.ComputedAt,
			"lambda_home": p.LambdaHome, "lambda_away": p.LambdaAway, "markets": p.Markets, "contributions": p.Contributions}
	}
	writeJSON(w, http.StatusOK, out)
}

// EvalRow is one ledger entry with its outcome.
type EvalRow struct {
	hint.Entry
	Game   canon.GameInfo `json:"game"`
	PClose *float64       `json:"p_close,omitempty"` // sharp close of the selection
	CLV    *float64       `json:"clv,omitempty"`
	Result string         `json:"result,omitempty"` // win | loss (60 min)
	Profit *float64       `json:"profit,omitempty"` // flat 1 unit
}

// EvalSummary aggregates the ledger (plan 03 §3 brána M2-hint).
type EvalSummary struct {
	N        int     `json:"n"`
	NClose   int     `json:"n_close"`
	CLV      float64 `json:"clv"`
	CLVLo    float64 `json:"clv_lo"`
	CLVHi    float64 `json:"clv_hi"`
	Settled  int     `json:"settled"`
	Wins     int     `json:"wins"`
	Yield    float64 `json:"yield"`
	Need     int     `json:"need"`
	GatePass bool    `json:"gate_pass"`
}

func (s *Server) eval(w http.ResponseWriter, r *http.Request) {
	_, es, err := s.recordedHints(r.Context())
	if err != nil {
		fail(w, http.StatusInternalServerError, "hints", err)
		return
	}
	now := s.Now()
	rows := make([]EvalRow, 0, len(es))
	var clvs []float64
	sum := EvalSummary{Need: 300}
	profit := 0.0
	for _, e := range es {
		g, err := s.Src.Game(r.Context(), e.GameID)
		if err != nil {
			fail(w, http.StatusInternalServerError, "game", err)
			return
		}
		row := EvalRow{Entry: e, Game: g}
		if !g.StartAt.After(now) {
			q, err := s.Src.LatestOdds(r.Context(), g.ID)
			if err != nil {
				fail(w, http.StatusInternalServerError, "odds", err)
				return
			}
			if f, ok := hint.Fair(hint.Books1X2(q)[s.App.Rule.Sharp]); ok {
				if i := hint.Index(e.Selection); i >= 0 {
					pc, c := f[i], hint.CLV(e.Price, f[i])
					row.PClose, row.CLV = &pc, &c
					clvs = append(clvs, c)
				}
			}
		}
		if g.Reg != nil {
			won := map[string]bool{"home": g.Reg.Home > g.Reg.Away, "draw": g.Reg.Home == g.Reg.Away, "away": g.Reg.Home < g.Reg.Away}[e.Selection]
			pr := -1.0
			row.Result = "loss"
			if won {
				pr, row.Result = e.Price-1, "win"
				sum.Wins++
			}
			row.Profit = &pr
			profit += pr
			sum.Settled++
		}
		rows = append(rows, row)
	}
	sum.N, sum.NClose = len(es), len(clvs)
	if n := float64(len(clvs)); n > 0 {
		for _, c := range clvs {
			sum.CLV += c
		}
		sum.CLV /= n
		if n > 1 {
			v := 0.0
			for _, c := range clvs {
				v += (c - sum.CLV) * (c - sum.CLV)
			}
			se := math.Sqrt(v / (n - 1) / n)
			sum.CLVLo, sum.CLVHi = sum.CLV-1.96*se, sum.CLV+1.96*se
		}
	}
	if sum.Settled > 0 {
		sum.Yield = profit / float64(sum.Settled)
	}
	sum.GatePass = sum.NClose >= sum.Need && sum.CLVLo > 0
	writeJSON(w, http.StatusOK, map[string]any{"rule": s.ruleInfo(), "summary": sum, "hints": rows, "notice": s.notice()})
}

// data serves the coverage counts, refreshed in the background every 30 min
// (the counts scan the whole odds table and take tens of seconds on JODA).
func (s *Server) data(w http.ResponseWriter, r *http.Request) {
	a := s.App
	a.mu.Lock()
	c, at, busy := a.coverage, a.covAt, a.covBusy
	if !busy && (c == nil || s.Now().Sub(at) > 30*time.Minute) {
		a.covBusy = true
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			v, err := a.Src.Coverage(ctx)
			a.mu.Lock()
			defer a.mu.Unlock()
			a.covBusy = false
			if err != nil {
				slog.Error("api", "msg", "coverage", "err", err)
				return
			}
			a.coverage, a.covAt = v, s.Now()
		}()
	}
	a.mu.Unlock()
	if c == nil {
		writeJSON(w, http.StatusOK, map[string]any{"loading": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"coverage": c, "at": at})
}

// ReportInfo names one embedded report.
type ReportInfo struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

func (s *Server) reports(w http.ResponseWriter, _ *http.Request) {
	ents, err := fs.ReadDir(s.App.Reports, ".")
	if err != nil {
		fail(w, http.StatusInternalServerError, "reports", err)
		return
	}
	out := []ReportInfo{}
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := fs.ReadFile(s.App.Reports, e.Name())
		if err != nil {
			continue
		}
		title, _, _ := strings.Cut(string(b), "\n")
		out = append(out, ReportInfo{Name: strings.TrimSuffix(e.Name(), ".md"), Title: strings.TrimLeft(title, "# ")})
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": out})
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if strings.ContainsAny(name, "/\\.") {
		fail(w, http.StatusBadRequest, "bad name", nil)
		return
	}
	b, err := fs.ReadFile(s.App.Reports, name+".md")
	if err != nil {
		fail(w, http.StatusNotFound, "report not found", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "markdown": string(b)})
}
