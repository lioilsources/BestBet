// Package teammatch maps a source-specific team name onto a canonical team.
// Order: explicit alias -> exact normalised name -> unique fuzzy token match.
package teammatch

import (
	"strings"
	"unicode"

	"github.com/lioilsources/BetWave/internal/hockey"
)

// Club-form tokens that carry no identity ("IK", "HC", "Hockey" ...).
var stopTokens = map[string]bool{
	"hc": true, "hk": true, "ik": true, "if": true, "bk": true, "sk": true, "ihc": true,
	"hockey": true, "club": true, "ice": true, "team": true, "ishockey": true, "ishockeyklubb": true,
	"fk": true, "ec": true, "ehc": true, "sc": true, "ev": true, "ab": true, "as": true, "ok": true,
	"the": true, "of": true, "mhk": true, "hkm": true,
}

// Norm lower-cases, folds diacritics and keeps letters, digits and single spaces.
func Norm(name string) string {
	s := strings.ToLower(hockey.Fold(name))
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			space = false
			continue
		}
		if !space && b.Len() > 0 {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}

// Tokens returns identity tokens of a normalised name.
func Tokens(norm string) []string {
	var out []string
	for _, t := range strings.Fields(norm) {
		if !stopTokens[t] {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return strings.Fields(norm)
	}
	return out
}

// Team is a canonical team.
type Team struct {
	ID   int32
	Name string
	Norm string
}

// Matcher resolves names within one league.
type Matcher struct {
	teams   []Team
	aliases map[string]int32 // alias_norm -> team id
}

// New builds a matcher from canonical teams and alias_norm -> team id.
func New(teams []Team, aliases map[string]int32) *Matcher {
	if aliases == nil {
		aliases = map[string]int32{}
	}
	return &Matcher{teams: teams, aliases: aliases}
}

// Method tells how a match was made.
type Method string

const (
	ByAlias Method = "alias"
	ByExact Method = "exact"
	ByFuzzy Method = "fuzzy"
	NoMatch Method = "none"
)

// Match returns the team id for name. Fuzzy matches must be unique; an ambiguous
// name ("Djurgården" vs "Djurgården J20") returns NoMatch so a human adds an alias.
func (m *Matcher) Match(name string) (int32, Method) {
	n := Norm(name)
	if id, ok := m.aliases[n]; ok {
		return id, ByAlias
	}
	for _, t := range m.teams {
		if t.Norm == n {
			return t.ID, ByExact
		}
	}
	best, bestScore, tie := int32(0), 0, false
	for _, t := range m.teams {
		s := score(Tokens(n), Tokens(t.Norm))
		switch {
		case s > bestScore:
			best, bestScore, tie = t.ID, s, false
		case s == bestScore && s > 0:
			tie = true
		}
	}
	if bestScore == 0 || tie {
		return 0, NoMatch
	}
	return best, ByFuzzy
}

// Add registers a new canonical team (after it was inserted).
func (m *Matcher) Add(t Team) { m.teams = append(m.teams, t) }

// score counts tokens of a that match some token of b, where two tokens match when
// equal or one is a prefix (>= 5 chars) of the other ("ostersunds" ~ "ostersund").
// Every identity token of the shorter name must match, otherwise the score is 0.
func score(a, b []string) int {
	if len(a) > len(b) {
		a, b = b, a
	}
	n := 0
	for _, x := range a {
		hit := false
		for _, y := range b {
			if tokenMatch(x, y) {
				hit = true
				break
			}
		}
		if !hit {
			return 0
		}
		n++
	}
	return n
}

func tokenMatch(x, y string) bool {
	if x == y {
		return true
	}
	if len(x) > len(y) {
		x, y = y, x
	}
	return len(x) >= 5 && strings.HasPrefix(y, x)
}
