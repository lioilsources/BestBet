package odds

import (
	"sort"
	"time"
)

// Point is one price of one selection at one time.
type Point struct {
	Key string // book|market|selection|line
	At  time.Time
	I   int // index into the caller's slice
}

// Thin keeps, per key, the prices the model and the G1 gate read: the opening
// price, the last at least 20 h before the start (t24h), the last at least 1 h
// before (t1h, the bet price) and the last before the start (closing). A full
// odds history is ~30 000 price changes per game; this keeps a few per
// selection. It returns index -> snapshot kind; a point serving several roles
// gets the latest one. With no start time only the first and last survive.
func Thin(points []Point, start time.Time) map[int]string {
	by := map[string][]Point{}
	for _, p := range points {
		by[p.Key] = append(by[p.Key], p)
	}
	out := map[int]string{}
	for _, ps := range by {
		sort.SliceStable(ps, func(a, b int) bool { return ps[a].At.Before(ps[b].At) })
		out[ps[0].I] = "opening"
		if start.IsZero() {
			out[ps[len(ps)-1].I] = "closing"
			continue
		}
		last := func(before time.Time) (Point, bool) {
			var got Point
			ok := false
			for _, p := range ps {
				if !p.At.Before(before) {
					break
				}
				got, ok = p, true
			}
			return got, ok
		}
		for _, r := range []struct {
			kind string
			d    time.Duration
		}{{"t24h", 20 * time.Hour}, {"t1h", time.Hour}, {"closing", 0}} {
			if p, ok := last(start.Add(-r.d)); ok {
				out[p.I] = r.kind
			}
		}
	}
	return out
}

// ThinAt is Thin for a history read at now: a t24h, t1h or closing price
// whose cut-off (start − 20 h, start − 1 h, start) is still in the future
// may be superseded, so it is marked "adhoc". A later read relabels it.
func ThinAt(points []Point, start, now time.Time) map[int]string {
	out := Thin(points, start)
	if start.IsZero() {
		return out
	}
	cut := map[string]time.Duration{"t24h": 20 * time.Hour, "t1h": time.Hour, "closing": 0}
	for i, k := range out {
		if d, ok := cut[k]; ok && start.Add(-d).After(now) {
			out[i] = "adhoc"
		}
	}
	return out
}
