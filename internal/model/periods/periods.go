// Package periods is the baseline hockey model (plan 00 §1.4.5, 01 §2.2):
//
//	log λ_home = μ + H + att[home] − def[away]
//	log λ_away = μ     + att[away] − def[home]
//
// fitted on regulation goals by time-weighted Poisson maximum likelihood with a
// ridge prior (promoted teams start at league average). Goals per period follow
// the league profile π₁..π₃. Regulation ties get one extra parameter δ
// (P ∝ Poisson·(1+δ·[h=a])) because pulled goalies make independent Poisson
// under-predict draws. OT/SO is a separate logistic step on the strength gap.
//
// No wave, goalie or lineup input: this is the benchmark every layer is measured
// against (01 §5 fáze 1).
package periods

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"gonum.org/v1/gonum/optimize"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/model/markets"
)

// Version identifies this model in predictions.
const Version = "periods-v1"

// Config holds hyperparameters. They are fixed a priori (docs/DECISIONS.md),
// not tuned on held-out seasons.
type Config struct {
	HalfLifeDays float64 // time weight half-life
	Ridge        float64 // L2 penalty on att/def (in units of weighted games)
	MaxGoals     int     // matrix size per side
	TieInflation bool    // fit δ
}

// DefaultConfig is the baseline.
func DefaultConfig() Config {
	return Config{HalfLifeDays: 180, Ridge: 5, MaxGoals: 12, TieInflation: true}
}

// Model is a fitted baseline.
type Model struct {
	Cfg     Config
	AsOf    time.Time
	Mu      float64
	HomeAdv float64
	Att     map[int32]float64
	Def     map[int32]float64
	Profile [3]float64 // share of regulation goals per period
	Delta   float64    // tie inflation
	OTGamma float64    // OT/SO home intercept
	OTBeta  float64    // OT/SO slope on strength gap
	NGames  int
}

// ErrNoData is returned when no played game precedes asOf.
var ErrNoData = errors.New("periods: no training games")

type obs struct {
	h, a   int // team indices
	gh, ga float64
	w      float64
}

// Fit trains on played matches strictly before asOf.
func Fit(train []canon.Match, asOf time.Time, cfg Config) (*Model, error) {
	idx := map[int32]int{}
	var ids []int32
	team := func(id int32) int {
		if i, ok := idx[id]; ok {
			return i
		}
		idx[id] = len(ids)
		ids = append(ids, id)
		return len(ids) - 1
	}
	var data []obs
	var games []canon.Match
	var periodGoals [3]float64
	for _, m := range train {
		if !m.Played() || !m.StartAt.Before(asOf) {
			continue
		}
		age := asOf.Sub(m.StartAt).Hours() / 24
		w := math.Pow(0.5, age/cfg.HalfLifeDays)
		data = append(data, obs{team(m.HomeTeam), team(m.AwayTeam), float64(m.Reg.Home), float64(m.Reg.Away), w})
		games = append(games, m)
		for k := 0; k < 3; k++ {
			periodGoals[k] += w * float64(m.Periods[k].Home+m.Periods[k].Away)
		}
	}
	if len(data) == 0 {
		return nil, ErrNoData
	}
	nT := len(ids)
	// x = [μ, H, att_0..att_{n-1}, def_0..def_{n-1}]
	nx := 2 + 2*nT
	f := func(x []float64) float64 {
		v, _ := nll(x, data, nT, cfg.Ridge, nil)
		return v
	}
	grad := func(g, x []float64) { nll(x, data, nT, cfg.Ridge, g) }
	x0 := make([]float64, nx)
	x0[0] = math.Log(meanGoals(data))
	res, err := optimize.Minimize(optimize.Problem{Func: f, Grad: grad}, x0, &optimize.Settings{GradientThreshold: 1e-7}, &optimize.LBFGS{})
	if err != nil && res == nil {
		return nil, fmt.Errorf("periods fit: %w", err)
	}
	x := res.X
	m := &Model{Cfg: cfg, AsOf: asOf, Mu: x[0], HomeAdv: x[1], Att: map[int32]float64{}, Def: map[int32]float64{}, NGames: len(data)}
	for i, id := range ids {
		m.Att[id] = x[2+i]
		m.Def[id] = x[2+nT+i]
	}
	tot := periodGoals[0] + periodGoals[1] + periodGoals[2]
	for k := range m.Profile {
		m.Profile[k] = periodGoals[k] / tot
	}
	if cfg.TieInflation {
		m.Delta = m.fitDelta(games, data)
	}
	m.fitOT(games, data)
	return m, nil
}

