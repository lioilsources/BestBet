package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/features"
)

// PlayerInfo is what the app shows about a skater (season so far).
type PlayerInfo struct {
	Name   string
	Games  int
	Points int
}

// TeamGames loads a team's skater lines of one season before a moment, as
// feature input, plus names and points for display.
func (s *Store) TeamGames(ctx context.Context, team int32, season int, before time.Time) ([]features.TeamGame, map[int32]PlayerInfo, error) {
	rows, err := s.Q.TeamPlayerLines(ctx, dbq.TeamPlayerLinesParams{TeamID: team, Season: int32(season), StartAt: pgtype.Timestamptz{Time: before, Valid: true}})
	if err != nil {
		return nil, nil, fmt.Errorf("team lines: %w", err)
	}
	var out []features.TeamGame
	info := map[int32]PlayerInfo{}
	var last int64
	for _, r := range rows {
		if len(out) == 0 || r.GameID != last {
			out = append(out, features.TeamGame{StartAt: r.StartAt.Time.UTC(), Season: season})
			last = r.GameID
		}
		g := &out[len(out)-1]
		pm := canon.PlayerMinutes{PlayerID: r.PlayerID}
		if r.Position != nil {
			pm.Position = *r.Position
		}
		if r.ToiSeconds != nil {
			pm.TOISeconds = int(*r.ToiSeconds)
		}
		g.Players = append(g.Players, pm)
		pi := info[r.PlayerID]
		pi.Name = r.Name
		pi.Games++
		if r.Goals != nil {
			pi.Points += int(*r.Goals)
		}
		if r.Assists != nil {
			pi.Points += int(*r.Assists)
		}
		info[r.PlayerID] = pi
	}
	return out, info, nil
}

// TeamLineup is a published pre-game lineup of one team.
type TeamLineup struct {
	Players     []int32
	ConfirmedAt time.Time // first poll that listed the lineup
}

// Lineup statuses as stored.
const (
	LineupAvailable = "available"
	LineupConfirmed = "confirmed"
)

// Lineups returns the published lists of a game per team and status.
func (s *Store) Lineups(ctx context.Context, gameID int64) (map[int32]map[string]TeamLineup, map[int32]string, error) {
	rows, err := s.Q.ListLineup(ctx, gameID)
	if err != nil {
		return nil, nil, fmt.Errorf("lineup: %w", err)
	}
	out := map[int32]map[string]TeamLineup{}
	names := map[int32]string{}
	for _, r := range rows {
		if out[r.TeamID] == nil {
			out[r.TeamID] = map[string]TeamLineup{}
		}
		l := out[r.TeamID][r.Status]
		l.Players = append(l.Players, r.PlayerID)
		if t := r.ConfirmedAt.Time.UTC(); l.ConfirmedAt.IsZero() || t.Before(l.ConfirmedAt) {
			l.ConfirmedAt = t
		}
		out[r.TeamID][r.Status] = l
		names[r.PlayerID] = r.Name
	}
	return out, names, nil
}

// LineupPlayer is one listed player of a pre-game lineup.
type LineupPlayer struct {
	PlayerID int32
	Position string
}

// ReplaceLineup stores a team's list of one status as seen at a poll: new
// players get confirmed_at = at, players no longer listed are dropped.
func (s *Store) ReplaceLineup(ctx context.Context, gameID int64, team int32, status string, ps []LineupPlayer, at time.Time, source string) error {
	keep := make([]int32, 0, len(ps))
	for _, p := range ps {
		keep = append(keep, p.PlayerID)
		pos := p.Position
		if err := s.Q.InsertLineupPlayer(ctx, dbq.InsertLineupPlayerParams{GameID: gameID, TeamID: team, Status: status, PlayerID: p.PlayerID,
			Position: &pos, ConfirmedAt: pgtype.Timestamptz{Time: at, Valid: true}, Source: source}); err != nil {
			return fmt.Errorf("lineup player: %w", err)
		}
	}
	if err := s.Q.DeleteLineupExcept(ctx, dbq.DeleteLineupExceptParams{GameID: gameID, TeamID: team, Status: status, Keep: keep}); err != nil {
		return fmt.Errorf("lineup prune: %w", err)
	}
	return nil
}

// AttachLineups fills Match.Lineups from the stored skater lines of a league.
func (s *Store) AttachLineups(ctx context.Context, slug string, ms []canon.Match) error {
	rows, err := s.Q.ListLeagueSkaterLines(ctx, slug)
	if err != nil {
		return fmt.Errorf("skater lines: %w", err)
	}
	by := map[int64]map[canon.Side][]canon.PlayerMinutes{}
	for _, r := range rows {
		if by[r.GameID] == nil {
			by[r.GameID] = map[canon.Side][]canon.PlayerMinutes{}
		}
		pm := canon.PlayerMinutes{PlayerID: r.PlayerID}
		if r.Position != nil {
			pm.Position = *r.Position
		}
		if r.ToiSeconds != nil {
			pm.TOISeconds = int(*r.ToiSeconds)
		}
		side := canon.Side(r.Side)
		by[r.GameID][side] = append(by[r.GameID][side], pm)
	}
	for i := range ms {
		for _, side := range []canon.Side{canon.Home, canon.Away} {
			if ps := by[ms[i].ID][side]; len(ps) > 0 {
				ms[i].Lineups = append(ms[i].Lineups, canon.Lineup{Side: side, Players: ps})
			}
		}
	}
	return nil
}
