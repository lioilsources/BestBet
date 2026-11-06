// Package api is the read-only HTTP API of plan 01 fáze 4:
//
//	GET /healthz                 data freshness (odds gaps over 24 h are an alert)
//	GET /matches?date=YYYY-MM-DD games of a day with the 1X2 view and the best offers
//	GET /matches/{id}/markets    all priced markets, offers per allowed book, contributions
//	GET /matches/{id}/waves      both teams' wave (Kalman level, direction, stability)
//	GET /journal                 the caller's bet journal (Bearer token)
//
// Prices are only ever shown for allowed (CZ licensed) books; every response
// that carries a price carries the § 5j warning.
package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/edge"
	"github.com/lioilsources/BestBet/internal/rating"
)

// Warning is the text required by § 5j of the Czech advertising act.
const Warning = "Ministerstvo financí varuje: Účastí na hazardní hře může vzniknout závislost."

// ErrNotFound marks a missing game.
var ErrNotFound = errors.New("not found")

// Source is what the API reads (implemented by *store.Store via an adapter).
type Source interface {
	GamesBetween(ctx context.Context, from, to time.Time) ([]canon.GameInfo, error)
	Game(ctx context.Context, id int64) (canon.GameInfo, error) // ErrNotFound
	LatestPrediction(ctx context.Context, gameID int64) (*canon.Prediction, error)
	LatestOdds(ctx context.Context, gameID int64) ([]canon.OddsQuote, error)
	LeagueMatches(ctx context.Context, slug string) ([]canon.Match, error)
	Journal(ctx context.Context, user string) ([]canon.JournalEntry, error)
	LastOddsCapture(ctx context.Context, source string) (time.Time, error)
}

// Server holds the configuration.
type Server struct {
	Src         Source
	Now         func() time.Time
	Books       map[string]bool // allowed books (CZ licence)
	Policy      edge.Policy
	Tokens      map[string]string // token -> user key
	HelpContact string            // shown next to the warning
	Loc         *time.Location    // day boundaries for /matches

	mu    sync.Mutex
	waves map[string]waveCache
}

type waveCache struct {
	at      time.Time
	matches []canon.Match
	tl      map[int64]map[canon.Side]rating.WaveState
}

// Handler returns the routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /matches", s.matches)
	mux.HandleFunc("GET /matches/{id}/markets", s.markets)
	mux.HandleFunc("GET /matches/{id}/waves", s.wavesHandler)
	mux.HandleFunc("GET /journal", s.journal)
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		slog.Error("api", "msg", msg, "err", err)
	}
	writeJSON(w, code, map[string]string{"error": msg})
}

type notice struct {
	Warning string `json:"warning"`
	Help    string `json:"help,omitempty"`
}

func (s *Server) notice() notice { return notice{Warning: Warning, Help: s.HelpContact} }

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	now := s.Now()
	out := map[string]any{"ok": true, "now": now}
	last := map[string]time.Time{}
	stale := true
	for _, src := range []string{"apihockey", "oddspapi"} {
		t, err := s.Src.LastOddsCapture(r.Context(), src)
		if err != nil {
			fail(w, http.StatusInternalServerError, "health", err)
			return
		}
		last[src] = t
		if now.Sub(t) < 24*time.Hour {
			stale = false
		}
	}
	out["odds_last_capture"] = last
	out["odds_stale"] = stale
	writeJSON(w, http.StatusOK, out)
}

// MatchSummary is one row of /matches.
type MatchSummary struct {
	canon.GameInfo
	ModelVersion string             `json:"model_version,omitempty"`
	ComputedAt   *time.Time         `json:"computed_at,omitempty"`
	P1X2         map[string]float64 `json:"p_1x2,omitempty"`
	PML          map[string]float64 `json:"p_ml,omitempty"`
	Best         []edge.Offer       `json:"best,omitempty"` // top 3 by expected log growth
}

func (s *Server) matches(w http.ResponseWriter, r *http.Request) {
	loc := s.Loc
	if loc == nil {
		loc = time.UTC
	}
	day := s.Now().In(loc)
	if d := r.URL.Query().Get("date"); d != "" {
		t, err := time.ParseInLocation("2006-01-02", d, loc)
		if err != nil {
			fail(w, http.StatusBadRequest, "date must be YYYY-MM-DD", nil)
			return
		}
		day = t
	}
	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	games, err := s.Src.GamesBetween(r.Context(), from, from.AddDate(0, 0, 1))
	if err != nil {
		fail(w, http.StatusInternalServerError, "games", err)
		return
	}
	out := make([]MatchSummary, 0, len(games))
	for _, g := range games {
		ms := MatchSummary{GameInfo: g}
		p, err := s.Src.LatestPrediction(r.Context(), g.ID)
		if err != nil {
			fail(w, http.StatusInternalServerError, "prediction", err)
			return
		}
		if p != nil {
			ms.ModelVersion, ms.ComputedAt = p.ModelVersion, &p.ComputedAt
			ms.P1X2, ms.PML = p.Markets["1x2"], p.Markets["ml"]
			q, err := s.Src.LatestOdds(r.Context(), g.ID)
			if err != nil {
				fail(w, http.StatusInternalServerError, "odds", err)
				return
			}
			offers := edge.Offers(p.Markets, q, s.Books, s.Policy)
			for _, o := range offers {
				if o.EV > 0 && len(ms.Best) < 3 {
					ms.Best = append(ms.Best, o)
				}
			}
		}
		out = append(out, ms)
	}
	writeJSON(w, http.StatusOK, map[string]any{"date": from.Format("2006-01-02"), "matches": out, "notice": s.notice()})
}

