// Package lineup turns a team's earlier games and a lineup (estimated or
// published) into what the app shows: missing regulars, returns and the
// share of ice time that is missing (plan 04 L2/L3). Pure logic.
package lineup

import (
	"time"

	"github.com/lioilsources/BestBet/internal/features"
)

// Status of a team's lineup for an upcoming game.
const (
	Estimate  = "estimate"  // the team's last game
	Confirmed = "confirmed" // published by the source before the game
	Unknown   = "unknown"   // too few earlier games
)

// Info is display data of one player (season so far).
type Info struct {
	Name   string
	Games  int
	Points int
}

// Player is one missing or returning regular.
type Player struct {
	PlayerID      int32   `json:"player_id"`
	Name          string  `json:"name"`
	Position      string  `json:"position"`
	TOIPerGame    float64 `json:"toi_min"` // minutes
	PointsPerGame float64 `json:"points_per_game"`
	Missed        int     `json:"missed"`
	Key           bool    `json:"key"`
}

// Team is one side's lineup view.
type Team struct {
	Status      string     `json:"status"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	Window      int        `json:"window"`
	Regulars    int        `json:"regulars"`
	Missing     []Player   `json:"missing"`
	Returning   []Player   `json:"returning"`
	Share       float64    `json:"share"` // missing regulars' TOI / all regulars' TOI
	KeyMissing  int        `json:"key_missing"`
}

// Game holds both sides.
type Game struct {
	Home Team `json:"home"`
	Away Team `json:"away"`
}

// Build computes one side. confirmed is nil when no lineup was published;
// then the last game's lineup stands in (estimate).
func Build(prev []features.TeamGame, info map[int32]Info, season int, confirmed []int32, confirmedAt time.Time) Team {
	t := Team{Status: Unknown, Missing: []Player{}, Returning: []Player{}}
	var ids []int32
	switch {
	case confirmed != nil:
		ids, t.Status = confirmed, Confirmed
		ca := confirmedAt
		t.ConfirmedAt = &ca
	case len(prev) > 0:
		ids, t.Status = prev[len(prev)-1].Lineup(), Estimate
	}
	a := features.AbsenceBefore(prev, season, ids)
	if !a.Known {
		t.Status = Unknown
		return t
	}
	t.Window, t.Regulars, t.Share = a.Window, a.Regulars, a.Share
	conv := func(p features.PlayerAbsence) Player {
		in := info[p.PlayerID]
		x := Player{PlayerID: p.PlayerID, Name: in.Name, Position: p.Position, TOIPerGame: p.TOIPerGame / 60, Missed: p.Missed, Key: p.Key}
		if in.Games > 0 {
			x.PointsPerGame = float64(in.Points) / float64(in.Games)
		}
		return x
	}
	for _, p := range a.Missing {
		t.Missing = append(t.Missing, conv(p))
		if p.Key {
			t.KeyMissing++
		}
	}
	for _, p := range a.Returning {
		t.Returning = append(t.Returning, conv(p))
	}
	return t
}
