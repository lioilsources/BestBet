package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/backtest"
	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/features"
	mp "github.com/lioilsources/BestBet/internal/model/marketprior"
)

// absInput is what the plan 04 L4 report needs.
type absInput struct {
	League  string
	Matches []canon.Match // with lineups attached
	Odds    map[int64][]canon.OddsQuote
	Base    []backtest.Row // periods-v1 walk (model prior)
	Sharp   string
	Holdout int
}

// absFeatures returns abs-est, abs-conf and abs-new per match (away − home).
func absFeatures(ms []canon.Match) map[int64]map[string]float64 {
	tl := features.AbsenceTimeline(ms)
	out := map[int64]map[string]float64{}
	for id, sides := range tl {
		h, a := sides[canon.Home], sides[canon.Away]
		x := map[string]float64{}
		if h.Known || a.Known {
			x["abs-est"] = a.Estimate - h.Estimate
			x["abs-conf"] = a.Confirmed - h.Confirmed
			x["abs-new"] = x["abs-conf"] - x["abs-est"]
		}
		out[id] = x
	}
	return out
}

type olsFit struct {
	n            int
	beta, lo, hi float64
	r2           float64
}

func ols(x, y []float64) (beta, r2 float64) {
	n := float64(len(x))
	var mx, my float64
	for i := range x {
		mx += x[i] / n
		my += y[i] / n
	}
	var sxy, sxx, syy float64
	for i := range x {
		sxy += (x[i] - mx) * (y[i] - my)
		sxx += (x[i] - mx) * (x[i] - mx)
		syy += (y[i] - my) * (y[i] - my)
	}
	if sxx == 0 || syy == 0 {
		return 0, 0
	}
	return sxy / sxx, sxy * sxy / (sxx * syy)
}

// bootOLS fits y = a + βx with a bootstrap interval over games (fixed seed).
func bootOLS(x, y []float64) olsFit {
	f := olsFit{n: len(x)}
	if len(x) < 3 {
		return f
	}
	f.beta, f.r2 = ols(x, y)
	r := rand.New(rand.NewSource(1))
	bs := make([]float64, 0, 2000)
	bx, by := make([]float64, len(x)), make([]float64, len(x))
	for k := 0; k < 2000; k++ {
		for i := range x {
			j := r.Intn(len(x))
			bx[i], by[i] = x[j], y[j]
		}
		b, _ := ols(bx, by)
		bs = append(bs, b)
	}
	sort.Float64s(bs)
	f.lo, f.hi = bs[50], bs[1949]
	return f
}

func logitHome(p [3]float64) float64 {
	q := p[0] / (p[0] + p[2])
	return math.Log(q / (1 - q))
}

type absVariant struct {
	label  string
	layers []string
	pred   map[int64][3]float64
	betas  map[string][]float64
}

type absPrior struct {
	name     string // opening, t24h, t1h, closing or model
	obs      []mp.Obs
	byID     map[int64]mp.Obs
	base     *absVariant
	variants []*absVariant
}

func (p *absPrior) delta(v *absVariant) backtest.Delta {
	var d []float64
	for id, pv := range v.pred {
		pb, ok := p.base.pred[id]
		if !ok {
			continue
		}
		y := p.byID[id].Y
		d = append(d, nll(pv, y)-nll(pb, y))
	}
	return meanCI(d)
}

func (p *absPrior) ll(v *absVariant) float64 {
	if len(v.pred) == 0 {
		return 0
	}
	s := 0.0
	for id, pv := range v.pred {
		s += nll(pv, p.byID[id].Y)
	}
	return s / float64(len(v.pred))
}

