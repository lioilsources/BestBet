package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/nhl"
)

// CoachGame is one game of a team with its head coach.
type CoachGame struct {
	Team  int32
	At    time.Time
	Coach string
}

// Tenure is a run of consecutive games of one team under one head coach.
type Tenure struct {
	Team     int32
	Coach    string
	From, To time.Time
}

// Tenures splits each team's games (any order) into consecutive runs under the
// same coach. Games without a coach name are skipped, not treated as a change.
func Tenures(games []CoachGame) []Tenure {
	gs := append([]CoachGame(nil), games...)
	sort.SliceStable(gs, func(i, j int) bool {
		if gs[i].Team != gs[j].Team {
			return gs[i].Team < gs[j].Team
		}
		return gs[i].At.Before(gs[j].At)
	})
	var out []Tenure
	for _, g := range gs {
		if g.Coach == "" {
			continue
		}
		if n := len(out); n > 0 && out[n-1].Team == g.Team && out[n-1].Coach == g.Coach {
			out[n-1].To = g.At
			continue
		}
		out = append(out, Tenure{Team: g.Team, Coach: g.Coach, From: g.At, To: g.At})
	}
	return out
}

// NHLCoaches reads the head coach of every finished NHL game from the public
// right-rail feed (cached forever) and rewrites the league's head-coach tenures.
func NHLCoaches(ctx context.Context, e Env, hc *httpcache.Client, row dbq.League) (int, error) {
	games, err := e.Store.Q.ListNHLFinishedGames(ctx, row.Slug)
	if err != nil {
		return 0, fmt.Errorf("games: %w", err)
	}
	var cg []CoachGame
	failed := 0
	for i, g := range games {
		u := fmt.Sprintf("%s/gamecenter/%d/right-rail", nhl.Base, *g.NhlID)
		b, _, err := hc.Get(ctx, httpcache.Request{URL: u, Key: u})
		if errors.Is(err, httpcache.ErrBudget) {
			return 0, err
		}
		if err != nil {
			failed++
			slog.Warn("nhl right-rail", "game", *g.NhlID, "err", err)
			continue
		}
		c, err := nhl.ParseRightRail(b)
		if err != nil {
			failed++
			continue
		}
		at := g.StartAt.Time.UTC()
		cg = append(cg, CoachGame{g.HomeTeamID, at, c.Home}, CoachGame{g.AwayTeamID, at, c.Away})
		if (i+1)%500 == 0 {
			slog.Info("nhl coaches", "done", i+1, "of", len(games))
		}
	}
	ts := Tenures(cg)
	if err := e.Store.Q.DeleteLeagueCoachTenures(ctx, dbq.DeleteLeagueCoachTenuresParams{LeagueID: row.ID, Source: "nhl"}); err != nil {
		return 0, fmt.Errorf("delete tenures: %w", err)
	}
	ids := map[string]int32{}
	for _, t := range ts {
		id, ok := ids[t.Coach]
		if !ok {
			id, err = e.Store.Q.GetCoachByName(ctx, t.Coach)
			if errors.Is(err, pgx.ErrNoRows) {
				id, err = e.Store.Q.InsertCoach(ctx, t.Coach)
			}
			if err != nil {
				return 0, fmt.Errorf("coach %s: %w", t.Coach, err)
			}
			ids[t.Coach] = id
		}
		if err := e.Store.Q.InsertCoachTenure(ctx, dbq.InsertCoachTenureParams{
			CoachID: id, TeamID: t.Team, Source: "nhl",
			FromDate: pgtype.Date{Time: t.From, Valid: true}, ToDate: pgtype.Date{Time: t.To, Valid: true},
		}); err != nil {
			return 0, fmt.Errorf("tenure: %w", err)
		}
	}
	slog.Info("nhl coaches", "games", len(games), "failed", failed, "tenures", len(ts), "coaches", len(ids))
	return len(ts), nil
}
