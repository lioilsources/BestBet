package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/lioilsources/BestBet/internal/db/dbq"
)

// UpsertASAGame stores a soccer game owned by American Soccer Analysis
// (plan 05 S1): the latest pull overwrites the result.
func (s *Store) UpsertASAGame(ctx context.Context, asaID string, in GameInput) (int64, string, error) {
	method := "external_id"
	if _, err := s.Q.GetGameByAsaID(ctx, &asaID); errors.Is(err, pgx.ErrNoRows) {
		method = "new"
	} else if err != nil {
		return 0, "", fmt.Errorf("get asa game %s: %w", asaID, err)
	}
	p := dbq.UpsertGameASAParams{
		LeagueID: in.LeagueID, Season: in.Season, StartAt: ts(in.StartAt), Status: in.Status,
		HomeTeamID: in.HomeTeamID, AwayTeamID: in.AwayTeamID, Phase: in.Phase,
		StageRaw: strPtr(in.StageRaw), AsaID: &asaID,
	}
	if err := fillResult(in, &p.HomeScore, &p.AwayScore, &p.Home60, &p.Away60, &p.PeriodScores, &p.OtHome, &p.OtAway, &p.SoWinner, &p.DecidedIn, &p.OtType); err != nil {
		return 0, "", err
	}
	id, err := s.Q.UpsertGameASA(ctx, p)
	if err != nil {
		return 0, "", fmt.Errorf("upsert asa game %s: %w", asaID, err)
	}
	return id, method, nil
}
