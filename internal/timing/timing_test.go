package timing

import (
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/hint"
)

func quotes(book string, t time.Time, h, d, a float64) []canon.OddsQuote {
	mk := func(sel string, p float64) canon.OddsQuote {
		return canon.OddsQuote{Book: book, Market: "1x2", Selection: sel, Price: p, CapturedAt: t}
	}
	return []canon.OddsQuote{mk("home", h), mk("draw", d), mk("away", a)}
}

func TestAnalyze(t *testing.T) {
	start := time.Date(2026, 3, 1, 18, 0, 0, 0, time.UTC)
	t0 := start.Add(-6 * time.Hour)
	var qs []canon.OddsQuote
	qs = append(qs, quotes("pinnacle", t0, 2.40, 4.20, 2.70)...)
	qs = append(qs, quotes("betano.cz", t0, 2.30, 4.00, 2.60)...)
	// Sharp move on home within 10 minutes; betano stays, so its home price is
	// now value; it follows 50 minutes later.
	qs = append(qs, quotes("pinnacle", t0.Add(40*time.Minute), 2.05, 4.30, 3.30)...)
	qs = append(qs, quotes("betano.cz", t0.Add(90*time.Minute), 1.95, 4.10, 3.20)...)
	// A quote after the start is ignored.
	qs = append(qs, quotes("pinnacle", start.Add(time.Minute), 1.50, 5.0, 6.0)...)

	r := hint.Rule{Sharp: "pinnacle", Soft: []string{"betano.cz"}, MinEdge: 0.03, MaxPrice: 10}
	res := Analyze(Fixture{GameID: 1, Start: start, Quotes: qs}, r)
	if len(res.Events) != 1 {
		t.Fatalf("events %+v", res.Events)
	}
	e := res.Events[0]
	if e.Selection != "home" || e.Delta < MoveMin || len(e.Reactions) != 1 {
		t.Fatalf("event %+v", e)
	}
	rc := e.Reactions[0]
	if !rc.Reacted || rc.Led || rc.Delay != 50*time.Minute {
		t.Fatalf("reaction %+v", rc)
	}
	if len(res.Windows) != 1 {
		t.Fatalf("windows %+v", res.Windows)
	}
	w := res.Windows[0]
	if w.First.Selection != "home" || !w.AfterMove || w.To.Sub(w.From) != 50*time.Minute || !w.HasClose {
		t.Fatalf("window %+v", w)
	}
	if w.CLV <= 0 {
		t.Fatalf("clv %v", w.CLV)
	}
	if p := CatchProb(50*time.Minute, 2*time.Hour); p < 0.41 || p > 0.42 {
		t.Fatalf("catch %v", p)
	}
}
