// Package nhl parses the public NHL API (api-web.nhle.com, no key): the daily
// score feed (every goal with its period, so period, OT and shootout results
// come from one request per day) and the boxscore (goalies with shots and
// saves). The NHL is the efficient control league of plan 00 §1.4.5.
package nhl

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/hockey"
)

// Base is the API root.
const Base = "https://api-web.nhle.com/v1"

// Team in a game.
type Team struct {
	ID     int
	Abbrev string
	Name   string
	Score  *int
	SOG    *int
}

// Goal is one goal with the score after it.
type Goal struct {
	Period     int
	PeriodType string // REG | OT | SO
	Home, Away int    // cumulative
}

// Game from the score feed.
type Game struct {
	ID       int64
	Season   int // start year (20252026 → 2025)
	GameType int // 1 preseason, 2 regular, 3 play-off, 4 all-star
	StartAt  time.Time
	State    string // FUT PRE LIVE CRIT FINAL OFF
	Schedule string // OK PPD CNCL
	Home     Team
	Away     Team
	Goals    []Goal
	LastType string // gameOutcome.lastPeriodType
}

type rawTeam struct {
	ID     int    `json:"id"`
	Abbrev string `json:"abbrev"`
	Name   struct {
		Default string `json:"default"`
	} `json:"name"`
	CommonName struct {
		Default string `json:"default"`
	} `json:"commonName"`
	PlaceName struct {
		Default string `json:"default"`
	} `json:"placeName"`
	Score *int `json:"score"`
	SOG   *int `json:"sog"`
}

func (r rawTeam) team() Team {
	name := r.Name.Default
	if r.PlaceName.Default != "" && r.CommonName.Default != "" {
		name = r.PlaceName.Default + " " + r.CommonName.Default
	}
	return Team{ID: r.ID, Abbrev: r.Abbrev, Name: name, Score: r.Score, SOG: r.SOG}
}

