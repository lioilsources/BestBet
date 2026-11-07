// Package oddspapi is a client for OddsPapi v4 (api.oddspapi.io): tournaments,
// fixtures, market catalogue and historical odds. Ice hockey is sportId 15.
//
// Only parts of the response schema are documented publicly; fields not in the
// docs are read with fallbacks (see pick*). Raw responses stay in the on-disk
// cache, so a parser fix never costs requests. Historical odds start 2026-01.
package oddspapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/odds"
)

// DefaultBase is the public endpoint.
const DefaultBase = "https://api.oddspapi.io/v4"

// SportHockey is OddsPapi's ice hockey id.
const SportHockey = 15

// Client calls OddsPapi. The key travels as the apiKey query parameter and is
// stripped from cache keys.
type Client struct {
	Base string
	Key  string
	HTTP *httpcache.Client
}

func (c *Client) get(ctx context.Context, path string, q url.Values, ttl time.Duration) ([]byte, error) {
	key := "oddspapi" + path + "?" + q.Encode()
	q2 := url.Values{}
	for k, v := range q {
		q2[k] = v
	}
	q2.Set("apiKey", c.Key)
	req := httpcache.Request{URL: c.Base + path + "?" + q2.Encode(), Key: key, TTL: ttl}
	body, cached, err := c.HTTP.Get(ctx, req)
	if err != nil && strings.Contains(err.Error(), "http 429") {
		// Cooldown ("Please wait 3.95 seconds"); one retry after it.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryAfter(err.Error())):
		}
		body, cached, err = c.HTTP.Get(ctx, req)
	}
	if err != nil {
		return nil, fmt.Errorf("oddspapi %s: %w", path, err)
	}
	slog.Info("oddspapi", "path", path, "q", q.Encode(), "cached", cached, "bytes", len(body))
	return body, nil
}

// Tournament from /tournaments.
type Tournament struct {
	TournamentID   int    `json:"tournamentId"`
	TournamentSlug string `json:"tournamentSlug"`
	TournamentName string `json:"tournamentName"`
	CategoryName   string `json:"categoryName"`
}

// Tournaments lists hockey competitions (cached for a week).
func (c *Client) Tournaments(ctx context.Context) ([]Tournament, error) {
	b, err := c.get(ctx, "/tournaments", url.Values{"sportId": {strconv.Itoa(SportHockey)}}, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}
	var out []Tournament
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("decode tournaments: %w", err)
	}
	return out, nil
}

// Fixture is the subset of a fixture we use.
type Fixture struct {
	FixtureID string
	Home      string
	Away      string
	StartTime time.Time
	StatusID  int
}

// Fixtures lists fixtures of a tournament in [from, to].
func (c *Client) Fixtures(ctx context.Context, tournamentID int, from, to time.Time, ttl time.Duration) ([]Fixture, error) {
	q := url.Values{
		"tournamentId": {strconv.Itoa(tournamentID)},
		"from":         {from.UTC().Format("2006-01-02")},
		"to":           {to.UTC().Format("2006-01-02")},
	}
	b, err := c.get(ctx, "/fixtures", q, ttl)
	if err != nil {
		return nil, err
	}
	return ParseFixtures(b)
}

// ParseFixtures decodes a fixtures array with field-name fallbacks.
func ParseFixtures(b []byte) ([]Fixture, error) {
	var raw []map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("decode fixtures: %w", err)
	}
	out := make([]Fixture, 0, len(raw))
	for _, m := range raw {
		f := Fixture{
			FixtureID: pickString(m, "fixtureId", "id"),
			Home:      pickString(m, "participant1Name", "homeName", "home", "participant1"),
			Away:      pickString(m, "participant2Name", "awayName", "away", "participant2"),
		}
		f.StatusID, _ = strconv.Atoi(pickString(m, "statusId"))
		if s := pickString(m, "startTime", "trueStartTime"); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return nil, fmt.Errorf("fixture %s startTime %q: %w", f.FixtureID, s, err)
			}
			f.StartTime = t.UTC()
		}
		if f.FixtureID == "" {
			return nil, fmt.Errorf("fixture without fixtureId: %v", m)
		}
		out = append(out, f)
	}
	return out, nil
}

// MarketDef describes one market id from the catalogue.
type MarketDef struct {
	Name     string
	Handicap *float64
	Period   string
	Outcomes map[string]string // outcomeId -> name ("1", "X", "2", "Over", ...)
}

// Markets loads the hockey market catalogue (cached for a week).
func (c *Client) Markets(ctx context.Context) (map[string]MarketDef, error) {
	b, err := c.get(ctx, "/markets", url.Values{"sportId": {strconv.Itoa(SportHockey)}}, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}
	return ParseMarkets(b)
}

