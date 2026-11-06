package edge

import (
	"math"
	"testing"

	"github.com/lioilsources/BestBet/internal/canon"
)

func TestOffers(t *testing.T) {
	p := canon.MarketProbs{
		"1x2":   {"home": 0.50, "draw": 0.22, "away": 0.28},
		"total": {"over@5.5": 0.45, "under@5.5": 0.55},
	}
	l := 5.5
	q := []canon.OddsQuote{
		{Book: "cz", Market: "1x2", Selection: "home", Price: 2.30},
		{Book: "cz", Market: "1x2", Selection: "draw", Price: 4.00},
		{Book: "cz", Market: "1x2", Selection: "away", Price: 3.00},
		{Book: "cz", Market: "total", Selection: "over", Line: &l, Price: 2.0},
		{Book: "offshore", Market: "1x2", Selection: "home", Price: 3.0}, // not allowed: never shown
	}
	pol := DefaultPolicy()
	offers := Offers(p, q, map[string]bool{"cz": true}, pol)
	if len(offers) != 4 {
		t.Fatalf("%d offers: %+v", len(offers), offers)
	}
	for _, o := range offers {
		if o.Book != "cz" {
			t.Fatalf("unlicensed book shown: %+v", o)
		}
		if o.Green {
			t.Fatalf("no CLV-proven market: nothing may be green: %+v", o)
		}
	}
	home := offers[0] // highest log growth first
	if home.Selection != "home" || home.EV <= 0 || home.Stake <= 0 || home.Stake > pol.MaxStake || home.Why != "trh bez prokázaného CLV" {
		t.Fatalf("home: %+v", home)
	}
	// Shin implied probabilities of a complete market sum to one.
	sum := 0.0
	for _, o := range offers {
		if o.Market == "1x2" {
			sum += o.PImplied
		}
	}
	if math.Abs(sum-1) > 1e-6 {
		t.Fatalf("implied sum %.4f", sum)
	}
	// Incomplete total market: implied = 1/price.
	for _, o := range offers {
		if o.Market == "total" && o.PImplied != 0.5 {
			t.Fatalf("total: %+v", o)
		}
	}
	pol.CLVMarkets["1x2"] = true
	if !Offers(p, q, map[string]bool{"cz": true}, pol)[0].Green {
		t.Fatal("1x2 with CLV and edge must be green")
	}
}
