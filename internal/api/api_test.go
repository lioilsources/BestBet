package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/edge"
	"github.com/lioilsources/BestBet/internal/live"
	"github.com/lioilsources/BestBet/internal/model/periods"
)

type fake struct {
	games []canon.Match
	pred  map[int64]*canon.Prediction
	odds  map[int64][]canon.OddsQuote
}

func (f *fake) info(m canon.Match) canon.GameInfo {
	return canon.GameInfo{Match: m, LeagueName: "Sim", HomeName: "H", AwayName: "A", Status: "scheduled"}
}
func (f *fake) GamesBetween(_ context.Context, from, to time.Time) ([]canon.GameInfo, error) {
	var out []canon.GameInfo
	for _, m := range f.games {
		if !m.StartAt.Before(from) && m.StartAt.Before(to) {
			out = append(out, f.info(m))
		}
	}
	return out, nil
}
func (f *fake) Game(_ context.Context, id int64) (canon.GameInfo, error) {
	for _, m := range f.games {
		if m.ID == id {
			return f.info(m), nil
		}
	}
	return canon.GameInfo{}, ErrNotFound
}
func (f *fake) LatestPrediction(_ context.Context, id int64) (*canon.Prediction, error) {
	return f.pred[id], nil
}
func (f *fake) LatestOdds(_ context.Context, id int64) ([]canon.OddsQuote, error) {
	return f.odds[id], nil
}
func (f *fake) LeagueMatches(context.Context, string) ([]canon.Match, error) { return f.games, nil }
func (f *fake) Journal(_ context.Context, user string) ([]canon.JournalEntry, error) {
	return []canon.JournalEntry{{ID: 1, Book: "tipsport", Market: "1x2", Selection: "home", Price: 2, Stake: 100, Note: &user}}, nil
}
func (f *fake) LastOddsCapture(context.Context, string) (time.Time, error) {
	return time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), nil
}

func setup(t *testing.T) (*Server, *fake, canon.Match) {
	t.Helper()
	games, _ := periods.Simulate(periods.DefaultSim())
	for i := range games {
		games[i].League = "sim"
	}
	now := games[len(games)-200].StartAt.Add(-time.Hour)
	for i := range games {
		if !games[i].StartAt.Before(now) {
			games[i].Reg, games[i].Final, games[i].Periods = nil, nil, nil
		}
	}
	ps, err := live.Predict(games, now, 48*time.Hour, periods.DefaultConfig())
	if err != nil || len(ps) == 0 {
		t.Fatal(err)
	}
	f := &fake{games: games, pred: map[int64]*canon.Prediction{}, odds: map[int64][]canon.OddsQuote{}}
	for i := range ps {
		f.pred[ps[i].MatchID] = &ps[i]
	}
	var target canon.Match
	for _, g := range games {
		if f.pred[g.ID] != nil {
			target = g
			break
		}
	}
	at := target.StartAt.Add(-2 * time.Hour)
	for _, b := range []string{"tipsport", "offshore"} {
		for sel, price := range map[string]float64{"home": 1.6, "draw": 6.0, "away": 9.0} {
			f.odds[target.ID] = append(f.odds[target.ID], canon.OddsQuote{Book: b, Market: "1x2", Selection: sel, Price: price, CapturedAt: at})
		}
	}
	s := &Server{Src: f, Now: func() time.Time { return now }, Books: map[string]bool{"tipsport": true}, Policy: edge.DefaultPolicy(),
		Tokens: map[string]string{"secret": "olin"}}
	return s, f, target
}

func get(t *testing.T, h http.Handler, path, token string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("%s: %v: %s", path, err, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "offshore") {
		t.Fatalf("%s shows a book without CZ licence", path)
	}
	return rec.Code, out
}

func TestAPI(t *testing.T) {
	s, _, target := setup(t)
	h := s.Handler()
	day := target.StartAt.Format("2006-01-02")
	code, out := get(t, h, "/matches?date="+day, "")
	if code != 200 || len(out["matches"].([]any)) == 0 || out["notice"].(map[string]any)["warning"] != Warning {
		t.Fatalf("matches %d %v", code, out)
	}
	code, out = get(t, h, "/matches/"+itoa(target.ID)+"/markets", "")
	if code != 200 || out["prediction"] == nil || len(out["offers"].([]any)) != 3 {
		t.Fatalf("markets %d %v", code, out)
	}
	for _, o := range out["offers"].([]any) {
		if o.(map[string]any)["green"] == true {
			t.Fatal("nothing may be green without CLV-proven markets")
		}
	}
	code, out = get(t, h, "/matches/"+itoa(target.ID)+"/waves", "")
	if code != 200 || len(out["home"].(map[string]any)["series"].([]any)) != 20 {
		t.Fatalf("waves %d %v", code, out)
	}
	if code, _ := get(t, h, "/matches/999999999/markets", ""); code != 404 {
		t.Fatalf("missing game %d", code)
	}
	if code, _ := get(t, h, "/journal", "wrong"); code != 401 {
		t.Fatalf("journal without token %d", code)
	}
	if code, out := get(t, h, "/journal", "secret"); code != 200 || len(out["entries"].([]any)) != 1 {
		t.Fatalf("journal %d %v", code, out)
	}
	if code, out := get(t, h, "/healthz", ""); code != 200 || out["odds_stale"] != true {
		t.Fatalf("healthz %d %v", code, out)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
