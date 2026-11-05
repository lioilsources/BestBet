// Package swehockey parses public pages of stats.swehockey.se: the season schedule
// (results with periods) and the per-game events page (goalkeeper summary, GK in/out).
package swehockey

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/lioilsources/BetWave/internal/hockey"
)

// ScheduleGame is one row of /ScheduleAndResults/Schedule/{id}.
type ScheduleGame struct {
	GameID    int    // events id, or the match code when HasEvents is false
	HasEvents bool   // a /Game/Events page exists (goalies available)
	Label     string // play-off series label, e.g. "Åttondelsfinal 1"
	StartAt   time.Time
	Home      string
	Away      string
	Played    bool
	Final     hockey.Score   // as printed, incl. the shootout winner goal
	Periods   []hockey.Score // as printed: 3 periods, then OT, then SO
	Result    *hockey.Result // nil when not played or not parseable
	ParseErr  string
}

var (
	reGameID    = regexp.MustCompile(`/Game/Events/(\d+)`)
	reMatchCode = regexp.MustCompile(`lnkTooltip\s+(\d{6,})\s`)
	reDate      = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	reTime      = regexp.MustCompile(`\b\d{2}:\d{2}\b`)
	reScore     = regexp.MustCompile(`^(\d+)\s*-\s*(\d+)$`)
	rePeriods   = regexp.MustCompile(`\(([^)]*)\)`)
)

// ParseSchedule reads all game rows of both layouts:
//   - regular season: Date | Time | Game | Result (link to /Game/Events/{id}) | Periods | ...
//   - play-off:       Round | Date Time | Game + <i>series label</i> | Result (no link) | Periods | ...
//
// Dates are printed only on the first game of a day, so the last seen date carries
// over. Rows without an events link get the match code (span title) as GameID and
// HasEvents=false: their goalies cannot be fetched. loc is the league time zone;
// shootouts tells whether a 5th score is a shootout (regular season) or another OT.
func ParseSchedule(page []byte, loc *time.Location, shootouts bool) ([]ScheduleGame, error) {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return nil, fmt.Errorf("parse schedule: %w", err)
	}
	var out []ScheduleGame
	date := ""
	for _, tr := range findAll(doc, "tr") {
		tds := children(tr, "td")
		if len(tds) < 5 {
			continue
		}
		g := ScheduleGame{}
		teamsCell, scoreCell, periodsCell, clock := -1, -1, -1, ""
		for i, td := range tds {
			c := text(td)
			if d := reDate.FindString(c); d != "" {
				date = d
			}
			if clock == "" {
				clock = reTime.FindString(c)
			}
			switch {
			case reScore.MatchString(c) && scoreCell < 0:
				scoreCell = i
			case teamsCell < 0 && strings.Contains(textExcept(td, "i"), " - ") && !reDate.MatchString(c):
				teamsCell = i
			case periodsCell < 0 && strings.HasPrefix(c, "(") && rePeriods.MatchString(c):
				periodsCell = i
			}
		}
		if teamsCell < 0 {
			continue
		}
		raw := renderAttrs(tr)
		if m := reGameID.FindStringSubmatch(raw); m != nil {
			g.GameID, _ = strconv.Atoi(m[1])
			g.HasEvents = true
		} else if m := reMatchCode.FindStringSubmatch(raw); m != nil {
			g.GameID, _ = strconv.Atoi(m[1])
		} else {
			continue
		}
		if date == "" {
			return nil, fmt.Errorf("game %d: no date before first row", g.GameID)
		}
		if clock == "" {
			clock = "00:00"
		}
		g.StartAt, err = time.ParseInLocation("2006-01-02 15:04", date+" "+clock, loc)
		if err != nil {
			return nil, fmt.Errorf("game %d: time: %w", g.GameID, err)
		}
		g.Home, g.Away, err = splitTeams(textExcept(tds[teamsCell], "i"))
		if err != nil {
			return nil, fmt.Errorf("game %d: %w", g.GameID, err)
		}
		for _, n := range findAll(tds[teamsCell], "i") {
			g.Label = text(n)
		}
		if scoreCell >= 0 {
			sm := reScore.FindStringSubmatch(text(tds[scoreCell]))
			g.Played = true
			g.Final.Home, _ = strconv.Atoi(sm[1])
			g.Final.Away, _ = strconv.Atoi(sm[2])
			if periodsCell >= 0 {
				pm := rePeriods.FindStringSubmatch(text(tds[periodsCell]))
				for _, p := range strings.Split(pm[1], ",") {
					s, err := hockey.ParseScore(strings.TrimSpace(p))
					if err != nil {
						g.ParseErr = err.Error()
						break
					}
					g.Periods = append(g.Periods, s)
				}
			}
			if g.ParseErr == "" {
				r, err := resultFromPeriods(g.Periods, g.Final, shootouts)
				if err != nil {
					g.ParseErr = err.Error()
				} else {
					g.Result = &r
				}
			}
		}
		out = append(out, g)
	}
	return out, nil
}