func (s *Server) game(w http.ResponseWriter, r *http.Request) (canon.GameInfo, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		fail(w, http.StatusBadRequest, "bad id", nil)
		return canon.GameInfo{}, false
	}
	g, err := s.Src.Game(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		fail(w, http.StatusNotFound, "game not found", nil)
		return g, false
	}
	if err != nil {
		fail(w, http.StatusInternalServerError, "game", err)
		return g, false
	}
	return g, true
}

func (s *Server) markets(w http.ResponseWriter, r *http.Request) {
	g, ok := s.game(w, r)
	if !ok {
		return
	}
	p, err := s.Src.LatestPrediction(r.Context(), g.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "prediction", err)
		return
	}
	out := map[string]any{"game": g, "notice": s.notice(), "policy": map[string]any{
		"edge_1x2": s.Policy.Edge1X2, "edge_other": s.Policy.EdgeOther, "kelly": s.Policy.Kelly, "max_stake": s.Policy.MaxStake,
		"clv_markets": keys(s.Policy.CLVMarkets),
	}}
	if p == nil {
		out["prediction"] = nil
		writeJSON(w, http.StatusOK, out)
		return
	}
	q, err := s.Src.LatestOdds(r.Context(), g.ID)
	if err != nil {
		fail(w, http.StatusInternalServerError, "odds", err)
		return
	}
	out["prediction"] = map[string]any{"model_version": p.ModelVersion, "computed_at": p.ComputedAt, "inputs_hash": p.InputsHash,
		"lambda_home": p.LambdaHome, "lambda_away": p.LambdaAway, "markets": p.Markets, "contributions": p.Contributions}
	out["offers"] = edge.Offers(p.Markets, q, s.Books, s.Policy)
	writeJSON(w, http.StatusOK, out)
}

func keys(m map[string]bool) []string {
	out := []string{}
	for k, v := range m {
		if v {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// Wave is one team's wave for the UI (01 §2.1: level, direction, stability).
type Wave struct {
	Team      int32       `json:"team"`
	Level     float64     `json:"level"`     // Kalman goal-difference rating
	Direction float64     `json:"direction"` // level change over ten games (velocity × 10)
	Stability float64     `json:"stability"` // sd of the level estimate
	Games     int         `json:"games"`
	Series    []WavePoint `json:"series"` // last 20 games before this one
}

// WavePoint is the pre-game state of one earlier game.
type WavePoint struct {
	Date     time.Time `json:"date"`
	MatchID  int64     `json:"match_id"`
	Level    float64   `json:"level"`
	Velocity float64   `json:"velocity"`
}

func (s *Server) timeline(ctx context.Context, league string) (waveCache, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.waves == nil {
		s.waves = map[string]waveCache{}
	}
	if c, ok := s.waves[league]; ok && s.Now().Sub(c.at) < 5*time.Minute {
		return c, nil
	}
	ms, err := s.Src.LeagueMatches(ctx, league)
	if err != nil {
		return waveCache{}, err
	}
	canon.SortMatches(ms)
	c := waveCache{at: s.Now(), matches: ms, tl: rating.WaveTimeline(ms, rating.DefaultWaveConfig())}
	s.waves[league] = c
	return c, nil
}

func (s *Server) wavesHandler(w http.ResponseWriter, r *http.Request) {
	g, ok := s.game(w, r)
	if !ok {
		return
	}
	c, err := s.timeline(r.Context(), g.League)
	if err != nil {
		fail(w, http.StatusInternalServerError, "waves", err)
		return
	}
	st, ok := c.tl[g.ID]
	if !ok {
		fail(w, http.StatusNotFound, "no wave for this game", nil)
		return
	}
	build := func(team int32, side canon.Side) Wave {
		ws := st[side]
		wv := Wave{Team: team, Level: ws.Level, Direction: 10 * ws.Velocity, Stability: ws.LevelSD, Games: ws.Games}
		for i := len(c.matches) - 1; i >= 0 && len(wv.Series) < 20; i-- {
			m := c.matches[i]
			if !m.StartAt.Before(g.StartAt) || (m.HomeTeam != team && m.AwayTeam != team) {
				continue
			}
			sd := canon.Home
			if m.AwayTeam == team {
				sd = canon.Away
			}
			x := c.tl[m.ID][sd]
			wv.Series = append(wv.Series, WavePoint{Date: m.StartAt, MatchID: m.ID, Level: x.Level, Velocity: x.Velocity})
		}
		for i, j := 0, len(wv.Series)-1; i < j; i, j = i+1, j-1 {
			wv.Series[i], wv.Series[j] = wv.Series[j], wv.Series[i]
		}
		return wv
	}
	writeJSON(w, http.StatusOK, map[string]any{"game": g, "home": build(g.HomeTeam, canon.Home), "away": build(g.AwayTeam, canon.Away),
		"note": "Vlna je popis formy; na Allsvenskan predikci nezlepšila (docs/RESULTS_PHASE2.md)."})
}

func (s *Server) journal(w http.ResponseWriter, r *http.Request) {
	tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	user := ""
	if ok {
		for t, u := range s.Tokens {
			if subtle.ConstantTimeCompare([]byte(t), []byte(tok)) == 1 {
				user = u
			}
		}
	}
	if user == "" {
		fail(w, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	es, err := s.Src.Journal(r.Context(), user)
	if err != nil {
		fail(w, http.StatusInternalServerError, "journal", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": es, "notice": s.notice()})
}
