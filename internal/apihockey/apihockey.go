// Package apihockey is a client for API-HOCKEY (api-sports.io, v1): leagues, games
// with period scores, and pre-match odds. Free tier = 100 requests/day, so every
// call goes through httpcache with a budget.
package apihockey

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/hockey"
	"github.com/lioilsources/BestBet/internal/httpcache"
)

// DefaultBase is the public endpoint.
const DefaultBase = "https://v1.hockey.api-sports.io"

// Client wraps httpcache with the API key header.
type Client struct {
	Base string
	HTTP *httpcache.Client
	// Remaining is the daily quota left as reported by the last response (-1 = unknown).
	Remaining int
}

// New builds a client. key goes to the x-apisports-key header, never to the cache key.
func New(key string, hc *httpcache.Client) *Client {
	c := &Client{Base: DefaultBase, HTTP: hc, Remaining: -1}
	if hc.Header == nil {
		hc.Header = http.Header{}
	}
	hc.Header.Set("x-apisports-key", key)
	hc.OnReply = func(r *http.Response) {
		if v := r.Header.Get("x-ratelimit-requests-remaining"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				c.Remaining = n
			}
		}
	}
	return c
}

type envelope struct {
	Errors   json.RawMessage `json:"errors"`
	Results  int             `json:"results"`
	Paging   struct{ Current, Total int }
	Response json.RawMessage `json:"response"`
}

// apiError turns the "errors" field ([] or {"key": "msg"}) into an error.
func apiError(raw json.RawMessage) error {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "[]" || s == "{}" || s == "null" {
		return nil
	}
	var m map[string]string
	if json.Unmarshal(raw, &m) == nil && len(m) > 0 {
		parts := make([]string, 0, len(m))
		for k, v := range m {
			parts = append(parts, k+": "+v)
		}
		return fmt.Errorf("api-hockey: %s", strings.Join(parts, "; "))
	}
	return fmt.Errorf("api-hockey: %s", s)
}

func (c *Client) get(ctx context.Context, path string, q url.Values, ttl time.Duration, out any) (envelope, error) {
	u := c.Base + path + "?" + q.Encode()
	body, cached, err := c.HTTP.Get(ctx, httpcache.Request{URL: u, Key: "apihockey" + path + "?" + q.Encode(), TTL: ttl})
	if err != nil {
		return envelope{}, err
	}
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return env, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := apiError(env.Errors); err != nil {
		return env, err
	}
	if err := json.Unmarshal(env.Response, out); err != nil {
		return env, fmt.Errorf("decode %s response: %w", path, err)
	}
	slog.Info("api-hockey", "path", path, "q", q.Encode(), "results", env.Results, "cached", cached, "remaining", c.Remaining)
	return env, nil
}

// League as returned by /leagues.
type League struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Country struct {
		Name string `json:"name"`
		Code string `json:"code"`
	} `json:"country"`
	Seasons []struct {
		Season  int    `json:"season"`
		Current bool   `json:"current"`
		Start   string `json:"start"`
		End     string `json:"end"`
	} `json:"seasons"`
}

// Leagues searches leagues by name fragment; an empty search lists every
// league in one request. Cached for a week.
func (c *Client) Leagues(ctx context.Context, search string) ([]League, error) {
	var out []League
	q := url.Values{}
	if search != "" {
		q.Set("search", search)
	}
	_, err := c.get(ctx, "/leagues", q, 7*24*time.Hour, &out)
	return out, err
}

// Game as returned by /games.
type Game struct {
	ID        int     `json:"id"`
	Date      string  `json:"date"`
	Timestamp int64   `json:"timestamp"`
	Week      *string `json:"week"`
	Status    struct {
		Long  string `json:"long"`
		Short string `json:"short"`
	} `json:"status"`
	League struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Season int    `json:"season"`
	} `json:"league"`
	Teams struct {
		Home Team `json:"home"`
		Away Team `json:"away"`
	} `json:"teams"`
	Scores struct {
		Home *int `json:"home"`
		Away *int `json:"away"`
	} `json:"scores"`
	Periods struct {
		First     *string `json:"first"`
		Second    *string `json:"second"`
		Third     *string `json:"third"`
		Overtime  *string `json:"overtime"`
		Penalties *string `json:"penalties"`
	} `json:"periods"`
}

