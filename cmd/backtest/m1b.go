package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/backtest"
	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/features"
	"github.com/lioilsources/BestBet/internal/model/layers"
	mp "github.com/lioilsources/BestBet/internal/model/marketprior"
)

// M1b F1 layers in report order (docs/DECISIONS.md 2026-11-08).
var m1bLayers = []string{"goalie", "wave-level", "wave-vel", "b2b", "rest-days", "pressure", "coach-change", "periods"}

// m1bInput is what the market-prior report needs.
type m1bInput struct {
	League  string
	Matches []canon.Match
	Odds    map[int64][]canon.OddsQuote
	Base    []backtest.Row // periods baseline walk (for the "periods" layer)
	Tenures []features.CoachTenure
	Sharp   string
	Books   []string // CZ books for opening bets
	Holdout int
}

// m1bFeatures returns the prior-independent layer features of every match.
func m1bFeatures(in m1bInput) map[int64]map[string]float64 {
	li := layers.BuildInputs(in.Matches)
	var press map[int64]map[canon.Side]bool
	if c := features.PressureConfigFor(in.League); c != nil {
		press = features.PressureTimeline(in.Matches, *c)
	}
	coach := features.CoachTimeline(in.Matches, in.Tenures)
	b := func(v bool) float64 {
		if v {
			return 1
		}
		return 0
	}
	out := map[int64]map[string]float64{}
	for _, m := range in.Matches {
		x := map[string]float64{}
		if g, ok := li.Goalie[m.ID]; ok {
			x["goalie"] = g[canon.Away].Adj() - g[canon.Home].Adj()
		}
		if w, ok := li.Wave[m.ID]; ok && w[canon.Home].Games >= 5 && w[canon.Away].Games >= 5 {
			x["wave-level"] = w[canon.Home].Level - w[canon.Away].Level
			x["wave-vel"] = 10 * (w[canon.Home].Velocity - w[canon.Away].Velocity)
		}
		if r, ok := li.Rest[m.ID]; ok {
			x["b2b"] = b(r[canon.Away].BackToBack) - b(r[canon.Home].BackToBack)
			if r[canon.Home].Known && r[canon.Away].Known {
				f := func(d float64) float64 { return math.Log(1 + math.Min(d, 4)) }
				x["rest-days"] = f(r[canon.Home].DaysSinceLast) - f(r[canon.Away].DaysSinceLast)
			}
		}
		if p, ok := press[m.ID]; ok {
			x["pressure"] = b(p[canon.Home]) - b(p[canon.Away])
		}
		if c, ok := coach[m.ID]; ok {
			x["coach-change"] = b(c[canon.Home]) - b(c[canon.Away])
		}
		out[m.ID] = x
	}
	return out
}

type m1bVariant struct {
	label  string
	layers []string
	pred   map[int64][3]float64
	betas  map[string][]float64
}

type m1bKind struct {
	kind     string
	obs      []mp.Obs
	byID     map[int64]mp.Obs
	sources  map[string]int
	variants []*m1bVariant // [0] = market alone
}

