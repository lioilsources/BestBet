package periods

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

// SimParams describes a synthetic league with known truth (tests, harness checks).
type SimParams struct {
	Seed     uint64
	Teams    int
	Seasons  int
	Rounds   int     // double round-robin repeated Rounds/2 times per season
	Mu       float64 // log mean goals per team (regulation)
	HomeAdv  float64
	SpreadSD float64 // sd of att and def
	Start    time.Time
}

// DefaultSim mimics HockeyAllsvenskan: 14 teams, 52 games each, ~2.8 goals/team.
func DefaultSim() SimParams {
	return SimParams{Seed: 7, Teams: 14, Seasons: 3, Rounds: 4, Mu: math.Log(2.8), HomeAdv: 0.08, SpreadSD: 0.15,
		Start: time.Date(2022, 9, 15, 17, 0, 0, 0, time.UTC)}
}

// SimTruth is the generating parameters.
type SimTruth struct {
	Att, Def map[int32]float64
}

// Simulate produces played matches with periods, OT (50/50 + small strength
// effect) and shootouts. Truth parameters stay fixed across seasons.
func Simulate(p SimParams) ([]canon.Match, SimTruth) {
	r := rand.New(rand.NewPCG(p.Seed, p.Seed^0x9e3779b97f4a7c15))
	truth := SimTruth{Att: map[int32]float64{}, Def: map[int32]float64{}}
	for t := 1; t <= p.Teams; t++ {
		truth.Att[int32(t)] = r.NormFloat64() * p.SpreadSD
		truth.Def[int32(t)] = r.NormFloat64() * p.SpreadSD
	}
	profile := [3]float64{0.31, 0.33, 0.36}
	var out []canon.Match
	id := int64(1)
	for s := 0; s < p.Seasons; s++ {
		day := p.Start.AddDate(s, 0, 0)
		for round := 0; round < p.Rounds/2; round++ {
			for h := 1; h <= p.Teams; h++ {
				for a := 1; a <= p.Teams; a++ {
					if h == a {
						continue
					}
					lh := math.Exp(p.Mu + p.HomeAdv + truth.Att[int32(h)] - truth.Def[int32(a)])
					la := math.Exp(p.Mu + truth.Att[int32(a)] - truth.Def[int32(h)])
					m := canon.Match{ID: id, League: "sim", Season: 2022 + s, StartAt: day, HomeTeam: int32(h), AwayTeam: int32(a), Phase: "regular"}
					reg := canon.Score{}
					for k := 0; k < 3; k++ {
						ps := canon.Score{Home: poisson(r, lh*profile[k]), Away: poisson(r, la*profile[k])}
						m.Periods = append(m.Periods, ps)
						reg.Home += ps.Home
						reg.Away += ps.Away
					}
					m.Reg = &reg
					final := reg
					m.DecidedIn = "reg"
					if reg.Home == reg.Away {
						sh := truth.Att[int32(h)] + truth.Def[int32(h)] - truth.Att[int32(a)] - truth.Def[int32(a)]
						if r.Float64() < 1/(1+math.Exp(-(0.1+1.0*sh))) {
							final.Home++
						} else {
							final.Away++
						}
						m.DecidedIn = "ot"
						if r.Float64() < 0.4 {
							m.DecidedIn = "so"
						}
					}
					m.Final = &final
					out = append(out, m)
					id++
					// ~3.5 games per team per week
					if id%7 == 0 {
						day = day.Add(24 * time.Hour)
					}
				}
			}
		}
	}
	return out, truth
}

func poisson(r *rand.Rand, lambda float64) int {
	l, k, p := math.Exp(-lambda), 0, 1.0
	for {
		p *= r.Float64()
		if p <= l {
			return k
		}
		k++
	}
}