// resultFromPeriods maps swehockey's period list. With shootouts (regular season):
// 3 = regulation, 4 = + OT, 5 = + OT (0-0) + shootout. Without (play-off): every
// entry after the third is another 20-minute OT period and they are summed.
// The printed final must agree with what we derive.
func resultFromPeriods(p []hockey.Score, final hockey.Score, shootouts bool) (hockey.Result, error) {
	if len(p) < 3 || (shootouts && len(p) > 5) {
		return hockey.Result{}, fmt.Errorf("unexpected %d period scores", len(p))
	}
	var ot, so *hockey.Score
	if shootouts {
		if len(p) >= 4 {
			ot = &p[3]
		}
		if len(p) == 5 {
			so = &p[4]
		}
	} else if len(p) > 3 {
		sum := hockey.Score{}
		for _, x := range p[3:] {
			sum.Home += x.Home
			sum.Away += x.Away
		}
		ot = &sum
	}
	r, err := hockey.BuildResult(p[:3], ot, so)
	if err != nil {
		return r, err
	}
	if r.Final != final {
		return r, fmt.Errorf("derived final %d-%d != printed %d-%d", r.Final.Home, r.Final.Away, final.Home, final.Away)
	}
	return r, nil
}

func splitTeams(s string) (string, string, error) {
	parts := strings.SplitN(s, " - ", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("teams %q: want 'Home - Away'", s)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

// Goalie is one goalkeeper line of a game.
type Goalie struct {
	Side         string // home | away
	Jersey       int
	Name         string // "First Last"
	Saves        int
	ShotsAgainst int
	GoalsAgainst int
	SavePct      float64 // 0..1
	Starter      bool
	TOISeconds   int // from GK In/Out events; 0 when unknown
}

// GameEvents is what we take from /Game/Events/{id}.
type GameEvents struct {
	HomeAbbr, AwayAbbr string
	HomeName, AwayName string
	Goalies            []Goalie
}

var (
	reTitle   = regexp.MustCompile(`^(.+?) - (.+?) \(\d+-\d+\)$`)
	rePlayer  = regexp.MustCompile(`^(\d+)\.\s*(.+)$`)
	reSaves   = regexp.MustCompile(`^\((\d+)/(\d+)\)$`)
	rePct     = regexp.MustCompile(`^(\d+)[,.](\d+)%$`)
	reClock   = regexp.MustCompile(`^(\d{1,3}):(\d{2})$`)
	rePeriodH = regexp.MustCompile(`(?i)^(1st|2nd|3rd|\d+th) period$|^(?i)overtime|^(?i)shootout|^(?i)game winning shots`)
)

// ParseEvents extracts teams and goalkeepers. The starter is the goalie with
// "GK In" at 00:00; TOI comes from paired GK In/Out events (events are listed
// newest first, so they are sorted by game clock).
func ParseEvents(page []byte) (GameEvents, error) {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return GameEvents{}, fmt.Errorf("parse events: %w", err)
	}
	toks := tokens(doc)
	var ev GameEvents
	for i, t := range toks {
		if m := reTitle.FindStringSubmatch(t); m != nil && ev.HomeAbbr == "" {
			ev.HomeAbbr, ev.AwayAbbr = m[1], m[2]
			if i+1 < len(toks) {
				ev.HomeName, ev.AwayName, _ = splitTeams(toks[i+1])
			}
			break
		}
	}
	if ev.HomeAbbr == "" {
		return ev, fmt.Errorf("events: title 'HOME - AWAY (h-a)' not found")
	}
	side := func(abbr string) string {
		switch abbr {
		case ev.HomeAbbr:
			return "home"
		case ev.AwayAbbr:
			return "away"
		}
		return ""
	}

	start := indexOf(toks, "Goalkeeper Summary")
	if start < 0 {
		return ev, fmt.Errorf("events: no goalkeeper summary")
	}
	byKey := map[string]int{} // side|jersey -> index in ev.Goalies
	i := start + 1
	for i+3 < len(toks) && !rePeriodH.MatchString(toks[i]) {
		s := side(toks[i])
		pm := rePlayer.FindStringSubmatch(toks[i+1])
		pct := rePct.FindStringSubmatch(toks[i+2])
		sv := reSaves.FindStringSubmatch(toks[i+3])
		if s == "" || pm == nil || pct == nil || sv == nil {
			i++
			continue
		}
		g := Goalie{Side: s, Name: flipName(pm[2])}
		g.Jersey, _ = strconv.Atoi(pm[1])
		g.Saves, _ = strconv.Atoi(sv[1])
		g.ShotsAgainst, _ = strconv.Atoi(sv[2])
		g.GoalsAgainst = g.ShotsAgainst - g.Saves
		if g.ShotsAgainst > 0 {
			g.SavePct = float64(g.Saves) / float64(g.ShotsAgainst)
		}
		byKey[fmt.Sprintf("%s|%d", s, g.Jersey)] = len(ev.Goalies)
		ev.Goalies = append(ev.Goalies, g)
		i += 4
	}
	if len(ev.Goalies) == 0 {
		return ev, fmt.Errorf("events: goalkeeper summary is empty")
	}

	type gk struct {
		at  int
		in  bool
		key string
	}
	var events []gk
	end := 60 * 60
	for j := 1; j+2 < len(toks); j++ {
		if toks[j] != "GK In" && toks[j] != "GK Out" {
			continue
		}
		at, ok := clock(toks[j-1])
		if !ok {
			continue
		}
		s := side(toks[j+1])
		pm := rePlayer.FindStringSubmatch(toks[j+2])
		if s == "" || pm == nil {
			continue
		}
		jersey, _ := strconv.Atoi(pm[1])
		events = append(events, gk{at: at, in: toks[j] == "GK In", key: fmt.Sprintf("%s|%d", s, jersey)})
		if at > end {
			end = at
		}
	}
	sort.SliceStable(events, func(a, b int) bool { return events[a].at < events[b].at })
	inSince := map[string]int{}
	for _, e := range events {
		idx, ok := byKey[e.key]
		if !ok {
			continue
		}
		if e.in {
			if e.at == 0 {
				ev.Goalies[idx].Starter = true
			}
			inSince[e.key] = e.at
			continue
		}
		if t0, ok := inSince[e.key]; ok {
			ev.Goalies[idx].TOISeconds += e.at - t0
			delete(inSince, e.key)
		}
	}
	for key, t0 := range inSince {
		ev.Goalies[byKey[key]].TOISeconds += end - t0
	}
	// No GK events at all (older seasons): the first listed goalie per side started.
	if len(events) == 0 {
		seen := map[string]bool{}
		for k := range ev.Goalies {
			if !seen[ev.Goalies[k].Side] {
				ev.Goalies[k].Starter = true
				seen[ev.Goalies[k].Side] = true
			}
		}
	}
	return ev, nil
}

func clock(s string) (int, bool) {
	m := reClock.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	mm, _ := strconv.Atoi(m[1])
	ss, _ := strconv.Atoi(m[2])
	return mm*60 + ss, true
}

// flipName turns "Werner, Adam" into "Adam Werner".
func flipName(s string) string {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(parts[1]) + " " + strings.TrimSpace(parts[0])
}

// ---- small HTML helpers ----

func findAll(n *html.Node, tag string) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == tag {
			out = append(out, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func children(n *html.Node, tag string) []*html.Node {
	var out []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			out = append(out, c)
		}
	}
	return out
}

// text joins the text nodes under n with single spaces.
func text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

// textExcept is text() without the subtrees of the given tag.
func textExcept(n *html.Node, skip string) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == skip {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

// renderAttrs concatenates all attribute values under n (to find hrefs).
func renderAttrs(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for _, a := range n.Attr {
			b.WriteString(a.Val)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// tokens returns non-empty, whitespace-collapsed text nodes in document order,
// skipping script and style.
func tokens(n *html.Node) []string {
	var out []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		if n.Type == html.TextNode {
			if s := strings.Join(strings.Fields(n.Data), " "); s != "" {
				out = append(out, s)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func indexOf(xs []string, s string) int {
	for i, x := range xs {
		if x == s {
			return i
		}
	}
	return -1
}