// Team reference inside a game.
type Team struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Games lists one league season. Finished seasons are cached forever; the current
// season for ttl.
func (c *Client) Games(ctx context.Context, league, season int, ttl time.Duration) ([]Game, error) {
	var out []Game
	_, err := c.get(ctx, "/games", url.Values{"league": {strconv.Itoa(league)}, "season": {strconv.Itoa(season)}}, ttl, &out)
	return out, err
}

// Status maps API status codes onto games.status.
func Status(short string) string {
	switch short {
	case "NS", "TBD":
		return "scheduled"
	case "P1", "P2", "P3", "OT", "PT", "BT", "LIVE", "INTR":
		return "live"
	case "FT", "AOT", "AP":
		return "finished"
	case "POST":
		return "postponed"
	case "CANC", "ABD":
		return "cancelled"
	default:
		return "other"
	}
}

// Result derives the normalised result of a finished game.
func (g Game) Result() (hockey.Result, error) {
	ps := []*string{g.Periods.First, g.Periods.Second, g.Periods.Third}
	periods := make([]hockey.Score, 0, 3)
	for i, p := range ps {
		if p == nil {
			return hockey.Result{}, fmt.Errorf("game %d: period %d missing", g.ID, i+1)
		}
		s, err := hockey.ParseScore(*p)
		if err != nil {
			return hockey.Result{}, fmt.Errorf("game %d: %w", g.ID, err)
		}
		periods = append(periods, s)
	}
	opt := func(p *string) (*hockey.Score, error) {
		if p == nil || strings.TrimSpace(*p) == "" {
			return nil, nil
		}
		s, err := hockey.ParseScore(*p)
		if err != nil {
			return nil, fmt.Errorf("game %d: %w", g.ID, err)
		}
		return &s, nil
	}
	ot, err := opt(g.Periods.Overtime)
	if err != nil {
		return hockey.Result{}, err
	}
	so, err := opt(g.Periods.Penalties)
	if err != nil {
		return hockey.Result{}, err
	}
	if g.Status.Short != "AP" {
		so = nil // some feeds send "0-0" penalties for every game
	}
	if g.Status.Short == "FT" {
		ot = nil
	}
	return hockey.BuildResult(periods, ot, so)
}

// StartAt parses the game time.
func (g Game) StartAt() (time.Time, error) {
	if g.Timestamp > 0 {
		return time.Unix(g.Timestamp, 0).UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, g.Date)
	if err != nil {
		return t, fmt.Errorf("game %d date %q: %w", g.ID, g.Date, err)
	}
	return t.UTC(), nil
}

// OddsItem as returned by /odds.
type OddsItem struct {
	Game struct {
		ID int `json:"id"`
	} `json:"game"`
	Update     string `json:"update"`
	Bookmakers []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Bets []struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Values []struct {
				Value string          `json:"value"`
				Odd   json.RawMessage `json:"odd"`
			} `json:"values"`
		} `json:"bets"`
	} `json:"bookmakers"`
}

// Odds lists pre-match odds for one league season, one page at a time.
// The API only keeps odds of upcoming and recent games, so history must be
// collected by snapshotting; never cache (TTL -1).
func (c *Client) Odds(ctx context.Context, league, season, page int) ([]OddsItem, int, error) {
	var out []OddsItem
	q := url.Values{"league": {strconv.Itoa(league)}, "season": {strconv.Itoa(season)}}
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	env, err := c.get(ctx, "/odds", q, -1, &out)
	return out, env.Paging.Total, err
}
