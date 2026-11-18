// Package fdcouk parses the football-data.co.uk "extra leagues" CSV (e.g.
// new/USA.csv): one row per game with the score after regular or extra time
// and closing 1X2 prices of Pinnacle, Bet365, Betfair Exchange and the market
// maximum and average. Kick-off times are UTC (verified against American
// Soccer Analysis in plan 05 S0). It is an odds feed: it never creates games.
package fdcouk

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// URLUSA is the MLS file.
const URLUSA = "https://www.football-data.co.uk/new/USA.csv"

// Books maps the column prefix of closing 1X2 prices to our book name.
var Books = []struct{ Prefix, Book string }{
	{"PSC", "pinnacle"},
	{"B365C", "bet365"},
	{"BFEC", "betfair_ex"},
	{"MaxC", "market_max"},
	{"AvgC", "market_avg"},
}

// Row is one game.
type Row struct {
	League  string
	Season  int
	StartAt time.Time
	Home    string
	Away    string
	HG, AG  int
	// Closing 1X2 per book: home, draw, away. Missing prices are left out.
	Closing map[string][3]float64
}

// FixtureID is a stable id for fixture_links (the file has none).
func (r Row) FixtureID() string {
	return r.StartAt.Format("2006-01-02T15:04") + "|" + r.Home + "|" + r.Away
}

// Parse reads the whole file.
func Parse(b []byte) ([]Row, error) {
	cr := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))))
	cr.FieldsPerRecord = -1
	head, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("fdcouk header: %w", err)
	}
	col := map[string]int{}
	for i, h := range head {
		col[strings.TrimSpace(h)] = i
	}
	for _, need := range []string{"League", "Season", "Date", "Time", "Home", "Away", "HG", "AG"} {
		if _, ok := col[need]; !ok {
			return nil, fmt.Errorf("fdcouk: no column %s", need)
		}
	}
	get := func(rec []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}
	var out []Row
	for line := 2; ; line++ {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("fdcouk line %d: %w", line, err)
		}
		if get(rec, "Home") == "" {
			continue
		}
		r := Row{League: get(rec, "League"), Home: get(rec, "Home"), Away: get(rec, "Away"), Closing: map[string][3]float64{}}
		if r.Season, err = strconv.Atoi(get(rec, "Season")); err != nil {
			return nil, fmt.Errorf("fdcouk line %d season: %w", line, err)
		}
		if r.StartAt, err = time.Parse("02/01/2006 15:04", get(rec, "Date")+" "+get(rec, "Time")); err != nil {
			return nil, fmt.Errorf("fdcouk line %d date: %w", line, err)
		}
		if r.HG, err = strconv.Atoi(get(rec, "HG")); err != nil {
			return nil, fmt.Errorf("fdcouk line %d HG: %w", line, err)
		}
		if r.AG, err = strconv.Atoi(get(rec, "AG")); err != nil {
			return nil, fmt.Errorf("fdcouk line %d AG: %w", line, err)
		}
		for _, bk := range Books {
			var p [3]float64
			ok := true
			for i, s := range []string{"H", "D", "A"} {
				v, err := strconv.ParseFloat(get(rec, bk.Prefix+s), 64)
				if err != nil || v <= 1 {
					ok = false
					break
				}
				p[i] = v
			}
			if ok {
				r.Closing[bk.Book] = p
			}
		}
		out = append(out, r)
	}
	return out, nil
}
