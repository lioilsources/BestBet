package hint

import (
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

func quotes(book string, h, d, a float64, at time.Time) []canon.OddsQuote {
	return []canon.OddsQuote{
		{Book: book, Market: "1x2", Selection: "home", Price: h, CapturedAt: at},
		{Book: book, Market: "1x2", Selection: "draw", Price: d, CapturedAt: at},
		{Book: book, Market: "1x2", Selection: "away", Price: a, CapturedAt: at},
	}
}

func TestFind(t *testing.T) {
	at := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	sharp := quotes("pinnacle", 2.0, 4.2, 3.3, at)
	// Same view: no hint.
	if h := Default().Find(append(sharp, quotes("betano.cz", 1.95, 4.1, 3.2, at)...)); h != nil {
		t.Fatalf("unexpected %+v", h)
	}
	// Soft book is slow on the away side: a hint on away at betano.
	qs := append(sharp, quotes("betano.cz", 1.75, 4.1, 4.2, at)...)
	h := Default().Find(qs)
	if h == nil || h.Book != "betano.cz" || h.Selection != "away" || h.EV <= 0 || h.Edge < 0.03 {
		t.Fatalf("%+v", h)
	}
	// Without the sharp book there is nothing to compare to.
	if h := Default().Find(quotes("betano.cz", 1.75, 4.1, 4.2, at)); h != nil {
		t.Fatalf("no sharp: %+v", h)
	}
	if c := CLV(2.0, 0.5); c != 0 {
		t.Fatalf("clv %v", c)
	}
}
