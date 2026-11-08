package marketprior

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

func TestAdjustZeroIsPrior(t *testing.T) {
	p0 := [3]float64{0.45, 0.22, 0.33}
	if p := Adjust(p0, 0); math.Abs(p[0]-p0[0]) > 1e-12 || math.Abs(p[2]-p0[2]) > 1e-12 {
		t.Fatalf("Adjust(0) changed the prior: %v", p)
	}
	if p := Adjust(p0, 0.2); p[0] <= p0[0] || p[2] >= p0[2] {
		t.Fatalf("positive shift must favour home: %v", p)
	}
}

// A feature with a true effect is recovered; a noise feature stays near zero.
func TestFitRecoversBeta(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	p0 := [3]float64{0.42, 0.23, 0.35}
	var obs []Obs
	for i := 0; i < 20000; i++ {
		x := r.NormFloat64()
		p := Adjust(p0, 0.3*x)
		u, y := r.Float64(), 2
		if u < p[0] {
			y = 0
		} else if u < p[0]+p[1] {
			y = 1
		}
		obs = append(obs, Obs{ID: int64(i), Prior: p0, X: map[string]float64{"sig": x, "noise": r.NormFloat64()}, Y: y})
	}
	m := Fit(obs, []string{"sig", "noise"}, Config{})
	if math.Abs(m.Beta["sig"]-0.3) > 0.04 || math.Abs(m.Beta["noise"]) > 0.04 {
		t.Fatalf("beta = %v", m.Beta)
	}
}

func TestPriorAtFallsBackToMedian(t *testing.T) {
	q := func(book, sel string, price float64) canon.OddsQuote {
		return canon.OddsQuote{Book: book, Market: "1x2", Selection: sel, Price: price, Kind: "t24h", CapturedAt: time.Now()}
	}
	quotes := []canon.OddsQuote{
		q("a", "home", 2.0), q("a", "draw", 4.0), q("a", "away", 3.2),
		q("b", "home", 2.2), q("b", "draw", 4.2), q("b", "away", 3.0),
		q("pinnacle", "home", 2.1), // incomplete: not used
	}
	p, ok := PriorAt(quotes, "t24h", "pinnacle")
	if !ok || p.Source != "median" || math.Abs(p.P[0]+p.P[1]+p.P[2]-1) > 1e-9 {
		t.Fatalf("prior = %+v ok=%v", p, ok)
	}
	if _, ok := PriorAt(quotes, "opening", "pinnacle"); ok {
		t.Fatal("no opening quotes: expected no prior")
	}
}
