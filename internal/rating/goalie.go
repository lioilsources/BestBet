// Package rating holds slow-moving strength estimates (plan 00 §1.4.2):
// goalie ratings (save percentage shrunk to the league mean by shots faced)
// now; a Kalman team rating with velocity follows. Pure functions over canon.
package rating

import (
	"fmt"
	"math"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

// GoalieConfig controls the shrinkage.
type GoalieConfig struct {
	PriorShots   float64 // pseudo-shots at league average (shrinkage strength)
	HalfLifeDays float64 // decay of old shots
}

// DefaultGoalieConfig: ~600 shots is where hockey SV% starts to separate from
// noise; one season of half-life keeps last year's form relevant.
func DefaultGoalieConfig() GoalieConfig { return GoalieConfig{PriorShots: 600, HalfLifeDays: 365} }

// GoalieState is the pre-game view of one side's goaltending.
type GoalieState struct {
	Starter     string
	StarterSV   float64 // shrunk save % of the starter before the game
	StarterShot float64 // decayed shots behind the estimate (sample size)
	TeamSV      float64 // shots-weighted shrunk SV% of the team's goalies before the game
	LeagueSV    float64
	Known       bool // a starter is listed for the game
}

// Adj is the log multiplier for the opponent's expected goals: how much more
// (> 0) or less (< 0) this starter concedes than the team's usual goaltending.
func (g GoalieState) Adj() float64 {
	if !g.Known {
		return 0
	}
	return math.Log((1 - g.StarterSV) / (1 - g.TeamSV))
}

type acc struct {
	saves, shots float64
	at           time.Time
}

func (a *acc) decay(to time.Time, halfLife float64) {
	if a.at.IsZero() {
		a.at = to
		return
	}
	d := to.Sub(a.at).Hours() / 24
	if d > 0 {
		f := math.Pow(0.5, d/halfLife)
		a.saves *= f
		a.shots *= f
		a.at = to
	}
}

// GoalieTimeline walks matches in time order and returns, for every match with
// goalie lines, the state of both sides computed only from earlier games
// (causal: a game's own saves never enter its own rating).
func GoalieTimeline(matches []canon.Match, cfg GoalieConfig) map[int64]map[canon.Side]GoalieState {
	ms := append([]canon.Match(nil), matches...)
	canon.SortMatches(ms)
	goalies := map[string]*acc{} // team|name -> acc
	teams := map[int32]*acc{}
	league := &acc{}
	out := map[int64]map[canon.Side]GoalieState{}
	const fallbackSV = 0.900
	shrunk := func(a *acc, toward float64) float64 {
		return (a.saves + cfg.PriorShots*toward) / (a.shots + cfg.PriorShots)
	}
	for _, m := range ms {
		if len(m.Goalies) == 0 {
			continue
		}
		league.decay(m.StartAt, cfg.HalfLifeDays)
		lsv := fallbackSV
		if league.shots > 1000 {
			lsv = league.saves / league.shots
		}
		st := map[canon.Side]GoalieState{}
		for _, side := range []canon.Side{canon.Home, canon.Away} {
			team := m.HomeTeam
			if side == canon.Away {
				team = m.AwayTeam
			}
			ta := teams[team]
			if ta == nil {
				ta = &acc{}
				teams[team] = ta
			}
			ta.decay(m.StartAt, cfg.HalfLifeDays)
			gs := GoalieState{LeagueSV: lsv, TeamSV: shrunk(ta, lsv)}
			for _, g := range m.Goalies {
				if g.Side != side || !g.Starter {
					continue
				}
				key := teamKey(team, g.Player)
				ga := goalies[key]
				if ga == nil {
					ga = &acc{}
					goalies[key] = ga
				}
				ga.decay(m.StartAt, cfg.HalfLifeDays)
				gs.Starter = g.Player
				gs.StarterSV = shrunk(ga, lsv)
				gs.StarterShot = ga.shots
				gs.Known = true
				break
			}
			st[side] = gs
		}
		out[m.ID] = st
		// Update after the game.
		for _, g := range m.Goalies {
			team := m.HomeTeam
			if g.Side == canon.Away {
				team = m.AwayTeam
			}
			key := teamKey(team, g.Player)
			ga := goalies[key]
			if ga == nil {
				ga = &acc{}
				goalies[key] = ga
			}
			ga.decay(m.StartAt, cfg.HalfLifeDays)
			sv, sh := float64(g.Saves), float64(g.ShotsAgainst)
			ga.saves += sv
			ga.shots += sh
			teams[team].saves += sv
			teams[team].shots += sh
			league.saves += sv
			league.shots += sh
		}
	}
	return out
}

// Goalies are keyed per team: the same name on two clubs is treated as two
// goalies until players have ids (a transfer restarts the shrinkage, which is
// conservative).
func teamKey(team int32, name string) string {
	return fmt.Sprintf("%d|%s", team, name)
}