// ParseScore decodes /v1/score/{date}.
func ParseScore(b []byte) ([]Game, error) {
	var doc struct {
		Games []struct {
			ID                int64   `json:"id"`
			Season            int     `json:"season"`
			GameType          int     `json:"gameType"`
			StartTimeUTC      string  `json:"startTimeUTC"`
			GameState         string  `json:"gameState"`
			GameScheduleState string  `json:"gameScheduleState"`
			HomeTeam          rawTeam `json:"homeTeam"`
			AwayTeam          rawTeam `json:"awayTeam"`
			GameOutcome       *struct {
				LastPeriodType string `json:"lastPeriodType"`
			} `json:"gameOutcome"`
			Goals []struct {
				Period           int `json:"period"`
				PeriodDescriptor struct {
					PeriodType string `json:"periodType"`
				} `json:"periodDescriptor"`
				HomeScore int `json:"homeScore"`
				AwayScore int `json:"awayScore"`
			} `json:"goals"`
		} `json:"games"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("decode nhl score: %w", err)
	}
	out := make([]Game, 0, len(doc.Games))
	for _, r := range doc.Games {
		t, err := time.Parse(time.RFC3339, r.StartTimeUTC)
		if err != nil {
			return nil, fmt.Errorf("nhl game %d start %q: %w", r.ID, r.StartTimeUTC, err)
		}
		g := Game{ID: r.ID, Season: r.Season / 10000, GameType: r.GameType, StartAt: t.UTC(), State: r.GameState,
			Schedule: r.GameScheduleState, Home: r.HomeTeam.team(), Away: r.AwayTeam.team()}
		if r.GameOutcome != nil {
			g.LastType = r.GameOutcome.LastPeriodType
		}
		for _, x := range r.Goals {
			g.Goals = append(g.Goals, Goal{Period: x.Period, PeriodType: x.PeriodDescriptor.PeriodType, Home: x.HomeScore, Away: x.AwayScore})
		}
		out = append(out, g)
	}
	return out, nil
}

// Status maps the NHL state onto games.status.
func (g Game) Status() string {
	switch {
	case g.Schedule == "PPD":
		return "postponed"
	case g.Schedule == "CNCL":
		return "cancelled"
	case g.State == "OFF" || g.State == "FINAL":
		return "finished"
	case g.State == "LIVE" || g.State == "CRIT":
		return "live"
	case g.State == "FUT" || g.State == "PRE":
		return "scheduled"
	}
	return "other"
}

// Phase maps the game type; preseason and all-star games are not stored.
func (g Game) Phase() (string, bool) {
	switch g.GameType {
	case 2:
		return hockey.PhaseRegular, true
	case 3:
		return "playoff", true
	}
	return "", false
}

// Result rebuilds periods, OT and the shootout from the goal list and checks
// it against the final score.
func (g Game) Result() (hockey.Result, error) {
	periods := make([]hockey.Score, 3)
	var ot *hockey.Score
	var so *hockey.Score
	ph, pa := 0, 0
	for _, x := range g.Goals {
		dh, da := x.Home-ph, x.Away-pa
		ph, pa = x.Home, x.Away
		if dh+da != 1 || dh < 0 || da < 0 {
			return hockey.Result{}, fmt.Errorf("nhl game %d: goal list not cumulative at %d-%d", g.ID, x.Home, x.Away)
		}
		switch {
		case x.PeriodType == "REG" && x.Period >= 1 && x.Period <= 3:
			periods[x.Period-1].Home += dh
			periods[x.Period-1].Away += da
		case x.PeriodType == "OT":
			if ot == nil {
				ot = &hockey.Score{}
			}
			ot.Home += dh
			ot.Away += da
		case x.PeriodType == "SO":
			so = &hockey.Score{Home: dh, Away: da} // the decisive shootout "goal"
		default:
			return hockey.Result{}, fmt.Errorf("nhl game %d: goal in period %d %s", g.ID, x.Period, x.PeriodType)
		}
	}
	if so != nil && ot == nil {
		ot = &hockey.Score{}
	}
	r, err := hockey.BuildResult(periods, ot, so)
	if err != nil {
		return r, fmt.Errorf("nhl game %d: %w", g.ID, err)
	}
	if g.Home.Score != nil && g.Away.Score != nil && (r.Final.Home != *g.Home.Score || r.Final.Away != *g.Away.Score) {
		return r, fmt.Errorf("nhl game %d: goals give %d-%d, score is %d-%d", g.ID, r.Final.Home, r.Final.Away, *g.Home.Score, *g.Away.Score)
	}
	return r, nil
}

// Goalie line from the boxscore.
type Goalie struct {
	Side         string
	PlayerID     int
	Name         string
	Starter      bool
	ShotsAgainst int
	Saves        int
	GoalsAgainst int
	TOISeconds   int
}

// ParseBoxscore returns the goalies who played or started.
func ParseBoxscore(b []byte) ([]Goalie, error) {
	type rg struct {
		PlayerID int `json:"playerId"`
		Name     struct {
			Default string `json:"default"`
		} `json:"name"`
		Starter      bool   `json:"starter"`
		ShotsAgainst int    `json:"shotsAgainst"`
		Saves        int    `json:"saves"`
		GoalsAgainst int    `json:"goalsAgainst"`
		TOI          string `json:"toi"`
	}
	var doc struct {
		PlayerByGameStats struct {
			HomeTeam struct {
				Goalies []rg `json:"goalies"`
			} `json:"homeTeam"`
			AwayTeam struct {
				Goalies []rg `json:"goalies"`
			} `json:"awayTeam"`
		} `json:"playerByGameStats"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("decode nhl boxscore: %w", err)
	}
	var out []Goalie
	for side, gs := range map[string][]rg{"home": doc.PlayerByGameStats.HomeTeam.Goalies, "away": doc.PlayerByGameStats.AwayTeam.Goalies} {
		for _, x := range gs {
			toi := mmss(x.TOI)
			if toi == 0 && !x.Starter {
				continue
			}
			out = append(out, Goalie{Side: side, PlayerID: x.PlayerID, Name: x.Name.Default, Starter: x.Starter,
				ShotsAgainst: x.ShotsAgainst, Saves: x.Saves, GoalsAgainst: x.GoalsAgainst, TOISeconds: toi})
		}
	}
	return out, nil
}

func mmss(s string) int {
	m, sec, ok := strings.Cut(s, ":")
	if !ok {
		return 0
	}
	a, err1 := strconv.Atoi(m)
	b, err2 := strconv.Atoi(sec)
	if err1 != nil || err2 != nil {
		return 0
	}
	return a*60 + b
}

// HeadCoaches holds the head coach of each side, from gamecenter/<id>/right-rail.
type HeadCoaches struct {
	Home, Away string
}

// ParseRightRail returns the head coaches; empty names when the feed has none.
func ParseRightRail(b []byte) (HeadCoaches, error) {
	type coach struct {
		HeadCoach struct {
			Default string `json:"default"`
		} `json:"headCoach"`
	}
	var doc struct {
		GameInfo struct {
			HomeTeam coach `json:"homeTeam"`
			AwayTeam coach `json:"awayTeam"`
		} `json:"gameInfo"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return HeadCoaches{}, fmt.Errorf("decode nhl right-rail: %w", err)
	}
	return HeadCoaches{Home: strings.TrimSpace(doc.GameInfo.HomeTeam.HeadCoach.Default),
		Away: strings.TrimSpace(doc.GameInfo.AwayTeam.HeadCoach.Default)}, nil
}