func absReport(in absInput, generated time.Time) string {
	abs := absFeatures(in.Matches)
	ctx := m1bFeatures(m1bInput{League: in.League, Matches: in.Matches})
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# M3b L4: absence hráčů proti trhu: %s\n\n", in.League)
	w("Vygenerováno %s příkazem `backtest -absence`. Konfigurace pevně předem: `docs/DECISIONS.md` 2026-11-13 (L4). Prahy absencí z plánu 04 L2: pravidelný hráč ≥ 60 %% z posledních 20 zápasů sezony a TOI nad mediánem týmu, brankáři mimo.\n\n", generated.UTC().Format("2006-01-02 15:04 UTC"))
	w("- `abs-est`: podíl chybějících minut pravidelných hráčů, hosté − domácí, sestava = poslední zápas týmu.\n- `abs-conf`: totéž se skutečnou sestavou zápasu (potvrzená sestava; v backtestu zpětný pohled).\n- `abs-new`: abs-conf − abs-est, tedy co přinese až potvrzená sestava.\n\n")

	// Sample.
	var nReg, nKnown, nzEst, nzConf int
	var sumEst, sumConf float64
	for _, m := range in.Matches {
		if !m.Played() || m.Phase != "regular" || len(m.Lineups) == 0 {
			continue
		}
		nReg++
		x, ok := abs[m.ID]
		if !ok || len(x) == 0 {
			continue
		}
		nKnown++
		if x["abs-est"] != 0 {
			nzEst++
			sumEst += math.Abs(x["abs-est"])
		}
		if x["abs-conf"] != 0 {
			nzConf++
			sumConf += math.Abs(x["abs-conf"])
		}
	}
	w("## Vzorek\n\nZápasů základní části se sestavou: %d, s absencemi spočítanými aspoň u jedné strany: %d. ", nReg, nKnown)
	w("abs-est ≠ 0 v %d zápasech (průměr |x| %s), abs-conf ≠ 0 v %d (průměr |x| %s).\n\n",
		nzEst, cz("%.3f", sumEst/math.Max(1, float64(nzEst))), nzConf, cz("%.3f", sumConf/math.Max(1, float64(nzConf))))

	// H7.
	w("## H7: vysvětlují absence pohyb Pinnaclu mezi t24h a uzavřením?\n\n")
	w("y = logit q_close − logit q_t24h, q = p_dom / (p_dom + p_hosté), Shin celého 1X2 Pinnaclu. OLS y = a + β·x, interval bootstrap po zápasech (2 000×). Kladné β: chybí-li víc hostům, trh posune k domácím.\n\n")
	w("| Proměnná | n | β | 95%% interval | R² |\n|---|---|---|---|---|\n")
	for _, name := range []string{"abs-est", "abs-conf", "abs-new"} {
		var xs, ys []float64
		for _, m := range in.Matches {
			if !m.Played() || m.Phase != "regular" {
				continue
			}
			x, ok := abs[m.ID][name]
			if !ok {
				continue
			}
			p24, ok1 := sharpAt(in.Odds[m.ID], "t24h", in.Sharp)
			pc, ok2 := sharpAt(in.Odds[m.ID], "closing", in.Sharp)
			if !ok1 || !ok2 {
				continue
			}
			xs = append(xs, x)
			ys = append(ys, logitHome(pc)-logitHome(p24))
		}
		f := bootOLS(xs, ys)
		w("| %s | %d | %s | %s až %s | %s |\n", name, f.n, cz("%+.3f", f.beta), cz("%+.3f", f.lo), cz("%+.3f", f.hi), cz("%.4f", f.r2))
	}

	// F-abs.
	baseP := map[int64][3]float64{}
	for _, r := range in.Base {
		x := r.Pred.Markets["1x2"]
		baseP[r.Match.ID] = [3]float64{x["home"], x["draw"], x["away"]}
	}
	var priors []*absPrior
	build := func(name string, prior func(m canon.Match) ([3]float64, bool), baseLayers []string) *absPrior {
		p := &absPrior{name: name, byID: map[int64]mp.Obs{}}
		for _, m := range in.Matches {
			if !m.Played() || m.Phase != "regular" || (in.Holdout != 0 && m.Season < in.Holdout) {
				continue
			}
			pr, ok := prior(m)
			if !ok {
				continue
			}
			x := map[string]float64{}
			for k, v := range ctx[m.ID] {
				x[k] = v
			}
			for k, v := range abs[m.ID] {
				x[k] = v
			}
			o := mp.Obs{ID: m.ID, At: m.StartAt, Prior: pr, X: x, Y: outcome(m)}
			p.obs = append(p.obs, o)
			p.byID[m.ID] = o
		}
		p.base = &absVariant{label: "bez absence", layers: baseLayers}
		p.variants = []*absVariant{
			{label: "+abs-est", layers: append(append([]string{}, baseLayers...), "abs-est")},
			{label: "+abs-conf", layers: append(append([]string{}, baseLayers...), "abs-conf")},
		}
		for _, v := range append([]*absVariant{p.base}, p.variants...) {
			v.pred, v.betas = mp.Walk(p.obs, v.layers, mp.Config{}, 100)
		}
		return p
	}
	for _, kind := range mp.Kinds {
		k := kind
		priors = append(priors, build(k, func(m canon.Match) ([3]float64, bool) {
			pr, ok := mp.PriorAt(in.Odds[m.ID], k, in.Sharp)
			return pr.P, ok
		}, nil))
	}
	model := build("model", func(m canon.Match) ([3]float64, bool) {
		p, ok := baseP[m.ID]
		return p, ok
	}, []string{"goalie", "b2b"})

	type test struct {
		p   *absPrior
		v   *absVariant
		d   backtest.Delta
		sig bool
	}
	var tests []*test
	var pv []float64
	all := map[*absVariant]*test{}
	for _, p := range append(append([]*absPrior{}, priors...), model) {
		for _, v := range p.variants {
			t := &test{p: p, v: v, d: p.delta(v)}
			all[v] = t
			if p.name != "closing" {
				tests = append(tests, t)
				pv = append(pv, t.d.P)
			}
		}
	}
	for i, s := range backtest.BenjaminiHochberg(pv, 0.10) {
		tests[i].sig = s
	}
	table := func(p *absPrior) {
		w("| Varianta | Zápasů | Δ log-loss | Δ / LL | 95%% interval | p | BH | medián β |\n|---|---|---|---|---|---|---|---|\n")
		base := p.ll(p.base)
		w("| %s | %d | – | – | – | – | – | – |\n", p.base.label, len(p.base.pred))
		for _, v := range p.variants {
			t := all[v]
			bh := "—"
			if p.name != "closing" {
				bh = "ne"
				if t.sig && t.d.MeanDelta1X2 < 0 {
					bh = "**ano**"
				}
			}
			rel := 0.0
			if base > 0 {
				rel = t.d.MeanDelta1X2 / base
			}
			l := v.layers[len(v.layers)-1]
			w("| %s | %d | %s | %s | %s až %s | %s | %s | %s |\n", v.label, t.d.N, cz("%+.4f", t.d.MeanDelta1X2), cz("%+.2f %%", 100*rel),
				cz("%+.4f", t.d.CILo), cz("%+.4f", t.d.CIHi), cz("%.3f", t.d.P), bh, cz("%.3f", median(v.betas[l])))
		}
		w("\n")
	}
	w("\n## F-abs: absence nad trhem jako priorem\n\nOdložené sezony od %d/%02d. Prior = Shin snímku (Pinnacle, jinak medián knih). Δ = párový rozdíl log-loss 1X2 proti trhu samotnému, záporné = absence trh zlepšuje. BH (q = 0,10) přes všechny varianty s absencí mimo closing.\n\n", in.Holdout, (in.Holdout+1)%100)
	for _, p := range priors {
		w("### Snímek %s (log-loss trhu %s)\n\n", p.name, cz("%.4f", p.ll(p.base)))
		table(p)
	}
	w("## F-abs: absence nad modelem\n\nPrior = out-of-sample `periods-v1`, vrstvy goalie + b2b ve všech variantách (log-loss bez absence %s). Δ proti variantě bez absence.\n\n", cz("%.4f", model.ll(model.base)))
	table(model)

	// Gate B2.
	w("## Brána B2\n\nProjde abs-est na opening, t24h nebo t1h, nebo abs-conf na t1h, proti trhu jako prioru: Δ < 0, 95%% interval pod nulou a BH.\n\n| Proměnná | Snímek | Δ | Interval pod nulou | BH | Prošla |\n|---|---|---|---|---|---|\n")
	pass := false
	for _, p := range priors {
		for _, v := range p.variants {
			if p.name == "closing" || (v.label == "+abs-conf" && p.name != "t1h") {
				continue
			}
			t := all[v]
			ok := t.d.MeanDelta1X2 < 0 && t.d.CIHi < 0 && t.sig
			pass = pass || ok
			w("| %s | %s | %s | %s | %s | %s |\n", strings.TrimPrefix(v.label, "+"), p.name, cz("%+.4f", t.d.MeanDelta1X2), yes(t.d.CIHi < 0), yes(t.sig), yes(ok))
		}
	}
	if pass {
		w("\n**Verdikt B2: prošlo.** Podle plánu 04 §6 přijde návrh nové verze hintu, než se zapíše první záznam.\n")
	} else {
		w("\n**Verdikt B2: neprošlo.** Absence zůstávají jen informací v UI; model ani hint se nemění.\n")
	}
	return b.String()
}
