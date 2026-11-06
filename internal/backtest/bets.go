package backtest

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/devig"
)

// BetConfig is the staking policy of plan 01 §2.3 (fixed, not tuned).
type BetConfig struct {
	Sharp     string   // reference book for the close (pinnacle)
	Books     []string // books we may bet at (CZ licensed)
	EdgeMin   float64  // p_model − p_implied threshold (0.03 for 1X2)
	Kelly     float64  // Kelly fraction (0.25)
	MaxStake  float64  // stake cap as share of bankroll (0.02)
	MaxPerDay int      // bets per day (5)
	BetBefore time.Duration
	StartBank float64
	MaxPrice  float64 // ignore longshots above this price (sanity, 10)
}

// DefaultBetConfig follows 01 §2.3. Bets are placed with the last price at
// least BetBefore (1 h) before the start, the close is the sharp book's last
// pre-start price.
func DefaultBetConfig() BetConfig {
	return BetConfig{Sharp: "pinnacle", Books: []string{"tipsport", "fortuna", "chance", "betano"}, EdgeMin: 0.03,
		Kelly: 0.25, MaxStake: 0.02, MaxPerDay: 5, BetBefore: time.Hour, StartBank: 1, MaxPrice: 10}
}

// Bet is one simulated 1X2 bet.
type Bet struct {
	Match     canon.Match
	Selection string
	Book      string
	Price     float64
	PModel    float64
	PImplied  float64 // Shin-devigged price of the bet book at bet time
	PClose    float64 // Shin-devigged sharp close, 0 if missing
	Stake     float64 // share of the bankroll at the start of the day
	Amount    float64 // stake in bankroll units
	Won       bool
	Profit    float64 // in bankroll units
}

// CLV is log(price taken / fair closing price); > 0 = beat the close.
func (b Bet) CLV() (float64, bool) {
	if b.PClose <= 0 {
		return 0, false
	}
	return math.Log(b.Price * b.PClose), true
}

var sels1x2 = []string{"home", "draw", "away"}

// lastBefore returns the latest price per selection of one book before t.
func lastBefore(quotes []canon.OddsQuote, book, market string, t time.Time) map[string]float64 {
	best := map[string]canon.OddsQuote{}
	for _, q := range quotes {
		if q.Book != book || q.Market != market || !q.CapturedAt.Before(t) {
			continue
		}
		if cur, ok := best[q.Selection]; !ok || q.CapturedAt.After(cur.CapturedAt) {
			best[q.Selection] = q
		}
	}
	out := map[string]float64{}
	for s, q := range best {
		out[s] = q.Price
	}
	return out
}

func shin1x2(prices map[string]float64) ([]float64, bool) {
	ps := make([]float64, 3)
	for i, s := range sels1x2 {
		p, ok := prices[s]
		if !ok || p <= 1 {
			return nil, false
		}
		ps[i] = p
	}
	q, _, err := devig.Shin(ps)
	return q, err == nil
}

// SimulateBets walks the rows in time order and places at most one 1X2 bet per
// game (the selection with the highest edge above the threshold), staking
// fractional Kelly of the bankroll at the start of the day.
func SimulateBets(rows []Row, odds map[int64][]canon.OddsQuote, cfg BetConfig) []Bet {
	rs := append([]Row(nil), rows...)
	sort.Slice(rs, func(i, j int) bool { return rs[i].Match.StartAt.Before(rs[j].Match.StartAt) })
	type cand struct {
		bet  Bet
		edge float64
	}
	byDay := map[string][]cand{}
	var days []string
	for _, r := range rs {
		qs := odds[r.Match.ID]
		if len(qs) == 0 {
			continue
		}
		var pClose []float64
		if cl := lastBefore(qs, cfg.Sharp, "1x2", r.Match.StartAt); len(cl) == 3 {
			pClose, _ = shin1x2(cl)
		}
		p := p1x2(r)
		var best *cand
		for _, book := range cfg.Books {
			prices := lastBefore(qs, book, "1x2", r.Match.StartAt.Add(-cfg.BetBefore))
			imp, ok := shin1x2(prices)
			if !ok {
				continue
			}
			for i, s := range sels1x2 {
				price := prices[s]
				edge := p[i] - imp[i]
				if edge < cfg.EdgeMin || p[i]*price <= 1 || price > cfg.MaxPrice {
					continue
				}
				if best != nil && edge <= best.edge {
					continue
				}
				b := Bet{Match: r.Match, Selection: s, Book: book, Price: price, PModel: p[i], PImplied: imp[i], Won: r.Outcome1X2 == i}
				if pClose != nil {
					b.PClose = pClose[i]
				}
				best = &cand{b, edge}
			}
		}
		if best == nil {
			continue
		}
		day := r.Match.StartAt.Format("2006-01-02")
		if _, ok := byDay[day]; !ok {
			days = append(days, day)
		}
		byDay[day] = append(byDay[day], *best)
	}
	bank := cfg.StartBank
	var out []Bet
	for _, d := range days {
		cs := byDay[d]
		sort.SliceStable(cs, func(i, j int) bool { return cs[i].edge > cs[j].edge })
		if len(cs) > cfg.MaxPerDay {
			cs = cs[:cfg.MaxPerDay]
		}
		dayBank := bank
		for _, c := range cs {
			b := c.bet
			f := cfg.Kelly * (b.PModel*b.Price - 1) / (b.Price - 1)
			b.Stake = math.Min(f, cfg.MaxStake)
			b.Amount = b.Stake * dayBank
			if b.Won {
				b.Profit = b.Amount * (b.Price - 1)
			} else {
				b.Profit = -b.Amount
			}
			bank += b.Profit
			out = append(out, b)
		}
	}
	return out
}

