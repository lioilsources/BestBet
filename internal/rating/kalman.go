package rating

import (
	"math"

	"github.com/lioilsources/BestBet/internal/canon"
)

// WaveConfig controls the team Kalman filter (plan 00 §1.4.2): each team has a
// level r (regulation goal difference against an average team on neutral ice)
// and a velocity v (change of r per game). Fixed a priori, docs/DECISIONS.md.
type WaveConfig struct {
	ObsVar       float64 // variance of one game's goal difference around expectation
	LevelNoise   float64 // process variance of r per game
	VelNoise     float64 // process variance of v per game
	VelDecay     float64 // v ← ρ·v per game (velocity is a short-lived wave)
	SeasonShrink float64 // r ← k·r at a new season (regression of squads)
	PriorLevel   float64 // variance of r for a team seen the first time / new season add-on
	PriorVel     float64 // variance of v at start
	HomeHalf     float64 // half-life in games of the league home-advantage estimate
}

// DefaultWaveConfig: one game of hockey goal difference has a variance of ~6;
// r may drift ~0.4 goals over a season; velocity decays within ~10 games.
func DefaultWaveConfig() WaveConfig {
	return WaveConfig{ObsVar: 6, LevelNoise: 0.003, VelNoise: 0.0002, VelDecay: 0.9,
		SeasonShrink: 0.75, PriorLevel: 0.3, PriorVel: 0.0025, HomeHalf: 500}
}

// WaveState is one team's pre-game filter state.
type WaveState struct {
	Level, Velocity float64
	LevelSD         float64
	Games           int // games seen by the filter
}

type kTeam struct {
	r, v          float64
	prr, prv, pvv float64 // covariance
	games, season int
}

func (t *kTeam) predict(c WaveConfig) {
	// F = [[1,1],[0,ρ]]
	r := t.r + t.v
	v := c.VelDecay * t.v
	prr := t.prr + 2*t.prv + t.pvv + c.LevelNoise
	prv := c.VelDecay * (t.prv + t.pvv)
	pvv := c.VelDecay*c.VelDecay*t.pvv + c.VelNoise
	t.r, t.v, t.prr, t.prv, t.pvv = r, v, prr, prv, pvv
}

// WaveTimeline runs the filter over matches in time order and returns each
// side's state before every match (causal). Only played matches update it.
func WaveTimeline(matches []canon.Match, c WaveConfig) map[int64]map[canon.Side]WaveState {
	ms := append([]canon.Match(nil), matches...)
	canon.SortMatches(ms)
	teams := map[int32]*kTeam{}
	out := map[int64]map[canon.Side]WaveState{}
	hfa, hfaW := 0.0, 0.0
	decay := math.Pow(0.5, 1/c.HomeHalf)
	get := func(id int32, season int) *kTeam {
		t := teams[id]
		if t == nil {
			t = &kTeam{prr: c.PriorLevel, pvv: c.PriorVel, season: season}
			teams[id] = t
		}
		if season != t.season {
			t.r *= c.SeasonShrink
			t.v = 0
			t.prr = c.SeasonShrink*c.SeasonShrink*t.prr + c.PriorLevel
			t.prv = 0
			t.pvv = c.PriorVel
			t.season = season
		}
		return t
	}
	for _, m := range ms {
		h, a := get(m.HomeTeam, m.Season), get(m.AwayTeam, m.Season)
		h.predict(c)
		a.predict(c)
		out[m.ID] = map[canon.Side]WaveState{
			canon.Home: {Level: h.r, Velocity: h.v, LevelSD: math.Sqrt(h.prr), Games: h.games},
			canon.Away: {Level: a.r, Velocity: a.v, LevelSD: math.Sqrt(a.prr), Games: a.games},
		}
		if !m.Played() {
			continue
		}
		gd := float64(m.Reg.Home - m.Reg.Away)
		home := 0.0
		if hfaW > 0 {
			home = hfa / hfaW
		}
		// Innovation against y = r_h − r_a + home + noise (team blocks independent).
		gap := h.r - a.r
		e := gd - home - gap
		s := h.prr + a.prr + c.ObsVar
		kr, kv := h.prr/s, h.prv/s
		h.r += kr * e
		h.v += kv * e
		h.prr, h.prv, h.pvv = h.prr-h.prr*h.prr/s, h.prv-h.prr*h.prv/s, h.pvv-h.prv*h.prv/s
		kr, kv = a.prr/s, a.prv/s
		a.r -= kr * e
		a.v -= kv * e
		a.prr, a.prv, a.pvv = a.prr-a.prr*a.prr/s, a.prv-a.prr*a.prv/s, a.pvv-a.prv*a.prv/s
		h.games++
		a.games++
		// League home advantage: decayed mean of (gd − pre-game strength gap).
		hfa = decay*hfa + (gd - gap)
		hfaW = decay*hfaW + 1
	}
	return out
}
