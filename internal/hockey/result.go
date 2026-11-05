// Package hockey holds source-independent game logic: regulation vs final score,
// OT/SO split, phase detection and play-off series state.
package hockey

import (
	"fmt"
	"strconv"
	"strings"
)

// Score is one "home-away" pair.
type Score struct{ Home, Away int }

// Result is a game outcome normalised so that 1X2 (60 min) and ML (incl. OT/SO)
// can both be settled from it.
type Result struct {
	Periods   []Score // regulation periods 1..3
	OT        *Score  // nil when the game ended in regulation
	SOWinner  string  // "home" | "away" | ""
	Final     Score   // incl. OT and the single shootout winner goal
	Reg       Score   // 60 min
	DecidedIn string  // reg | ot | so
}

// ParseScore parses "2-1" or "2 - 1".
func ParseScore(s string) (Score, error) {
	parts := strings.Split(strings.ReplaceAll(s, " ", ""), "-")
	if len(parts) != 2 {
		return Score{}, fmt.Errorf("score %q: want h-a", s)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return Score{}, fmt.Errorf("score %q: %w", s, err)
	}
	a, err := strconv.Atoi(parts[1])
	if err != nil {
		return Score{}, fmt.Errorf("score %q: %w", s, err)
	}
	return Score{h, a}, nil
}

// BuildResult derives the normalised result. periods holds 3 regulation periods;
// ot and so are optional (nil = not played). so is the shootout tally ("3-2")
// or the single-goal form used by some feeds ("0-1"); only its sign matters.
func BuildResult(periods []Score, ot, so *Score) (Result, error) {
	if len(periods) != 3 {
		return Result{}, fmt.Errorf("want 3 regulation periods, got %d", len(periods))
	}
	r := Result{Periods: periods, DecidedIn: "reg"}
	for _, p := range periods {
		r.Reg.Home += p.Home
		r.Reg.Away += p.Away
	}
	r.Final = r.Reg
	if r.Reg.Home != r.Reg.Away {
		if (ot != nil && ot.Home+ot.Away > 0) || so != nil {
			return Result{}, fmt.Errorf("overtime after decided regulation %d-%d", r.Reg.Home, r.Reg.Away)
		}
		return r, nil
	}
	if ot != nil {
		o := *ot
		r.OT = &o
		r.Final.Home += o.Home
		r.Final.Away += o.Away
		if o.Home != o.Away {
			r.DecidedIn = "ot"
			return r, nil
		}
	}
	if so == nil {
		// Tied after regulation and OT without a shootout: a regular-season tie
		// (some leagues) or incomplete data.
		r.DecidedIn = "reg"
		return r, nil
	}
	if r.OT == nil {
		r.OT = &Score{}
	}
	switch {
	case so.Home > so.Away:
		r.SOWinner = "home"
		r.Final.Home++
	case so.Away > so.Home:
		r.SOWinner = "away"
		r.Final.Away++
	default:
		return Result{}, fmt.Errorf("shootout without winner %d-%d", so.Home, so.Away)
	}
	r.DecidedIn = "so"
	return r, nil
}

// OTFormat is the extra-time format a league plays in a phase.
type OTFormat struct {
	Regular string `yaml:"regular"` // e.g. 3v3_5 (+ shootout)
	Playoff string `yaml:"playoff"` // e.g. 5v5_20 (repeated until a goal)
}

// OTType maps the decided result to the games.ot_type value.
func OTType(r Result, phase string, f OTFormat) string {
	switch r.DecidedIn {
	case "reg":
		return "none"
	case "so":
		return "shootout"
	}
	format := f.Regular
	if phase != "regular" && phase != "pre_playoff" {
		format = f.Playoff
	}
	if format == "" {
		return "unknown"
	}
	return format
}
