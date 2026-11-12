// Package timing measures how CZ books follow the sharp market (plan 04 L6):
// sharp moves, the delay until a soft book moves the same way, and how long
// market-value-v1 hint windows stay open. Pure logic over canonical quotes.
package timing

import (
	"math"
	"sort"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/hint"
)

// Thresholds fixed in advance (plan 04 L6). They are not tuned.
const (
	MoveMin    = 0.02             // sharp move: Shin probability change of at least 2 pp
	MoveWindow = 30 * time.Minute // ... within 30 minutes
	ReactMin   = 0.01             // a soft book reacts: its Shin probability moves 1 pp the same way
)

// Fixture is one game's full pre-start 1X2 history over all books.
type Fixture struct {
	GameID int64
	Start  time.Time
	Quotes []canon.OddsQuote
}

// Reaction of one soft book to one sharp move.
type Reaction struct {
	Book    string
	Led     bool          // the book had already moved that way in the 30 min before
	Reacted bool          // moved the same way before the start
	Delay   time.Duration // event -> reaction
}

// Event is one sharp move.
type Event struct {
	At        time.Time
	Selection string
	Delta     float64 // sharp probability change over MoveWindow
	ToStart   time.Duration
	Reactions []Reaction
}

// Window is one continuous stretch in which the hint rule fires.
type Window struct {
	From, To  time.Time
	First     hint.Hint
	CLV       float64 // of the first hint, against the sharp close
	AfterMove bool    // opened within 2 h after a sharp move
	ToStart   time.Duration
	HasClose  bool
}

// Result of one fixture.
type Result struct {
	Events  []Event
	Windows []Window
}

type state struct {
	at time.Time
	p  map[string][3]float64 // fair 1X2 per book at this moment
	q  []canon.OddsQuote     // latest quote per book and selection
}

// states replays the history: one state per distinct capture time.
func states(f Fixture) []state {
	qs := make([]canon.OddsQuote, 0, len(f.Quotes))
	for _, q := range f.Quotes {
		if q.Market == "1x2" && q.CapturedAt.Before(f.Start) {
			qs = append(qs, q)
		}
	}
	sort.SliceStable(qs, func(i, j int) bool { return qs[i].CapturedAt.Before(qs[j].CapturedAt) })
	latest := map[string]canon.OddsQuote{}
	var out []state
	for i := 0; i < len(qs); {
		t := qs[i].CapturedAt
		for ; i < len(qs) && qs[i].CapturedAt.Equal(t); i++ {
			latest[qs[i].Book+"|"+qs[i].Selection] = qs[i]
		}
		cur := make([]canon.OddsQuote, 0, len(latest))
		for _, q := range latest {
			cur = append(cur, q)
		}
		st := state{at: t, p: map[string][3]float64{}, q: cur}
		for b, m := range hint.Books1X2(cur) {
			if p, ok := hint.Fair(m); ok {
				st.p[b] = p
			}
		}
		out = append(out, st)
	}
	return out
}

// at returns the last state at or before t (false when none).
func at(ss []state, t time.Time) (state, bool) {
	i := sort.Search(len(ss), func(i int) bool { return ss[i].at.After(t) })
	if i == 0 {
		return state{}, false
	}
	return ss[i-1], true
}

// Analyze finds sharp moves, soft-book reactions and hint windows.
func Analyze(f Fixture, r hint.Rule) Result {
	ss := states(f)
	var res Result
	if len(ss) == 0 {
		return res
	}
	var lastEvent time.Time
	for si, s := range ss {
		ps, ok := s.p[r.Sharp]
		if !ok || (!lastEvent.IsZero() && s.at.Sub(lastEvent) < MoveWindow) {
			continue
		}
		ref, ok := at(ss, s.at.Add(-MoveWindow))
		if !ok {
			continue
		}
		pr, ok := ref.p[r.Sharp]
		if !ok {
			continue
		}
		best, bi := 0.0, -1
		for _, i := range []int{0, 2} { // home and away; the draw follows them
			if d := ps[i] - pr[i]; math.Abs(d) > math.Abs(best) {
				best, bi = d, i
			}
		}
		if math.Abs(best) < MoveMin {
			continue
		}
		lastEvent = s.at
		ev := Event{At: s.at, Selection: hint.Selections[bi], Delta: best, ToStart: f.Start.Sub(s.at)}
		for _, b := range r.Soft {
			base, ok := s.p[b]
			if !ok {
				continue
			}
			rc := Reaction{Book: b}
			if prev, ok := ref.p[b]; ok && sameWay(base[bi]-prev[bi], best) {
				rc.Led = true
			}
			for _, later := range ss[si+1:] {
				if p, ok := later.p[b]; ok && sameWay(p[bi]-base[bi], best) {
					rc.Reacted, rc.Delay = true, later.at.Sub(s.at)
					break
				}
			}
			ev.Reactions = append(ev.Reactions, rc)
		}
		res.Events = append(res.Events, ev)
	}
	closing, hasClose := ss[len(ss)-1].p[r.Sharp]
	var open *Window
	for _, s := range ss {
		h := r.Find(s.q)
		switch {
		case h != nil && open == nil:
			w := Window{From: s.at, First: *h, ToStart: f.Start.Sub(s.at), HasClose: hasClose}
			if hasClose {
				w.CLV = hint.CLV(h.Price, closing[hint.Index(h.Selection)])
			}
			for _, e := range res.Events {
				if !s.at.Before(e.At) && s.at.Sub(e.At) <= 2*time.Hour {
					w.AfterMove = true
				}
			}
			open = &w
		case h == nil && open != nil:
			open.To = s.at
			res.Windows = append(res.Windows, *open)
			open = nil
		}
	}
	if open != nil {
		open.To = f.Start
		res.Windows = append(res.Windows, *open)
	}
	return res
}

func sameWay(d, dir float64) bool {
	return math.Abs(d) >= ReactMin && (d > 0) == (dir > 0)
}

// CatchProb is the chance that a poll every interval (random phase) sees a
// window of length d.
func CatchProb(d, interval time.Duration) float64 {
	if interval <= 0 {
		return 1
	}
	return math.Min(1, float64(d)/float64(interval))
}