// ParseMarkets decodes the catalogue with field-name fallbacks.
func ParseMarkets(b []byte) (map[string]MarketDef, error) {
	var raw []map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("decode markets: %w", err)
	}
	out := map[string]MarketDef{}
	for _, m := range raw {
		id := pickString(m, "marketId", "id")
		d := MarketDef{
			Name:     pickString(m, "marketName", "name", "marketType"),
			Period:   pickString(m, "period"),
			Outcomes: map[string]string{},
		}
		if h := pickString(m, "handicap", "line"); h != "" {
			if v, err := strconv.ParseFloat(h, 64); err == nil {
				d.Handicap = &v
			}
		}
		if outs, ok := m["outcomes"].([]any); ok {
			for _, o := range outs {
				if om, ok := o.(map[string]any); ok {
					d.Outcomes[pickString(om, "outcomeId", "id")] = pickString(om, "outcomeName", "name")
				}
			}
		}
		out[id] = d
	}
	return out, nil
}

// HistoryPoint is one historical price.
type HistoryPoint struct {
	Book      string
	MarketID  string
	OutcomeID string
	CreatedAt time.Time
	Price     float64
}

// HistoricalOdds returns every price change of a fixture for up to 3 books.
// Finished fixtures never change, so they are cached forever.
func (c *Client) HistoricalOdds(ctx context.Context, fixtureID string, books []string, finished bool) ([]HistoryPoint, error) {
	if len(books) == 0 || len(books) > 3 {
		return nil, fmt.Errorf("historical-odds takes 1-3 bookmakers, got %d", len(books))
	}
	ttl := time.Duration(-1)
	if finished {
		ttl = 0
	}
	b, err := c.get(ctx, "/historical-odds", url.Values{"fixtureId": {fixtureID}, "bookmakers": {strings.Join(books, ",")}}, ttl)
	if err != nil {
		return nil, err
	}
	return ParseHistorical(b)
}

// ParseHistorical flattens bookmakers.<book>.markets.<id>.outcomes.<id>.players.<id>[].
func ParseHistorical(b []byte) ([]HistoryPoint, error) {
	var doc struct {
		Bookmakers map[string]struct {
			Markets map[string]struct {
				Outcomes map[string]struct {
					Players map[string][]struct {
						CreatedAt string          `json:"createdAt"`
						Price     json.RawMessage `json:"price"`
					} `json:"players"`
				} `json:"outcomes"`
			} `json:"markets"`
		} `json:"bookmakers"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("decode historical odds: %w", err)
	}
	var out []HistoryPoint
	for book, bm := range doc.Bookmakers {
		for mid, mk := range bm.Markets {
			for oid, oc := range mk.Outcomes {
				for _, entries := range oc.Players {
					for _, e := range entries {
						p, err := odds.ParsePrice(e.Price)
						if err != nil {
							continue // suspended lines come as 0 or 1
						}
						t, err := time.Parse(time.RFC3339, e.CreatedAt)
						if err != nil {
							return nil, fmt.Errorf("createdAt %q: %w", e.CreatedAt, err)
						}
						out = append(out, HistoryPoint{Book: book, MarketID: mid, OutcomeID: oid, CreatedAt: t.UTC(), Price: p})
					}
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.Before(b.CreatedAt)
		}
		return a.Book+a.MarketID+a.OutcomeID < b.Book+b.MarketID+b.OutcomeID
	})
	return out, nil
}

// Normalize maps a catalogue market + outcome onto a Quote using the same
// vocabulary as API-HOCKEY. Handicap/total lines come from the market definition.
func Normalize(def MarketDef, outcomeID string, price float64) odds.Quote {
	name := def.Name
	if def.Period != "" && !strings.Contains(strings.ToLower(name), "period") {
		name += " " + def.Period
	}
	val := def.Outcomes[outcomeID]
	switch strings.ToLower(val) {
	case "1":
		val = "Home"
	case "x":
		val = "Draw"
	case "2":
		val = "Away"
	}
	if def.Handicap != nil && *def.Handicap != 0 { // 0 = no line (1X2, winner)
		val = fmt.Sprintf("%s %s", val, strconv.FormatFloat(*def.Handicap, 'f', -1, 64))
		if strings.HasPrefix(strings.ToLower(val), "home") || strings.HasPrefix(strings.ToLower(val), "away") {
			if *def.Handicap > 0 {
				val = strings.Replace(val, " ", " +", 1)
			}
		}
	}
	q := odds.NormalizeAPIHockey(name, val, price)
	q.RawMarket = name
	return q
}

func pickString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		switch v := m[k].(type) {
		case string:
			if v != "" {
				return v
			}
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case map[string]any:
			if s := pickString(v, "name", "participantName"); s != "" {
				return s
			}
		}
	}
	return ""
}

var waitRE = regexp.MustCompile(`wait ([0-9.]+) seconds`)

// retryAfter reads the cooldown from a 429 body, plus a second of slack.
func retryAfter(msg string) time.Duration {
	d := 5 * time.Second
	if m := waitRE.FindStringSubmatch(msg); m != nil {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			d = time.Duration(v*float64(time.Second)) + time.Second
		}
	}
	return d
}