// BetSummary aggregates a set of bets.
type BetSummary struct {
	N         int
	Staked    float64
	Profit    float64
	Yield     float64 // profit / staked
	NCLV      int
	MeanCLV   float64
	CLVLo     float64
	CLVHi     float64
	EndBank   float64
	MaxDD     float64 // max drawdown of the bankroll, share of the peak
	BeatClose float64 // share of bets with CLV > 0
}

// SummariseBets computes yield, CLV with a normal 95 % interval and drawdown.
func SummariseBets(bets []Bet, startBank float64) BetSummary {
	s := BetSummary{N: len(bets), EndBank: startBank}
	var clv []float64
	peak, bank := startBank, startBank
	for _, b := range bets {
		bank += b.Profit
		peak = math.Max(peak, bank)
		if peak > 0 {
			s.MaxDD = math.Max(s.MaxDD, (peak-bank)/peak)
		}
		s.Profit += b.Profit
		s.Staked += b.Amount
		if c, ok := b.CLV(); ok {
			clv = append(clv, c)
			if c > 0 {
				s.BeatClose++
			}
		}
	}
	s.EndBank = bank
	if s.Staked > 0 {
		s.Yield = s.Profit / s.Staked
	}
	s.NCLV = len(clv)
	if n := float64(len(clv)); n >= 2 {
		for _, c := range clv {
			s.MeanCLV += c / n
		}
		v := 0.0
		for _, c := range clv {
			v += (c - s.MeanCLV) * (c - s.MeanCLV)
		}
		se := math.Sqrt(v / (n - 1) / n)
		s.CLVLo, s.CLVHi = s.MeanCLV-1.96*se, s.MeanCLV+1.96*se
		s.BeatClose /= n
	}
	return s
}

