// Package asa reads the public American Soccer Analysis API
// (app.americansocceranalysis.com/api/v1, no key): games with referee,
// managers and stadium, teams, and every shot with minute, shooter, xG,
// pattern of play and the score before the shot. Plan 05 S0 verified it for
// MLS 2013–2026; halves come from the score fields of the shots.
package asa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/soccer"
)

// Base is the API root.
const Base = "https://app.americansocceranalysis.com/api/v1"

// ShotBatch is how many games one shots request may name (verified in S0).
const ShotBatch = 30

// Team is one club.
type Team struct {
	ID   string `json:"team_id"`
	Name string `json:"team_name"`
}

// Game is one match.
type Game struct {
	ID          string    `json:"game_id"`
	StartAt     time.Time `json:"-"`
	RawStart    string    `json:"date_time_utc"`
	HomeTeam    string    `json:"home_team_id"`
	AwayTeam    string    `json:"away_team_id"`
	HomeScore   int       `json:"home_score"`
	AwayScore   int       `json:"away_score"`
	Referee     string    `json:"referee_id"`
	Stadium     string    `json:"stadium_id"`
	HomeManager string    `json:"home_manager_id"`
	AwayManager string    `json:"away_manager_id"`
	Minutes     int       `json:"expanded_minutes"`
	Season      string    `json:"season_name"`
	Matchday    int       `json:"matchday"`
	Knockout    bool      `json:"knockout_game"`
	Status      string    `json:"status"` // FullTime, ...
}

// Finished reports whether the game is over.
func (g Game) Finished() bool { return g.Status == "FullTime" }

// Shot is one attempt. HomeScore/AwayScore are the score before the shot.
type Shot struct {
	GameID         string  `json:"game_id"`
	Period         int     `json:"period_id"` // 1, 2 halves; 3, 4 extra time
	ExpandedMinute int     `json:"expanded_minute"`
	GameMinute     int     `json:"game_minute"`
	TeamID         string  `json:"team_id"`
	ShooterID      string  `json:"shooter_player_id"`
	ShooterName    string  `json:"shooter_player_name"`
	Goal           int     `json:"goal"`
	OwnGoal        int     `json:"own_goal"`
	HomeScore      int     `json:"home_score"`
	AwayScore      int     `json:"away_score"`
	XG             float64 `json:"shot_xg"`
	PSXG           float64 `json:"shot_psxg"`
	Pattern        string  `json:"pattern_of_play"` // Regular, Corner, Fastbreak, Set piece, Free kick, Penalty
	Order          int     `json:"shot_order"`
}

func decodeList[T any](b []byte, what string) ([]T, error) {
	var out []T
	if err := json.Unmarshal(b, &out); err != nil {
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(b, &e) == nil && e.Error != "" {
			return nil, fmt.Errorf("asa %s: %s", what, e.Error)
		}
		return nil, fmt.Errorf("asa %s: %w", what, err)
	}
	return out, nil
}

// ParseTeams decodes /{league}/teams.
func ParseTeams(b []byte) ([]Team, error) { return decodeList[Team](b, "teams") }

// ParseGames decodes /{league}/games.
func ParseGames(b []byte) ([]Game, error) {
	gs, err := decodeList[Game](b, "games")
	if err != nil {
		return nil, err
	}
	for i := range gs {
		t, err := time.Parse("2006-01-02 15:04:05 UTC", gs[i].RawStart)
		if err != nil {
			return nil, fmt.Errorf("asa game %s start %q: %w", gs[i].ID, gs[i].RawStart, err)
		}
		gs[i].StartAt = t
	}
	return gs, nil
}

// ParseShots decodes /{league}/games/shots.
func ParseShots(b []byte) ([]Shot, error) { return decodeList[Shot](b, "shots") }

// SeasonYear is the numeric season key (plan 05 §2: the year the season ends).
func (g Game) SeasonYear() (int, error) { return strconv.Atoi(g.Season) }

// Client fetches through the HTTP cache.
type Client struct {
	HTTP   *httpcache.Client
	League string // API path, e.g. "mls"
}

func (c *Client) get(ctx context.Context, path string, q url.Values, ttl time.Duration) ([]byte, error) {
	u := Base + "/" + c.League + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	b, _, err := c.HTTP.Get(ctx, httpcache.Request{URL: u, Key: u, TTL: ttl})
	return b, err
}

// Teams lists the league's clubs.
func (c *Client) Teams(ctx context.Context, ttl time.Duration) ([]Team, error) {
	b, err := c.get(ctx, "/teams", nil, ttl)
	if err != nil {
		return nil, err
	}
	return ParseTeams(b)
}

