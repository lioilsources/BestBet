package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lioilsources/BestBet/internal/asa"
	"github.com/lioilsources/BestBet/internal/config"
	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/fdcouk"
	"github.com/lioilsources/BestBet/internal/hockey"
	"github.com/lioilsources/BestBet/internal/odds"
	"github.com/lioilsources/BestBet/internal/soccer"
	"github.com/lioilsources/BestBet/internal/store"
)

// SoccerStats extends Stats with what only soccer ingest counts.
type SoccerStats struct {
	Stats
	NoHalves int // finished games whose halves could not be reconciled
	Shots    int
}

func (s SoccerStats) String() string {
	return fmt.Sprintf("%s no_halves=%d shots=%d", s.Stats.String(), s.NoHalves, s.Shots)
}

// soccerSeasonOngoing: MLS seasons are calendar years until the 2027 change
// (plan 05 §2); a season is final once its year is over.
func soccerSeasonOngoing(season int, now time.Time) bool { return now.Year() <= season }

// ASA ingests one season of American Soccer Analysis games with results by
// half (from the shots). ASA owns soccer games; teams are created by name.
func ASA(ctx context.Context, e Env, c *asa.Client, l config.League, row dbq.League, season int) (SoccerStats, error) {
	var st SoccerStats
	ttl := time.Duration(0)
	if soccerSeasonOngoing(season, e.Now()) {
		ttl = 6 * time.Hour
	}
	teamList, err := c.Teams(ctx, 7*24*time.Hour)
	if err != nil {
		return st, err
	}
	names := map[string]string{}
	for _, t := range teamList {
		names[t.ID] = t.Name
	}
	teams, err := e.Store.LoadTeams(ctx, row.ID, "asa", e.aliases("asa", l.Slug))
	if err != nil {
		return st, err
	}
	games, err := c.Games(ctx, season, ttl)
	if err != nil {
		return st, err
	}
	var finished []asa.Game
	for _, g := range games {
		if g.Finished() {
			finished = append(finished, g)
		}
	}
	shots, err := c.Shots(ctx, finished, ttl)
	if err != nil {
		return st, err
	}
	for _, g := range games {
		st.Seen++
		in, err := asaInput(ctx, teams, names, row.ID, g, shots[g.ID], &st)
		if err != nil {
			return st, err
		}
		_, method, err := e.Store.UpsertASAGame(ctx, g.ID, in)
		if err != nil {
			return st, err
		}
		if method == "new" {
			st.New++
		} else {
			st.Linked++
		}
	}
	if len(teams.Created) > 0 {
		slog.Info("asa new teams", "league", l.Slug, "teams", teams.Created)
	}
	slog.Info("asa", "league", l.Slug, "season", season, "stats", st.String(), "requests", c.HTTP.Used())
	return st, nil
}

func asaInput(ctx context.Context, teams *store.Teams, names map[string]string, leagueID int32, g asa.Game, shots []asa.Shot, st *SoccerStats) (store.GameInput, error) {
	in := store.GameInput{LeagueID: leagueID, StartAt: g.StartAt, Status: "scheduled", Phase: hockey.PhaseRegular}
	y, err := g.SeasonYear()
	if err != nil {
		return in, fmt.Errorf("asa game %s season %q: %w", g.ID, g.Season, err)
	}
	in.Season = int32(y)
	if g.Knockout {
		in.Phase = hockey.PhasePlayoff
		in.StageRaw = "knockout"
	}
	for _, side := range []struct {
		id  string
		dst *int32
	}{{g.HomeTeam, &in.HomeTeamID}, {g.AwayTeam, &in.AwayTeamID}} {
		name, ok := names[side.id]
		if !ok {
			return in, fmt.Errorf("asa game %s: unknown team id %s", g.ID, side.id)
		}
		if *side.dst, err = teams.Resolve(ctx, name, 0); err != nil {
			return in, err
		}
	}
	if !g.Finished() {
		return in, nil
	}
	in.Status = "finished"
	st.Shots += len(shots)
	r, otType, ok, err := asa.Result(g, shots)
	if err != nil {
		return in, fmt.Errorf("asa game %s: %w", g.ID, err)
	}
	if !ok {
		// Keep the score; without halves the game is not "played" for models.
		st.NoHalves++
		fin := soccer.Score{Home: g.HomeScore, Away: g.AwayScore}
		r = soccer.Result{Reg: fin, Final: fin, DecidedIn: "reg"}
		if g.Knockout && fin.Home == fin.Away {
			r.DecidedIn = "so"
		}
		otType = "unknown"
	}
	in.Result = &r
	in.OTType = otType
	return in, nil
}

// FDCOUK attaches football-data.co.uk closing 1X2 prices to soccer games. It
// is an odds feed: unknown teams or games are failed joins in fixture_links.
// Closing prices are stored one minute before kick-off (the file has no
// capture time; backtests read prices captured before the start only).
func FDCOUK(ctx context.Context, e Env, b []byte, l config.League, row dbq.League, league string) (Stats, error) {
	var st Stats
	rows, err := fdcouk.Parse(b)
	if err != nil {
		return st, err
	}
	teams, err := e.Store.LoadTeams(ctx, row.ID, "fdcouk", e.aliases("fdcouk", l.Slug))
	if err != nil {
		return st, err
	}
	seasons := map[int]bool{}
	for _, y := range l.Seasons {
		seasons[y] = true
	}
	var batch []dbq.InsertOddsBatchParams
	for _, r := range rows {
		// Only seasons the game source covers; others would be failed joins.
		if r.League != league || (len(seasons) > 0 && !seasons[r.Season]) {
			continue
		}
		st.Seen++
		gameID, detail, err := linkByTeams(ctx, e.Store, teams, row.ID, r.Home, r.Away, r.StartAt)
		if err != nil {
			return st, err
		}
		method := "date_teams"
		if gameID == 0 {
			method = "none"
			st.Failed++
		} else {
			st.Linked++
		}
		if err := e.Store.Link(ctx, "fdcouk", r.FixtureID(), row.ID, gameID, method, detail); err != nil {
			return st, err
		}
		at := r.StartAt.Add(-time.Minute)
		for book, p := range r.Closing {
			for i, sel := range []string{"home", "draw", "away"} {
				q := odds.Quote{Market: "1x2", Selection: sel, Price: p[i], RawMarket: "1X2 closing"}
				batch = append(batch, dbq.InsertOddsBatchParams(quoteParams(gameID, "fdcouk", r.FixtureID(), book, q, "closing", at)))
			}
		}
		if len(batch) >= 3000 {
			if err := insertQuotes(ctx, e.Store, batch); err != nil {
				return st, err
			}
			st.Quotes += len(batch)
			batch = batch[:0]
		}
	}
	if err := insertQuotes(ctx, e.Store, batch); err != nil {
		return st, err
	}
	st.Quotes += len(batch)
	slog.Info("fdcouk", "league", l.Slug, "stats", st.String())
	return st, nil
}
