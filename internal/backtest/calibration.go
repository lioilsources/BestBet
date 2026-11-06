package backtest

import (
	"fmt"
	"math"
	"strings"

	"github.com/lioilsources/BestBet/internal/model/calib"
)

// binaryEvent is one market selection as a yes/no event of a row.
type binaryEvent struct {
	Market, Label string
	P             func(Row) (float64, bool)
	Y             func(Row) bool
}

func mkt(market, key string) func(Row) (float64, bool) {
	return func(r Row) (float64, bool) {
		p, ok := r.Pred.Markets[market][key]
		return p, ok
	}
}

// calibEvents are the markets calibrated per selection (01 fáze 3 "kalibrace per trh").
var calibEvents = []binaryEvent{
	{"1x2", "domácí", mkt("1x2", "home"), func(r Row) bool { return r.Outcome1X2 == 0 }},
	{"1x2", "remíza", mkt("1x2", "draw"), func(r Row) bool { return r.Outcome1X2 == 1 }},
	{"1x2", "hosté", mkt("1x2", "away"), func(r Row) bool { return r.Outcome1X2 == 2 }},
	{"ml", "domácí", mkt("ml", "home"), func(r Row) bool { return r.HomeWonML }},
	{"total", "over 5,5", mkt("total", "over@5.5"), func(r Row) bool { return r.Match.Reg.Home+r.Match.Reg.Away > 5 }},
	{"puck_line", "domácí −1,5", mkt("puck_line", "home@-1.5"), func(r Row) bool { return r.Match.Reg.Home-r.Match.Reg.Away >= 2 }},
	{"p1_1x2", "remíza ve 1. třetině", mkt("p1_1x2", "draw"), func(r Row) bool {
		return len(r.Match.Periods) > 0 && r.Match.Periods[0].Home == r.Match.Periods[0].Away
	}},
	{"p1_total", "over 1,5 ve 1. třetině", mkt("p1_total", "over@1.5"), func(r Row) bool {
		return len(r.Match.Periods) > 0 && r.Match.Periods[0].Home+r.Match.Periods[0].Away > 1
	}},
}

func binLL(p float64, y bool) float64 {
	p = math.Min(math.Max(p, 1e-6), 1-1e-6)
	if y {
		return -math.Log(p)
	}
	return -math.Log(1 - p)
}

// RenderCalibration fits Platt and isotonic maps per market selection on the
// tuning season and reports log-loss on the held-out seasons: raw, Platt,
// isotonic. Calibration is never fitted on a held-out season.
func RenderCalibration(rows []Row, tune, holdout int) string {
	var fit, eval []Row
	for _, r := range rows {
		switch {
		case r.Match.Season == tune:
			fit = append(fit, r)
		case holdout != 0 && r.Match.Season >= holdout:
			eval = append(eval, r)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Kalibrace fitovaná na sezoně %d/%02d (%d zápasů), hodnocená na odložených sezonách (%d zápasů). Binární log-loss, nižší je lepší. Platt má 2 parametry, izotonická regrese je stupňovitá a na malém vzorku přefituje.\n\n",
		tune, (tune+1)%100, len(fit), len(eval))
	if len(fit) < 100 || len(eval) == 0 {
		b.WriteString("Málo dat pro kalibraci.\n")
		return b.String()
	}
	b.WriteString("| Trh | Výběr | Četnost (odložené) | Průměr modelu | Log-loss surový | Platt | Izotonická |\n|---|---|---|---|---|---|---|\n")
	for _, ev := range calibEvents {
		var ps []float64
		var ys []bool
		for _, r := range fit {
			if p, ok := ev.P(r); ok {
				ps, ys = append(ps, p), append(ys, ev.Y(r))
			}
		}
		if len(ps) < 100 {
			continue
		}
		pl, errP := calib.FitPlatt(ps, ys)
		iso, errI := calib.FitIsotonic(ps, ys, nil)
		var raw, llP, llI, rate, mean float64
		n := 0
		for _, r := range eval {
			p, ok := ev.P(r)
			if !ok {
				continue
			}
			y := ev.Y(r)
			n++
			raw += binLL(p, y)
			if errP == nil {
				llP += binLL(pl.Apply(p), y)
			}
			if errI == nil {
				llI += binLL(iso.Apply(p), y)
			}
			if y {
				rate++
			}
			mean += p
		}
		if n == 0 {
			continue
		}
		f := float64(n)
		cell := func(v float64, err error) string {
			if err != nil {
				return "—"
			}
			s := num(v / f)
			if v < raw {
				s = "**" + s + "**"
			}
			return s
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n", ev.Market, ev.Label, pct(rate/f), pct(mean/f), num(raw/f), cell(llP, errP), cell(llI, errI))
	}
	b.WriteString("\nTučně = lepší než surový model. Spolehlivost výhry domácích (1X2) na odložených sezonách, surový model:\n\n")
	var ps []float64
	var ys []bool
	for _, r := range eval {
		ps, ys = append(ps, p1x2(r)[0]), append(ys, r.Outcome1X2 == 0)
	}
	b.WriteString("| Koš | Zápasů | Model | Skutečnost |\n|---|---|---|---|\n")
	for _, bin := range calib.Reliability(ps, ys, 5) {
		if bin.N == 0 {
			continue
		}
		fmt.Fprintf(&b, "| %s–%s | %d | %s | %s |\n", pct(bin.Lo), pct(bin.Hi), bin.N, pct(bin.MeanP), pct(bin.Observed))
	}
	return b.String()
}
