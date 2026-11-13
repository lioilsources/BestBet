package features

import (
	"sort"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
)

// Absence thresholds, fixed in advance (plan 04 L2). They are not tuned.
const (
	AbsenceWindow   = 20  // last team games of the same season
	AbsenceMinGames = 5   // fewer earlier games: absences unknown
	RegularShare    = 0.6 // played at least this share of the window
	ReturnMinMissed = 3   // a return follows at least this many missed games
	keyTopFraction  = 3   // key player: average TOI in the top third of the team
	positionGoalie  = "G"
)

// TeamGame is one earlier game of a team: who dressed and their ice time.
type TeamGame struct {
	StartAt time.Time
	Season  int
	Players []canon.PlayerMinutes
}

// PlayerAbsence describes one regular player who is missing or returning.
type PlayerAbsence struct {
	PlayerID   int32
	Position   string
	TOIPerGame float64 // seconds, average over games played in the window
	Played     int     // games played in the window
	Missed     int     // consecutive team games missed right before this game
	Key        bool
}

// TeamAbsence is a team's absences before one game.
type TeamAbsence struct {
	Known     bool
	Window    int // team games in the window
	Regulars  int
	Missing   []PlayerAbsence
	Returning []PlayerAbsence
	Share     float64 // average TOI of missing regulars / average TOI of all regulars
}

type usage struct {
	position string
	games    int
	toi      int
}

// regulars returns the regulars of a window and whether each is key.
func regulars(window []TeamGame) (map[int32]PlayerAbsence, int) {
	u := map[int32]*usage{}
	for _, g := range window {
		for _, p := range g.Players {
			if p.Position == positionGoalie {
				continue
			}
			x := u[p.PlayerID]
			if x == nil {
				x = &usage{position: p.Position}
				u[p.PlayerID] = x
			}
			x.games++
			x.toi += p.TOISeconds
		}
	}
	if len(u) == 0 {
		return nil, 0
	}
	type row struct {
		id  int32
		avg float64
	}
	rows := make([]row, 0, len(u))
	for id, x := range u {
		rows = append(rows, row{id, float64(x.toi) / float64(x.games)})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].avg != rows[j].avg {
			return rows[i].avg > rows[j].avg
		}
		return rows[i].id < rows[j].id
	})
	avgs := make([]float64, len(rows))
	for i, r := range rows {
		avgs[i] = r.avg
	}
	med := median(avgs)
	keyN := (len(rows) + keyTopFraction - 1) / keyTopFraction
	out := map[int32]PlayerAbsence{}
	for i, r := range rows {
		x := u[r.id]
		if float64(x.games) < RegularShare*float64(len(window)) || r.avg <= med {
			continue
		}
		out[r.id] = PlayerAbsence{PlayerID: r.id, Position: x.position, TOIPerGame: r.avg, Played: x.games, Key: i < keyN}
	}
	return out, len(rows)
}

func median(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

// missedStreak counts the team games, newest first, the player did not dress.
func missedStreak(prev []TeamGame, id int32) int {
	n := 0
	for i := len(prev) - 1; i >= 0; i-- {
		if dressed(prev[i], id) {
			break
		}
		n++
	}
	return n
}

func dressed(g TeamGame, id int32) bool {
	for _, p := range g.Players {
		if p.PlayerID == id {
			return true
		}
	}
	return false
}

// AbsenceBefore compares a lineup with the team's regulars. prev holds the
// team's earlier games in time order; only games of season count. lineup is
// the dressed player ids (estimate: the last game's; confirmed: the real one).
func AbsenceBefore(prev []TeamGame, season int, lineup []int32) TeamAbsence {
	var same []TeamGame
	for _, g := range prev {
		if g.Season == season {
			same = append(same, g)
		}
	}
	var ta TeamAbsence
	if len(same) < AbsenceMinGames {
		return ta
	}
	ta.Known = true
	window := same
	if len(window) > AbsenceWindow {
		window = window[len(window)-AbsenceWindow:]
	}
	ta.Window = len(window)
	in := map[int32]bool{}
	for _, id := range lineup {
		in[id] = true
	}
	reg, _ := regulars(window)
	ta.Regulars = len(reg)
	var total, missing float64
	for id, p := range reg {
		total += p.TOIPerGame
		if in[id] {
			continue
		}
		p.Missed = missedStreak(same, id)
		missing += p.TOIPerGame
		ta.Missing = append(ta.Missing, p)
	}
	if total > 0 {
		ta.Share = missing / total
	}
	// Returning: dressed now, missed ReturnMinMissed+ games, regular in the
	// window that ended right before the absence began.
	for _, id := range lineup {
		if _, ok := reg[id]; ok {
			continue
		}
		k := missedStreak(same, id)
		if k < ReturnMinMissed {
			continue
		}
		before := same[:len(same)-k]
		if len(before) < AbsenceMinGames {
			continue
		}
		if len(before) > AbsenceWindow {
			before = before[len(before)-AbsenceWindow:]
		}
		r, _ := regulars(before)
		if p, ok := r[id]; ok {
			p.Missed = k
			ta.Returning = append(ta.Returning, p)
		}
	}
	sortAbsences(ta.Missing)
	sortAbsences(ta.Returning)
	return ta
}

func sortAbsences(xs []PlayerAbsence) {
	sort.Slice(xs, func(i, j int) bool {
		if xs[i].TOIPerGame != xs[j].TOIPerGame {
			return xs[i].TOIPerGame > xs[j].TOIPerGame
		}
		return xs[i].PlayerID < xs[j].PlayerID
	})
}

// TeamGames splits matches with lineups into each team's game list, in time order.
func TeamGames(matches []canon.Match) map[int32][]TeamGame {
	ms := append([]canon.Match(nil), matches...)
	canon.SortMatches(ms)
	out := map[int32][]TeamGame{}
	for _, m := range ms {
		for _, l := range m.Lineups {
			team := m.HomeTeam
			if l.Side == canon.Away {
				team = m.AwayTeam
			}
			if len(l.Players) == 0 {
				continue
			}
			out[team] = append(out[team], TeamGame{StartAt: m.StartAt, Season: m.Season, Players: l.Players})
		}
	}
	return out
}

// Lineup returns the dressed player ids of a team game.
func (g TeamGame) Lineup() []int32 {
	ids := make([]int32, 0, len(g.Players))
	for _, p := range g.Players {
		ids = append(ids, p.PlayerID)
	}
	return ids
}