// m1bReport runs the walk for every prior kind and layer and renders F1.
func m1bReport(in m1bInput, generated time.Time) string {
	feats := m1bFeatures(in)
	baseP := map[int64][3]float64{}
	for _, r := range in.Base {
		x := r.Pred.Markets["1x2"]
		baseP[r.Match.ID] = [3]float64{x["home"], x["draw"], x["away"]}
	}
	var kinds []*m1bKind
	for _, kind := range mp.Kinds {
		k := &m1bKind{kind: kind, byID: map[int64]mp.Obs{}, sources: map[string]int{}}
		for _, m := range in.Matches {
			if !m.Played() || m.Phase != "regular" || (in.Holdout != 0 && m.Season < in.Holdout) {
				continue
			}
			pr, ok := mp.PriorAt(in.Odds[m.ID], kind, in.Sharp)
			if !ok {
				continue
			}
			x := map[string]float64{}
			for l, v := range feats[m.ID] {
				x[l] = v
			}
			if pb, ok := baseP[m.ID]; ok {
				x["periods"] = math.Log(pb[0]/pb[2]) - math.Log(pr.P[0]/pr.P[2])
			}
			o := mp.Obs{ID: m.ID, At: m.StartAt, Prior: pr.P, X: x, Y: outcome(m)}
			k.obs = append(k.obs, o)
			k.byID[m.ID] = o
			k.sources[pr.Source]++
		}
		vs := []*m1bVariant{{label: "trh sám"}}
		for _, l := range m1bLayers {
			vs = append(vs, &m1bVariant{label: l, layers: []string{l}})
		}
		vs = append(vs, &m1bVariant{label: "vše", layers: m1bLayers})
		for _, v := range vs {
			v.pred, v.betas = mp.Walk(k.obs, v.layers, mp.Config{}, 100)
		}
		k.variants = vs
		kinds = append(kinds, k)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# M1b F1 — vlna předpokladů proti trhu: %s\n\n", in.League)
	fmt.Fprintf(&b, "Model `%s`, vygenerováno %s příkazem `backtest -m1b`. Konfigurace pevně předem: `docs/DECISIONS.md` 2026-11-08. Hodnotí se jen základní část odložených sezon (od %d/%02d), kde jsou kurzy.\n\n",
		mp.Version, generated.UTC().Format("2006-01-02 15:04 UTC"), in.Holdout, (in.Holdout+1)%100)
	b.WriteString("Prior = odmaržovaný (Shin) kurz daného snímku, Pinnacle, kde má celé 1X2, jinak medián knih. Vrstva posouvá log-šance domácí/hosté o β·x; β se fituje walk-forward po týdnech na dřívějších zápasech (ridge 1, vrstva s < 100 aktivními zápasy má β = 0, první okna s < 100 tréninkovými zápasy se přeskočí). Δ je párový rozdíl log-loss 1X2 (vrstva − trh sám) na stejných zápasech, záporné = vrstva trh zlepšuje.\n\n")

	b.WriteString("## Vzorek\n\n| Snímek | Zápasů s priorem | z toho Pinnacle | medián knih | hodnocených (po zahřátí) | Log-loss trhu |\n|---|---|---|---|---|---|\n")
	for _, k := range kinds {
		ids := ids(k.variants[0].pred)
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %s |\n", k.kind, len(k.obs), k.sources[in.Sharp], k.sources["median"], len(ids), cz("%.4f", ll(k, k.variants[0], ids)))
	}

	if s, one, g := pressureShares(in, kinds[0]); g > 0 {
		fmt.Fprintf(&b, "\nPressure (snímek opening): v boji o čáru je %s týmů v hodnocených zápasech, jen jedna strana v %d z %d zápasů; jen ty dávají x ≠ 0.\n", cz("%.0f %%", 100*float64(s)/float64(2*g)), one, g)
	}

	// Paired deltas and BH over single layers × non-closing kinds.
	type test struct {
		k   *m1bKind
		v   *m1bVariant
		d   backtest.Delta
		rel float64
	}
	var tests []test
	var pv []float64
	all := map[*m1bVariant]test{}
	for _, k := range kinds {
		for _, v := range k.variants[1:] {
			d := pairedDelta(k, k.variants[0], v)
			base := ll(k, k.variants[0], ids(k.variants[0].pred))
			t := test{k, v, d, 0}
			if base > 0 {
				t.rel = d.MeanDelta1X2 / base
			}
			all[v] = t
			if k.kind != "closing" && len(v.layers) == 1 {
				tests = append(tests, t)
				pv = append(pv, d.P)
			}
		}
	}
	sig := backtest.BenjaminiHochberg(pv, 0.10)
	passLL := map[string][]string{}
	sigOf := map[*m1bVariant]bool{}
	for i, t := range tests {
		sigOf[t.v] = sig[i]
		if sig[i] && t.rel <= -0.005 {
			passLL[t.v.label] = append(passLL[t.v.label], t.k.kind)
		}
	}

	b.WriteString("\n## Vrstvy proti trhu (log-loss)\n\nBH (q = 0,10) přes jednotlivé vrstvy × snímky opening, t24h, t1h. `closing` je diagnostika: když vrstva pomáhá i proti uzavíracímu kurzu, trh ji nezapracoval ani do začátku.\n\n")
	for _, k := range kinds {
		fmt.Fprintf(&b, "### Snímek %s\n\n| Vrstva | Zápasů | aktivní | Δ | Δ / LL trhu | 95%% interval | p | BH | medián β |\n|---|---|---|---|---|---|---|---|---|\n", k.kind)
		for _, v := range k.variants[1:] {
			t := all[v]
			bh := "—"
			if k.kind != "closing" && len(v.layers) == 1 {
				bh = "ne"
				if sigOf[v] && t.d.MeanDelta1X2 < 0 {
					bh = "**ano**"
				}
			}
			var bs []string
			for _, l := range v.layers {
				bs = append(bs, cz("%.3f", median(v.betas[l])))
			}
			fmt.Fprintf(&b, "| %s | %d | %d | %s | %s | %s až %s | %s | %s | %s |\n", v.label, t.d.N, activeObs(k, v), cz("%+.4f", t.d.MeanDelta1X2),
				cz("%+.2f %%", 100*t.rel), cz("%+.4f", t.d.CILo), cz("%+.4f", t.d.CIHi), cz("%.3f", t.d.P), bh, strings.Join(bs, " / "))
		}
		b.WriteString("\n")
	}

	// Market move and opening bets (opening prior from the sharp book only).
	var open *m1bKind
	for _, k := range kinds {
		if k.kind == "opening" {
			open = k
		}
	}
	b.WriteString("## Předjímá vrstva pohyb trhu? Sázky v otevíracím kurzu\n\n")
	fmt.Fprintf(&b, "Korelace (p_upr − p_open) s (p_close − p_open) pro výhru domácích, obojí %s. Sázky: otevírací kurz u %s, hrana ≥ 3 p. b. proti jejich Shinovi, EV > 0, kurz ≤ 10, jedna sázka na zápas; CLV = log(kurz × p uzavíracího %s), yield při flat vkladu.\n\n",
		in.Sharp, strings.Join(in.Books, ", "), in.Sharp)
	b.WriteString("| Varianta | Korelace s pohybem | n | Sázek | s uzavíracím Pinnaclem | CLV | 95% interval CLV | Yield |\n|---|---|---|---|---|---|---|---|\n")
	passCLV := map[string]bool{}
	for _, v := range open.variants {
		c, n := moveCorr(in, open, v)
		corr := "—"
		if anyBeta(v) && n >= 3 {
			corr = cz("%.3f", c)
		}
		s := openBets(in, open, v)
		clv, ci, y := "—", "—", "—"
		if s.n > 0 {
			y = cz("%+.1f %%", 100*s.yield)
		}
		if s.nCLV >= 2 {
			clv = cz("%+.2f %%", 100*s.clv)
			ci = cz("%+.2f", 100*s.lo) + " až " + cz("%+.2f %%", 100*s.hi)
		}
		if len(v.layers) == 1 && s.nCLV >= 100 && s.lo > 0 {
			passCLV[v.label] = true
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %s | %s | %s |\n", v.label, corr, n, s.n, s.nCLV, clv, ci, y)
	}
	b.WriteString("\nMimo plán (průzkumné, ne brána): sázky „trh sám“ rozdělené podle zdroje otevíracího prioru. U Pinnaclu je hrana rozdíl CZ knihy proti sharp kurzu; u mediánu jde o rozdíl mezi knihami navzájem.\n\n| Prior | Sázek | s uzavíracím Pinnaclem | CLV | 95% interval CLV | Yield |\n|---|---|---|---|---|---|\n")
	for _, src := range []string{in.Sharp, "median"} {
		s := openBetsWhere(in, open, open.variants[0], func(id int64) bool {
			pr, ok := mp.PriorAt(in.Odds[id], "opening", in.Sharp)
			return ok && pr.Source == src
		})
		clv, ci := "—", "—"
		if s.nCLV >= 2 {
			clv = cz("%+.2f %%", 100*s.clv)
			ci = cz("%+.2f", 100*s.lo) + " až " + cz("%+.2f %%", 100*s.hi)
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %s | %s | %s |\n", src, s.n, s.nCLV, clv, ci, cz("%+.1f %%", 100*s.yield))
	}

	// H3: share of the t24h → close gap the hindsight goalie explains.
	b.WriteString("\n## H3: horní mez hodnoty brankáře\n\n")
	b.WriteString(h3(kinds))

	// Gate A1.
	b.WriteString("\n## Brána A1\n\nVrstva projde, když (a) na snímku opening, t24h nebo t1h zlepší log-loss trhu aspoň o 0,5 % a projde BH, nebo (b) sázky v otevíracím kurzu mají CLV > 0 s 95% intervalem nad nulou na ≥ 100 sázkách.\n\n| Vrstva | (a) log-loss | (b) CLV v open | Prošla |\n|---|---|---|---|\n")
	any := false
	for _, l := range m1bLayers {
		a := "ne"
		if ks := passLL[l]; len(ks) > 0 {
			a = "ano (" + strings.Join(ks, ", ") + ")"
		}
		c := yes(passCLV[l])
		ok := len(passLL[l]) > 0 || passCLV[l]
		any = any || ok
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", l, a, c, yes(ok))
	}
	if any {
		b.WriteString("\n**Verdikt A1: prošla aspoň jedna vrstva**; do F2 jdou jen vrstvy s „ano“.\n")
	} else {
		b.WriteString("\n**Verdikt A1: neprošla žádná vrstva.** Podle plánu 02 směr A končí.\n")
	}
	return b.String()
}

// anyBeta reports whether a variant ever moved off the market (some β ≠ 0).
func anyBeta(v *m1bVariant) bool {
	for _, bs := range v.betas {
		for _, b := range bs {
			if b != 0 {
				return true
			}
		}
	}
	return false
}

// pressureShares counts evaluated team-games in a race, and games where exactly
// one side is (x ≠ 0), to show how often the layer can act.
func pressureShares(in m1bInput, k *m1bKind) (sides, oneSide, games int) {
	c := features.PressureConfigFor(in.League)
	if c == nil {
		return 0, 0, 0
	}
	tl := features.PressureTimeline(in.Matches, *c)
	for _, o := range k.obs {
		games++
		st := tl[o.ID]
		if st[canon.Home] {
			sides++
		}
		if st[canon.Away] {
			sides++
		}
		if st[canon.Home] != st[canon.Away] {
			oneSide++
		}
	}
	return sides, oneSide, games
}

func yes(ok bool) string {
	if ok {
		return "**ano**"
	}
	return "ne"
}

func outcome(m canon.Match) int {
	switch {
	case m.Reg.Home > m.Reg.Away:
		return 0
	case m.Reg.Home == m.Reg.Away:
		return 1
	}
	return 2
}

func ids(pred map[int64][3]float64) []int64 {
	out := make([]int64, 0, len(pred))
	for id := range pred {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func nll(p [3]float64, y int) float64 { return -math.Log(math.Max(p[y], 1e-6)) }

func ll(k *m1bKind, v *m1bVariant, xs []int64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, id := range xs {
		s += nll(v.pred[id], k.byID[id].Y)
	}
	return s / float64(len(xs))
}

func pairedDelta(k *m1bKind, base, v *m1bVariant) backtest.Delta {
	var d []float64
	for _, id := range ids(v.pred) {
		pb, ok := base.pred[id]
		if !ok {
			continue
		}
		y := k.byID[id].Y
		d = append(d, nll(v.pred[id], y)-nll(pb, y))
	}
	return meanCI(d)
}

func meanCI(d []float64) backtest.Delta {
	n := float64(len(d))
	out := backtest.Delta{N: len(d), P: 1}
	if n < 2 {
		return out
	}
	m := 0.0
	for _, x := range d {
		m += x / n
	}
	v := 0.0
	for _, x := range d {
		v += (x - m) * (x - m)
	}
	se := math.Sqrt(v / (n - 1) / n)
	out.MeanDelta1X2, out.CILo, out.CIHi = m, m-1.96*se, m+1.96*se
	if se > 0 {
		out.P = math.Erfc(math.Abs(m/se) / math.Sqrt2)
	}
	return out
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func activeObs(k *m1bKind, v *m1bVariant) int {
	n := 0
	for id := range v.pred {
		for _, l := range v.layers {
			if k.byID[id].X[l] != 0 {
				n++
				break
			}
		}
	}
	return n
}

// sharpAt returns the sharp book's devigged 1X2 of one kind.
func sharpAt(quotes []canon.OddsQuote, kind, sharp string) ([3]float64, bool) {
	pr, ok := mp.PriorAt(quotes, kind, sharp)
	if !ok || pr.Source != sharp {
		return [3]float64{}, false
	}
	return pr.P, true
}

func moveCorr(in m1bInput, k *m1bKind, v *m1bVariant) (float64, int) {
	var xs, ys []float64
	for _, id := range ids(v.pred) {
		op, ok1 := sharpAt(in.Odds[id], "opening", in.Sharp)
		cl, ok2 := sharpAt(in.Odds[id], "closing", in.Sharp)
		if !ok1 || !ok2 {
			continue
		}
		xs = append(xs, v.pred[id][0]-op[0])
		ys = append(ys, cl[0]-op[0])
	}
	n := len(xs)
	if n < 3 {
		return 0, n
	}
	var mx, my float64
	for i := range xs {
		mx += xs[i] / float64(n)
		my += ys[i] / float64(n)
	}
	var sxy, sxx, syy float64
	for i := range xs {
		sxy += (xs[i] - mx) * (ys[i] - my)
		sxx += (xs[i] - mx) * (xs[i] - mx)
		syy += (ys[i] - my) * (ys[i] - my)
	}
	if sxx == 0 || syy == 0 {
		return 0, n
	}
	return sxy / math.Sqrt(sxx*syy), n
}

type betStats struct {
	n, nCLV     int
	clv, lo, hi float64
	yield       float64
}

func openBets(in m1bInput, k *m1bKind, v *m1bVariant) betStats {
	return openBetsWhere(in, k, v, func(int64) bool { return true })
}

func openBetsWhere(in m1bInput, k *m1bKind, v *m1bVariant, keep func(int64) bool) betStats {
	var clvs []float64
	var s betStats
	profit := 0.0
	for _, id := range ids(v.pred) {
		if !keep(id) {
			continue
		}
		p := v.pred[id]
		qs := in.Odds[id]
		bestEdge, price, sel := 0.0, 0.0, -1
		for _, book := range in.Books {
			prices := map[string]float64{}
			for _, q := range qs {
				if q.Book == book && q.Market == "1x2" && q.Kind == "opening" {
					prices[q.Selection] = q.Price
				}
			}
			imp, ok := mp.Shin1X2(prices)
			if !ok {
				continue
			}
			for i, name := range []string{"home", "draw", "away"} {
				pr := prices[name]
				e := p[i] - imp[i]
				if e < 0.03 || p[i]*pr <= 1 || pr > 10 || e <= bestEdge {
					continue
				}
				bestEdge, price, sel = e, pr, i
			}
		}
		if sel < 0 {
			continue
		}
		s.n++
		if k.byID[id].Y == sel {
			profit += price - 1
		} else {
			profit--
		}
		if cl, ok := sharpAt(qs, "closing", in.Sharp); ok {
			clvs = append(clvs, math.Log(price*cl[sel]))
		}
	}
	if s.n > 0 {
		s.yield = profit / float64(s.n)
	}
	d := meanCI(clvs)
	s.nCLV, s.clv, s.lo, s.hi = len(clvs), d.MeanDelta1X2, d.CILo, d.CIHi
	return s
}

func h3(kinds []*m1bKind) string {
	var t24, cl *m1bKind
	for _, k := range kinds {
		switch k.kind {
		case "t24h":
			t24 = k
		case "closing":
			cl = k
		}
	}
	var mkt, gl *m1bVariant
	for _, v := range t24.variants {
		switch v.label {
		case "trh sám":
			mkt = v
		case "goalie":
			gl = v
		}
	}
	var a, g, c float64
	n := 0
	for _, id := range ids(gl.pred) {
		pc, ok := cl.byID[id]
		pm, ok2 := mkt.pred[id]
		if !ok || !ok2 {
			continue
		}
		y := t24.byID[id].Y
		a += nll(pm, y)
		g += nll(gl.pred[id], y)
		c += nll(pc.Prior, y)
		n++
	}
	if n == 0 {
		return "Chybí: žádné zápasy s kurzem t24h i uzavíracím.\n"
	}
	a, g, c = a/float64(n), g/float64(n), c/float64(n)
	gap := a - c
	share := math.NaN()
	if gap > 0 {
		share = (a - g) / gap
	}
	ok := gap > 0 && share >= 0.5 && (a-g)/a >= 0.005
	var b strings.Builder
	fmt.Fprintf(&b, "Na %d zápasech s kurzem t24h i uzavíracím: log-loss t24h %s, t24h + skutečný brankář %s, uzavírací %s. ", n, cz("%.4f", a), cz("%.4f", g), cz("%.4f", c))
	if gap <= 0 {
		b.WriteString("Uzavírací kurz není lepší než t24h, mezera k vysvětlení neexistuje. ")
	} else {
		fmt.Fprintf(&b, "Mezera t24h → close je %s (%s); brankář z ní vysvětlí %s. ", cz("%.4f", gap), cz("%.2f %%", 100*gap/a), cz("%.0f %%", 100*share))
	}
	fmt.Fprintf(&b, "H3 (≥ 50 %% mezery a ≥ 0,5 %% absolutně): %s.\n", yes(ok))
	return b.String()
}
