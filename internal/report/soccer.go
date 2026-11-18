package report

import (
	"fmt"
	"math"

	"github.com/lioilsources/BestBet/internal/devig"
)

// Gate Z1 of plan 05 S1.
const (
	MinSoccerJoin     = 0.99 // games with a football-data.co.uk row
	MinSoccerPinnacle = 0.95 // games with a closing Pinnacle 1X2
)

// MarketLogLoss is the mean log-loss of Shin-devigged closing prices against
// the 90-minute 1X2 result, per league and season.
type MarketLogLoss struct {
	Slug    string
	Season  int32
	N       int
	LogLoss float64
}

// SoccerLogLoss computes the closing-market baseline every soccer model is
// compared with (plan 05 S1).
func SoccerLogLoss(rows []RowClose) []MarketLogLoss {
	var out []MarketLogLoss
	for _, r := range rows {
		p, _, err := devig.Shin([]float64{r.PriceHome, r.PriceDraw, r.PriceAway})
		if err != nil {
			continue
		}
		i := 1
		switch {
		case r.Home60 > r.Away60:
			i = 0
		case r.Home60 < r.Away60:
			i = 2
		}
		if n := len(out); n == 0 || out[n-1].Slug != r.Slug || out[n-1].Season != r.Season {
			out = append(out, MarketLogLoss{Slug: r.Slug, Season: r.Season})
		}
		m := &out[len(out)-1]
		m.N++
		m.LogLoss += -math.Log(p[i])
	}
	for i := range out {
		out[i].LogLoss /= float64(out[i].N)
	}
	return out
}

// RowClose is one game's closing prices and 90-minute score (rows ordered by league, season).
type RowClose struct {
	Slug                            string
	Season                          int32
	Home60, Away60                  int32
	PriceHome, PriceDraw, PriceAway float64
}

func renderSoccer(w func(string, ...any), d Data) {
	if len(d.Soccer) == 0 {
		return
	}
	rows := make([]RowClose, len(d.SoccerClose))
	for i, r := range d.SoccerClose {
		rows[i] = RowClose{r.Slug, r.Season, r.Home60, r.Away60, r.PriceHome, r.PriceDraw, r.PriceAway}
	}
	ll := map[string]MarketLogLoss{}
	for _, m := range SoccerLogLoss(rows) {
		ll[fmt.Sprintf("%s/%d", m.Slug, m.Season)] = m
	}
	w("\n## Fotbal (plán 05 S1)\n\n")
	w("Zápasy z American Soccer Analysis, zavírací 1X2 z football-data.co.uk. Log-loss je zavírací Pinnacle (Shin) proti výsledku po 90 min: laťka pro každý pozdější model.\n\n")
	w("| Liga | Sezona | Zápasů | S poločasy | Play-off | Prodloužení | Penalty | S fd.co.uk | S Pinnaclem | Log-loss trhu | Z1 |\n|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, s := range d.Soccer {
		join := float64(s.WithFdcouk) / float64(s.Games)
		pin := float64(s.WithPinnacle) / float64(s.Games)
		gate := "ano"
		switch {
		case join < MinSoccerJoin:
			gate = "join " + pctNaN(join)
		case pin < MinSoccerPinnacle:
			gate = "Pinnacle " + pctNaN(pin)
		}
		l := "—"
		if m, ok := ll[fmt.Sprintf("%s/%d", s.Slug, s.Season)]; ok {
			l = czDecimal(fmt.Sprintf("%.4f", m.LogLoss))
		}
		w("| %s | %d | %d | %d | %d | %d | %d | %d | %d | %s | %s |\n",
			s.Slug, s.Season, s.Games, s.WithHalves, s.Knockout, s.ExtraTime, s.Penalties, s.WithFdcouk, s.WithPinnacle, l, gate)
	}
}
