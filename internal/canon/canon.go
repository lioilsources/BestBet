// Package canon defines the source-independent structures exchanged between
// ingest, the model, the backtest and the API (plan 01 §3, docs/DATA_CONTRACT.md).
// The model packages depend on canon only, never on the database or HTTP.
package canon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// Side of a game.
type Side string

const (
	Home Side = "home"
	Away Side = "away"
)

// Score is one home-away pair.
type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

// Match is a game as the model sees it. Result fields are nil before the game.
type Match struct {
	ID       int64     `json:"id"`
	League   string    `json:"league"`
	Season   int       `json:"season"`
	StartAt  time.Time `json:"start_at"`
	HomeTeam int32     `json:"home_team"`
	AwayTeam int32     `json:"away_team"`
	Phase    string    `json:"phase"`

	Periods   []Score `json:"periods,omitempty"`    // regulation periods 1..3
	Reg       *Score  `json:"reg,omitempty"`        // 60 min
	Final     *Score  `json:"final,omitempty"`      // incl. OT and shootout winner goal
	DecidedIn string  `json:"decided_in,omitempty"` // reg | ot | so

	Goalies []GoalieStart `json:"goalies,omitempty"`
	Lineups []Lineup      `json:"lineups,omitempty"`
}

// Played reports whether a regulation result is known.
func (m Match) Played() bool { return m.Reg != nil && len(m.Periods) == 3 }

// GoalieStart is a goalie appearance in a game.
type GoalieStart struct {
	Side         Side   `json:"side"`
	Player       string `json:"player"` // canonical name until players have ids everywhere
	PlayerID     int32  `json:"player_id,omitempty"`
	Starter      bool   `json:"starter"`
	ShotsAgainst int    `json:"shots_against"`
	Saves        int    `json:"saves"`
	TOISeconds   int    `json:"toi_seconds,omitempty"`
	Confirm      string `json:"confirm"` // UNCONFIRMED | EXPECTED | CONFIRMED
}

// Lineup is one team's players in a game.
type Lineup struct {
	Side    Side            `json:"side"`
	Players []PlayerMinutes `json:"players"`
	Absent  []int32         `json:"absent,omitempty"` // player ids known to miss the game
}

// PlayerMinutes is a player's usage in a game.
type PlayerMinutes struct {
	PlayerID   int32  `json:"player_id"`
	Position   string `json:"position"` // C LW RW F D G
	LineNo     int    `json:"line_no,omitempty"`
	TOISeconds int    `json:"toi_seconds,omitempty"`
}

// OddsQuote is one price at one time.
type OddsQuote struct {
	Book       string    `json:"book"`
	Market     string    `json:"market"`
	Selection  string    `json:"selection"`
	Line       *float64  `json:"line,omitempty"`
	Price      float64   `json:"price"`
	CapturedAt time.Time `json:"captured_at"`
	Kind       string    `json:"kind"` // t24h | t1h | closing | historical | adhoc
}

// ScoreMatrix holds P(home=h, away=a) for h, a in [0, len).
type ScoreMatrix [][]float64

// Sum is the total mass (should be ~1; truncation loses the tail).
func (s ScoreMatrix) Sum() float64 {
	t := 0.0
	for _, row := range s {
		for _, p := range row {
			t += p
		}
	}
	return t
}

// MarketProbs maps market -> selection key -> probability. Selection keys with a
// line are "over@5.5" or "home@-1.5"; correct score is "3-2".
type MarketProbs map[string]map[string]float64

// Contribution is one input's share of the home-win probability in percentage
// points versus the input at its league-average value (07 §4).
type Contribution struct {
	Input string  `json:"input"` // e.g. "goalie", "wave", "strength"
	Side  Side    `json:"side,omitempty"`
	PP    float64 `json:"pp"`
}

// Prediction is the model output for one game.
type Prediction struct {
	MatchID       int64          `json:"match_id"`
	ModelVersion  string         `json:"model_version"`
	InputsHash    string         `json:"inputs_hash"`
	ComputedAt    time.Time      `json:"computed_at"`
	LambdaHome    float64        `json:"lambda_home"`
	LambdaAway    float64        `json:"lambda_away"`
	Matrix        ScoreMatrix    `json:"matrix"`
	P1Matrix      ScoreMatrix    `json:"p1_matrix,omitempty"`
	OTHomeWin     float64        `json:"ot_home_win"` // P(home wins OT/SO | tied after 60)
	Markets       MarketProbs    `json:"markets"`
	Contributions []Contribution `json:"contributions,omitempty"`
}

// HashInputs is a stable hash of anything JSON-serialisable (map keys are
// sorted by encoding/json). It identifies the input set of a prediction.
func HashInputs(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("hash inputs: %w", err)
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:12]), nil
}

// SortMatches orders by start time, then id (deterministic walk-forward).
func SortMatches(ms []Match) {
	sort.SliceStable(ms, func(i, j int) bool {
		if !ms[i].StartAt.Equal(ms[j].StartAt) {
			return ms[i].StartAt.Before(ms[j].StartAt)
		}
		return ms[i].ID < ms[j].ID
	})
}