// nll returns the penalised negative log-likelihood and fills g when non-nil.
func nll(x []float64, data []obs, nT int, ridge float64, g []float64) (float64, error) {
	mu, H := x[0], x[1]
	att, def := x[2:2+nT], x[2+nT:]
	if g != nil {
		for i := range g {
			g[i] = 0
		}
	}
	v := 0.0
	for _, o := range data {
		lh := math.Exp(mu + H + att[o.h] - def[o.a])
		la := math.Exp(mu + att[o.a] - def[o.h])
		v -= o.w * (o.gh*math.Log(lh) - lh + o.ga*math.Log(la) - la)
		if g != nil {
			rh := o.w * (lh - o.gh) // d(-ll)/d(log λh)
			ra := o.w * (la - o.ga)
			g[0] += rh + ra
			g[1] += rh
			g[2+o.h] += rh
			g[2+nT+o.a] -= rh
			g[2+o.a] += ra
			g[2+nT+o.h] -= ra
		}
	}
	for i := 0; i < 2*nT; i++ {
		p := x[2+i]
		v += ridge / 2 * p * p
		if g != nil {
			g[2+i] += ridge * p
		}
	}
	return v, nil
}

func meanGoals(data []obs) float64 {
	s, w := 0.0, 0.0
	for _, o := range data {
		s += o.w * (o.gh + o.ga) / 2
		w += o.w
	}
	return math.Max(s/w, 0.1)
}

// Lambdas returns expected regulation goals. Unknown teams sit at league average.
func (m *Model) Lambdas(home, away int32) (float64, float64) {
	lh := math.Exp(m.Mu + m.HomeAdv + m.Att[home] - m.Def[away])
	la := math.Exp(m.Mu + m.Att[away] - m.Def[home])
	return lh, la
}

// Strength is att + def (higher = better), the OT/SO covariate.
func (m *Model) Strength(team int32) float64 { return m.Att[team] + m.Def[team] }

func poissonPMF(lambda float64, n int) []float64 {
	p := make([]float64, n+1)
	p[0] = math.Exp(-lambda)
	for k := 1; k <= n; k++ {
		p[k] = p[k-1] * lambda / float64(k)
	}
	return p
}

// Matrix builds a (MaxGoals+1)² distribution; delta inflates the diagonal.
func Matrix(lh, la float64, maxGoals int, delta float64) canon.ScoreMatrix {
	ph, pa := poissonPMF(lh, maxGoals), poissonPMF(la, maxGoals)
	s := make(canon.ScoreMatrix, maxGoals+1)
	tot := 0.0
	for h := range s {
		s[h] = make([]float64, maxGoals+1)
		for a := range s[h] {
			v := ph[h] * pa[a]
			if h == a {
				v *= 1 + delta
			}
			s[h][a] = v
			tot += v
		}
	}
	for h := range s {
		for a := range s[h] {
			s[h][a] /= tot
		}
	}
	return s
}

func tieProb(lh, la float64, maxGoals int) float64 {
	ph, pa := poissonPMF(lh, maxGoals), poissonPMF(la, maxGoals)
	t := 0.0
	for k := 0; k <= maxGoals; k++ {
		t += ph[k] * pa[k]
	}
	return t
}

// fitDelta maximises Σ w [tie·log(1+δ) − log(1+δ·T)] by golden-section search.
func (m *Model) fitDelta(games []canon.Match, data []obs) float64 {
	type row struct {
		tie  bool
		t, w float64
	}
	rows := make([]row, len(games))
	for i, g := range games {
		lh, la := m.Lambdas(g.HomeTeam, g.AwayTeam)
		rows[i] = row{g.Reg.Home == g.Reg.Away, tieProb(lh, la, m.Cfg.MaxGoals), data[i].w}
	}
	ll := func(d float64) float64 {
		v := 0.0
		for _, r := range rows {
			if r.tie {
				v += r.w * math.Log(1+d)
			}
			v -= r.w * math.Log(1+d*r.t)
		}
		return v
	}
	return golden(ll, -0.5, 2)
}

// fitOT fits P(home wins | tied after 60) = sigmoid(γ + β·(s_home − s_away)).
func (m *Model) fitOT(games []canon.Match, data []obs) {
	var xs, ws []float64
	var ys []bool
	for i, g := range games {
		if g.Final == nil || (g.DecidedIn != "ot" && g.DecidedIn != "so") {
			continue
		}
		xs = append(xs, m.Strength(g.HomeTeam)-m.Strength(g.AwayTeam))
		ys = append(ys, g.Final.Home > g.Final.Away)
		ws = append(ws, data[i].w)
	}
	m.OTGamma, m.OTBeta = logistic(xs, ys, ws, 1.0)
}

