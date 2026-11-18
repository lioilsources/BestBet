package report

import (
	"math"
	"testing"
)

func TestSoccerLogLoss(t *testing.T) {
	// Fair prices (no margin): Shin returns the implied probabilities.
	rows := []RowClose{
		{Slug: "usa-mls", Season: 2024, Home60: 2, Away60: 1, PriceHome: 2, PriceDraw: 4, PriceAway: 4},
		{Slug: "usa-mls", Season: 2024, Home60: 1, Away60: 1, PriceHome: 2, PriceDraw: 4, PriceAway: 4},
		{Slug: "usa-mls", Season: 2025, Home60: 0, Away60: 3, PriceHome: 2, PriceDraw: 4, PriceAway: 4},
	}
	got := SoccerLogLoss(rows)
	if len(got) != 2 || got[0].N != 2 || got[1].N != 1 {
		t.Fatalf("groups: %+v", got)
	}
	want := (math.Log(2) + math.Log(4)) / 2
	if math.Abs(got[0].LogLoss-want) > 1e-6 || math.Abs(got[1].LogLoss-math.Log(4)) > 1e-6 {
		t.Fatalf("log-loss: %+v want %.4f", got, want)
	}
}
