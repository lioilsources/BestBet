package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/lioilsources/BestBet/internal/db/dbq"
)

// UpsertNHLGame attaches an NHL API game to an existing game or creates it
// (same rules as swehockey: API-HOCKEY owns a game that has its id).
func (s *Store) UpsertNHLGame(ctx context.Context, nhlID int64, in GameInput) (int64, string, error) {
	if g, err := s.Q.GetGameByNhlID(ctx, &nhlID); err == nil {
		return g.ID, "external_id", s.attachNHL(ctx, g.ID, nhlID, in)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, "", fmt.Errorf("get game: %w", err)
	}
	g, err := s.FindGame(ctx, in.LeagueID, in.HomeTeamID, in.AwayTeamID, in.StartAt)
	if err != nil {
		return 0, "", err
	}
	if g != nil && g.NhlID == nil {
		return g.ID, "date_teams", s.attachNHL(ctx, g.ID, nhlID, in)
	}
	p := dbq.InsertGameNHLParams{
		LeagueID: in.LeagueID, Season: in.Season, StartAt: ts(in.StartAt), Status: in.Status,
		HomeTeamID: in.HomeTeamID, AwayTeamID: in.AwayTeamID, Phase: in.Phase,
		StageRaw: strPtr(in.StageRaw), NhlID: &nhlID,
	}
	if err := fillResult(in, &p.HomeScore, &p.AwayScore, &p.Home60, &p.Away60, &p.PeriodScores, &p.OtHome, &p.OtAway, &p.SoWinner, &p.DecidedIn, &p.OtType); err != nil {
		return 0, "", err
	}
	id, err := s.Q.InsertGameNHL(ctx, p)
	if err != nil {
		return 0, "", fmt.Errorf("insert nhl game %d: %w", nhlID, err)
	}
	return id, "new", nil
}

func (s *Store) attachNHL(ctx context.Context, gameID, nhlID int64, in GameInput) error {
	p := dbq.AttachNHLParams{ID: gameID, NhlID: &nhlID, Status: in.Status, StartAt: ts(in.StartAt), Phase: in.Phase}
	if err := fillResult(in, &p.HomeScore, &p.AwayScore, &p.Home60, &p.Away60, &p.PeriodScores, &p.OtHome, &p.OtAway, &p.SoWinner, &p.DecidedIn, &p.OtType); err != nil {
		return err
	}
	if err := s.Q.AttachNHL(ctx, p); err != nil {
		return fmt.Errorf("attach nhl %d: %w", nhlID, err)
	}
	return nil
}