// OTHomeWin is P(home wins OT/SO | tie).
func (m *Model) OTHomeWin(home, away int32) float64 {
	return sigmoid(m.OTGamma + m.OTBeta*(m.Strength(home)-m.Strength(away)))
}

// Predict returns the full prediction for a match.
func (m *Model) Predict(match canon.Match) (canon.Prediction, error) {
	lh, la := m.Lambdas(match.HomeTeam, match.AwayTeam)
	return m.PredictLambdas(match, lh, la, nil)
}

// PredictLambdas builds the prediction from given regulation intensities (layers
// on top of the baseline adjust λ and call this). extra goes into the inputs hash.
func (m *Model) PredictLambdas(match canon.Match, lh, la float64, extra any) (canon.Prediction, error) {
	reg := Matrix(lh, la, m.Cfg.MaxGoals, m.Delta)
	p1 := Matrix(lh*m.Profile[0], la*m.Profile[0], m.Cfg.MaxGoals, 0)
	ot := m.OTHomeWin(match.HomeTeam, match.AwayTeam)
	hash, err := canon.HashInputs(struct {
		V     string
		AsOf  time.Time
		N     int
		H, A  int32
		Extra any
	}{Version, m.AsOf, m.NGames, match.HomeTeam, match.AwayTeam, extra})
	if err != nil {
		return canon.Prediction{}, err
	}
	return canon.Prediction{
		MatchID: match.ID, ModelVersion: Version, InputsHash: hash, ComputedAt: m.AsOf,
		LambdaHome: lh, LambdaAway: la, Matrix: reg, P1Matrix: p1, OTHomeWin: ot,
		Markets: markets.Derive(reg, p1, ot),
	}, nil
}

// Weight is the time weight of a game played at t for a model fitted at asOf.
func (m *Model) Weight(t time.Time) float64 {
	return math.Pow(0.5, m.AsOf.Sub(t).Hours()/24/m.Cfg.HalfLifeDays)
}

// HomeWin is P(home wins in regulation) for given intensities (for contributions).
func (m *Model) HomeWin(lh, la float64) float64 {
	mx := Matrix(lh, la, m.Cfg.MaxGoals, m.Delta)
	p := 0.0
	for h := range mx {
		for a := 0; a < h; a++ {
			p += mx[h][a]
		}
	}
	return p
}

// Golden maximises f on [lo, hi] (exported for layers fitting one coefficient).
func Golden(f func(float64) float64, lo, hi float64) float64 { return golden(f, lo, hi) }

// Teams returns team ids sorted by strength (for reports).
func (m *Model) Teams() []int32 {
	ids := make([]int32, 0, len(m.Att))
	for id := range m.Att {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return m.Strength(ids[i]) > m.Strength(ids[j]) })
	return ids
}

func golden(f func(float64) float64, lo, hi float64) float64 {
	const r = 0.6180339887498949
	a, b := lo, hi
	c, d := b-r*(b-a), a+r*(b-a)
	fc, fd := f(c), f(d)
	for i := 0; i < 100 && b-a > 1e-7; i++ {
		if fc > fd {
			b, d, fd = d, c, fc
			c = b - r*(b-a)
			fc = f(c)
		} else {
			a, c, fc = c, d, fd
			d = a + r*(b-a)
			fd = f(d)
		}
	}
	return (a + b) / 2
}

// logistic fits weighted logistic regression y ~ γ + β·x with a ridge on β.
func logistic(xs []float64, ys []bool, ws []float64, ridge float64) (float64, float64) {
	g, b := 0.0, 0.0
	if len(xs) == 0 {
		return 0, 0
	}
	for iter := 0; iter < 50; iter++ {
		var g0, g1, h00, h01, h11 float64
		for i, x := range xs {
			p := sigmoid(g + b*x)
			y := 0.0
			if ys[i] {
				y = 1
			}
			d := ws[i] * (p - y)
			v := ws[i] * p * (1 - p)
			g0 += d
			g1 += d * x
			h00 += v
			h01 += v * x
			h11 += v * x * x
		}
		g1 += ridge * b
		h11 += ridge
		h00 += 1e-6
		det := h00*h11 - h01*h01
		if det <= 0 {
			break
		}
		dg := (h11*g0 - h01*g1) / det
		db := (h00*g1 - h01*g0) / det
		g -= dg
		b -= db
		if math.Abs(dg)+math.Abs(db) < 1e-10 {
			break
		}
	}
	return g, b
}

func sigmoid(x float64) float64 { return 1 / (1 + math.Exp(-x)) }