// Games lists one season.
func (c *Client) Games(ctx context.Context, season int, ttl time.Duration) ([]Game, error) {
	b, err := c.get(ctx, "/games", url.Values{"season_name": {strconv.Itoa(season)}}, ttl)
	if err != nil {
		return nil, err
	}
	return ParseGames(b)
}

// Shots returns the shots of the given games, ShotBatch games per request.
// ttl applies to batches with an unfinished game; finished batches are kept forever.
func (c *Client) Shots(ctx context.Context, games []Game, ttl time.Duration) (map[string][]Shot, error) {
	out := map[string][]Shot{}
	ids := make([]string, 0, len(games))
	done := map[string]bool{}
	for _, g := range games {
		ids = append(ids, g.ID)
		done[g.ID] = g.Finished()
	}
	sort.Strings(ids) // stable batches, stable cache keys
	for i := 0; i < len(ids); i += ShotBatch {
		batch := ids[i:min(i+ShotBatch, len(ids))]
		t := time.Duration(0)
		for _, id := range batch {
			if !done[id] {
				t = ttl
			}
		}
		b, err := c.get(ctx, "/games/shots", url.Values{"game_id": {strings.Join(batch, ",")}}, t)
		if err != nil {
			return nil, err
		}
		ss, err := ParseShots(b)
		if err != nil {
			return nil, err
		}
		for _, s := range ss {
			out[s.GameID] = append(out[s.GameID], s)
		}
	}
	return out, nil
}

// SortShots orders shots by period, minute and the source's shot order.
func SortShots(ss []Shot) {
	sort.SliceStable(ss, func(i, j int) bool {
		a, b := ss[i], ss[j]
		if a.Period != b.Period {
			return a.Period < b.Period
		}
		if a.ExpandedMinute != b.ExpandedMinute {
			return a.ExpandedMinute < b.ExpandedMinute
		}
		return a.Order < b.Order
	})
}

// after is the score after shot s (own goals count for the other side).
func after(s Shot, homeTeam string) soccer.Score {
	sc := soccer.Score{Home: s.HomeScore, Away: s.AwayScore}
	if s.Goal == 0 && s.OwnGoal == 0 {
		return sc
	}
	home := s.TeamID == homeTeam
	if s.OwnGoal != 0 {
		home = !home
	}
	if home {
		sc.Home++
	} else {
		sc.Away++
	}
	return sc
}

// stateAtEnd is the score when period p ended: the score before the first
// shot of a later period, else the score after the last shot up to p.
func stateAtEnd(ss []Shot, p int, homeTeam string) (soccer.Score, bool) {
	var last *Shot
	for i := range ss {
		if ss[i].Period > p {
			return soccer.Score{Home: ss[i].HomeScore, Away: ss[i].AwayScore}, true
		}
		last = &ss[i]
	}
	if last == nil {
		return soccer.Score{}, false
	}
	return after(*last, homeTeam), true
}

// Result builds the game result. The game's own score is authoritative for
// the end of play (shots miss most own goals); halves come from the score
// fields of the shots. ok is false when the halves cannot be reconciled with
// the final score, and the game should then be stored without halves.
//
// A knockout game that is level at the end went to penalties; the source does
// not name the shootout winner, so it stays unknown.
func Result(g Game, shots []Shot) (r soccer.Result, otType string, ok bool, err error) {
	ss := append([]Shot(nil), shots...)
	SortShots(ss)
	final := soccer.Score{Home: g.HomeScore, Away: g.AwayScore}
	extra := false
	for _, s := range ss {
		if s.Period > 2 {
			extra = true
		}
	}
	reg := final
	var et *soccer.Score
	if extra {
		var have bool
		if reg, have = stateAtEnd(ss, 2, g.HomeTeam); !have {
			return r, "", false, nil
		}
		e := soccer.Score{Home: final.Home - reg.Home, Away: final.Away - reg.Away}
		if e.Home < 0 || e.Away < 0 || reg.Home != reg.Away {
			return r, "", false, nil
		}
		et = &e
	}
	h1, have := stateAtEnd(ss, 1, g.HomeTeam)
	if !have {
		if reg.Home != 0 || reg.Away != 0 {
			return r, "", false, nil
		}
		h1 = soccer.Score{} // goalless and shotless first half
	}
	h2 := soccer.Score{Home: reg.Home - h1.Home, Away: reg.Away - h1.Away}
	if h2.Home < 0 || h2.Away < 0 {
		return r, "", false, nil
	}
	shootout := g.Knockout && final.Home == final.Away
	r, err = soccer.BuildResult(h1, h2, et, shootout, "")
	if err != nil {
		return r, "", false, err
	}
	otType = soccer.OTNone
	if et != nil {
		otType = soccer.OTExtra
	}
	return r, otType, true, nil
}
