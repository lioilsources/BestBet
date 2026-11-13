package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/nhl"
	"github.com/lioilsources/BestBet/internal/store"
)

// LineupWindow is how far ahead nhl-lineups looks for games (plan 04 L2).
const LineupWindow = 90 * time.Minute

// NHLLineups polls play-by-play of NHL games starting within LineupWindow.
// rosterSpots first list the available players (stored as available) and
// narrow to the dressed lineup minutes before the start (stored as
// confirmed; plan 04 L0). No game in the window means no request.
func NHLLineups(ctx context.Context, e Env, hc *httpcache.Client) (games, stored int, err error) {
	now := e.Now()
	rows, err := e.Store.Q.ListNHLGamesStarting(ctx, dbq.ListNHLGamesStartingParams{
		StartAt: pgtype.Timestamptz{Time: now.Add(-5 * time.Minute), Valid: true}, StartAt_2: pgtype.Timestamptz{Time: now.Add(LineupWindow), Valid: true}})
	if err != nil {
		return 0, 0, fmt.Errorf("upcoming nhl games: %w", err)
	}
	if len(rows) == 0 {
		return 0, 0, nil
	}
	players, err := e.Store.LoadPlayers(ctx, "nhl")
	if err != nil {
		return 0, 0, err
	}
	for _, g := range rows {
		if g.NhlID == nil {
			continue
		}
		u := fmt.Sprintf("%s/gamecenter/%d/play-by-play", nhl.Base, *g.NhlID)
		b, _, err := hc.Get(ctx, httpcache.Request{URL: u, Key: u, TTL: -1})
		if err != nil {
			slog.Warn("nhl lineup", "game", *g.NhlID, "err", err)
			continue
		}
		r, err := nhl.ParseRoster(b)
		if err != nil {
			return games, stored, err
		}
		games++
		for _, side := range []struct {
			team  int32
			spots []nhl.RosterSpot
		}{{g.HomeTeamID, r.Home}, {g.AwayTeamID, r.Away}} {
			status := store.LineupAvailable
			switch {
			case nhl.Dressed(side.spots):
				status = store.LineupConfirmed
			case len(side.spots) == 0:
				continue
			}
			ps := make([]store.LineupPlayer, 0, len(side.spots))
			for _, sp := range side.spots {
				id, err := players.Resolve(ctx, strconv.Itoa(sp.PlayerID), sp.Name, sp.Position)
				if err != nil {
					return games, stored, err
				}
				ps = append(ps, store.LineupPlayer{PlayerID: id, Position: sp.Position})
			}
			if err := e.Store.ReplaceLineup(ctx, g.ID, side.team, status, ps, now, "nhl"); err != nil {
				return games, stored, err
			}
			stored++
		}
	}
	slog.Info("nhl lineups", "games", games, "sides_stored", stored, "requests", hc.Used())
	return games, stored, nil
}