// MarketMove correlates the model's disagreement with the sharp opening price
// with the later open → close move (home win, 1X2): high correlation means the
// model anticipates what the market learns before kick-off (01 §5 fáze 2).
func MarketMove(rows []Row, odds map[int64][]canon.OddsQuote, sharp string) (corr float64, n int) {
	var xs, ys []float64
	for _, r := range rows {
		qs := odds[r.Match.ID]
		var first, last time.Time
		for _, q := range qs {
			if q.Book != sharp || q.Market != "1x2" || !q.CapturedAt.Before(r.Match.StartAt) {
				continue
			}
			if first.IsZero() || q.CapturedAt.Before(first) {
				first = q.CapturedAt
			}
			if q.CapturedAt.After(last) {
				last = q.CapturedAt
			}
		}
		if !first.Before(last) {
			continue // one capture: no move to measure
		}
		open, ok1 := shin1x2(lastBefore(qs, sharp, "1x2", first.Add(time.Second)))
		cl, ok2 := shin1x2(lastBefore(qs, sharp, "1x2", r.Match.StartAt))
		if !ok1 || !ok2 {
			continue
		}
		xs = append(xs, p1x2(r)[0]-open[0])
		ys = append(ys, cl[0]-open[0])
	}
	n = len(xs)
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

// GateInput collects what brána G1 (01 fáze 3) needs.
type GateInput struct {
	Rows    []Row
	Bets    []Bet
	Holdout int
	Sharp   string
	Odds    map[int64][]canon.OddsQuote
	// Ablation is the phase 2 verdict for the last row (text and pass).
	Ablation   string
	AblationOK bool
}

// RenderGate renders the G1 table; a criterion without data says "chybí".
func RenderGate(in GateInput) string {
	var b strings.Builder
	b.WriteString("| Kritérium | Práh | Hodnota | Splněno |\n|---|---|---|---|\n")
	var ho []Row
	for _, r := range in.Rows {
		if in.Holdout != 0 && r.Match.Season >= in.Holdout {
			ho = append(ho, r)
		}
	}
	// 1. log-loss vs devigged sharp close on held-out seasons.
	var mLL, sLL float64
	n := 0
	for _, r := range ho {
		cl, ok := shin1x2(lastBefore(in.Odds[r.Match.ID], in.Sharp, "1x2", r.Match.StartAt))
		if !ok {
			continue
		}
		mLL += -logc(p1x2(r)[r.Outcome1X2])
		sLL += -logc(cl[r.Outcome1X2])
		n++
	}
	if n == 0 {
		fmt.Fprintf(&b, "| Log-loss 1X2 vs. odmaržovaný %s, odložené sezony, ≥ 3 000 zápasů | model ≤ trh − 0,5 %% | chybí (žádné kurzy %s) | — |\n", in.Sharp, in.Sharp)
	} else {
		mLL /= float64(n)
		sLL /= float64(n)
		rel := (mLL - sLL) / sLL
		fmt.Fprintf(&b, "| Log-loss 1X2 vs. odmaržovaný %s, odložené sezony, ≥ 3 000 zápasů | model ≤ trh − 0,5 %% | model %s, trh %s (%s, n=%d) | %s |\n",
			in.Sharp, num(mLL), num(sLL), signedPct(rel), n, yes(rel <= -0.005 && n >= 3000))
	}
	// 2. CLV of edge bets.
	var hb []Bet
	for _, x := range in.Bets {
		if in.Holdout == 0 || x.Match.Season >= in.Holdout {
			hb = append(hb, x)
		}
	}
	s := SummariseBets(hb, 1)
	if s.NCLV == 0 {
		b.WriteString("| CLV sázek s hranou > 3 p. b. proti uzavíracímu kurzu | > 0, 95% interval nad nulou, ≥ 1 500 sázek | chybí (žádné sázky s uzavíracím kurzem) | — |\n")
	} else {
		fmt.Fprintf(&b, "| CLV sázek s hranou > 3 p. b. proti uzavíracímu kurzu | > 0, 95%% interval nad nulou, ≥ 1 500 sázek | %s (%s až %s), n=%d | %s |\n",
			signedPct(s.MeanCLV), signedPct(s.CLVLo), signedPct(s.CLVHi), s.NCLV, yes(s.CLVLo > 0 && s.NCLV >= 1500))
	}
	// 3. Yield at CZ prices per held-out season.
	bySeason := map[int][]Bet{}
	for _, x := range hb {
		bySeason[x.Match.Season] = append(bySeason[x.Match.Season], x)
	}
	if len(bySeason) == 0 {
		b.WriteString("| Yield při Kelly 1/4 u CZ kurzů | > 0 na obou odložených sezonách | chybí (žádné sázky) | — |\n")
	} else {
		var parts []string
		all := true
		var ss []int
		for k := range bySeason {
			ss = append(ss, k)
		}
		sortInts(ss)
		for _, k := range ss {
			y := SummariseBets(bySeason[k], 1)
			parts = append(parts, fmt.Sprintf("%d/%02d: %s (n=%d)", k, (k+1)%100, signedPct(y.Yield), y.N))
			all = all && y.Yield > 0
		}
		fmt.Fprintf(&b, "| Yield při Kelly 1/4 u CZ kurzů | > 0 na obou odložených sezonách | %s | %s |\n", strings.Join(parts, "; "), yes(all && len(ss) >= 2))
	}
	fmt.Fprintf(&b, "| Ablace: vlna + absence zlepšují log-loss proti baseline | ve všech ligách | %s | %s |\n", in.Ablation, yes(in.AblationOK))
	return b.String()
}

func yes(ok bool) string {
	if ok {
		return "**ano**"
	}
	return "ne"
}

func signedPct(f float64) string {
	return strings.Replace(fmt.Sprintf("%+.2f %%", 100*f), ".", ",", 1)
}
