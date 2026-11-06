package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/db/dbq"
)

// ErrNotFound is returned for a missing row.
var ErrNotFound = errors.New("store: not found")

// SavePrediction stores a prediction; the same inputs hash is stored once.
func (s *Store) SavePrediction(ctx context.Context, kind string, p canon.Prediction) error {
	mx, err := json.Marshal(p.Matrix)
	if err != nil {
		return err
	}
	mk, err := json.Marshal(p.Markets)
	if err != nil {
		return err
	}
	var contrib []byte
	if len(p.Contributions) > 0 {
		if contrib, err = json.Marshal(p.Contributions); err != nil {
			return err
		}
	}
	err = s.Q.InsertPrediction(ctx, dbq.InsertPredictionParams{
		GameID: p.MatchID, Kind: kind, ModelVersion: p.ModelVersion, InputsHash: p.InputsHash,
		ComputedAt: pgtype.Timestamptz{Time: p.ComputedAt, Valid: true}, LambdaHome: round3(p.LambdaHome), LambdaAway: round3(p.LambdaAway),
		ScoreMatrix: mx, Markets: mk, Contributions: contrib,
	})
	if err != nil {
		return fmt.Errorf("save prediction %d: %w", p.MatchID, err)
	}
	return nil
}

func round3(f float64) float64 { return float64(int64(f*1000+0.5)) / 1000 }

// LatestPrediction returns the newest live prediction of a game, nil if none.
func (s *Store) LatestPrediction(ctx context.Context, gameID int64) (*canon.Prediction, error) {
	r, err := s.Q.LatestPrediction(ctx, gameID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest prediction %d: %w", gameID, err)
	}
	p := &canon.Prediction{MatchID: r.GameID, ModelVersion: r.ModelVersion, InputsHash: r.InputsHash, ComputedAt: r.ComputedAt.Time.UTC(),
		LambdaHome: r.LambdaHome, LambdaAway: r.LambdaAway}
	if err := json.Unmarshal(r.ScoreMatrix, &p.Matrix); err != nil {
		return nil, fmt.Errorf("prediction %d matrix: %w", gameID, err)
	}
	if err := json.Unmarshal(r.Markets, &p.Markets); err != nil {
		return nil, fmt.Errorf("prediction %d markets: %w", gameID, err)
	}
	if len(r.Contributions) > 0 {
		if err := json.Unmarshal(r.Contributions, &p.Contributions); err != nil {
			return nil, fmt.Errorf("prediction %d contributions: %w", gameID, err)
		}
	}
	return p, nil
}

// GamesBetween lists games starting in [from, to).
func (s *Store) GamesBetween(ctx context.Context, from, to time.Time) ([]canon.GameInfo, error) {
	rows, err := s.Q.ListGamesBetween(ctx, dbq.ListGamesBetweenParams{FromTs: ts(from), ToTs: ts(to)})
	if err != nil {
		return nil, fmt.Errorf("games between: %w", err)
	}
	out := make([]canon.GameInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, gameInfo(dbq.GetGameRow(r)))
	}
	return out, nil
}

// Game returns one game.
func (s *Store) Game(ctx context.Context, id int64) (canon.GameInfo, error) {
	r, err := s.Q.GetGame(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return canon.GameInfo{}, ErrNotFound
	}
	if err != nil {
		return canon.GameInfo{}, fmt.Errorf("game %d: %w", id, err)
	}
	return gameInfo(r), nil
}

func gameInfo(r dbq.GetGameRow) canon.GameInfo {
	g := canon.GameInfo{
		Match: canon.Match{ID: r.ID, League: r.LeagueSlug, Season: int(r.Season), StartAt: r.StartAt.Time.UTC(),
			HomeTeam: r.HomeTeamID, AwayTeam: r.AwayTeamID, Phase: r.Phase},
		LeagueName: r.LeagueName, HomeName: r.HomeName, AwayName: r.AwayName, Status: r.Status,
	}
	if r.Home60 != nil && r.Away60 != nil {
		g.Reg = &canon.Score{Home: int(*r.Home60), Away: int(*r.Away60)}
	}
	if r.HomeScore != nil && r.AwayScore != nil {
		g.Final = &canon.Score{Home: int(*r.HomeScore), Away: int(*r.AwayScore)}
	}
	if r.DecidedIn != nil {
		g.DecidedIn = *r.DecidedIn
	}
	return g
}

// LatestOdds returns the latest pre-start price per book, market, selection, line.
func (s *Store) LatestOdds(ctx context.Context, gameID int64) ([]canon.OddsQuote, error) {
	rows, err := s.Q.LatestOdds(ctx, &gameID)
	if err != nil {
		return nil, fmt.Errorf("latest odds %d: %w", gameID, err)
	}
	out := make([]canon.OddsQuote, 0, len(rows))
	for _, r := range rows {
		out = append(out, canon.OddsQuote{Book: r.Book, Market: r.Market, Selection: r.Selection, Line: r.Line, Price: r.Price, CapturedAt: r.CapturedAt.Time.UTC()})
	}
	return out, nil
}

// LeagueMatches loads a league for the model (results, goalies; no odds).
func (s *Store) LeagueMatches(ctx context.Context, slug string) ([]canon.Match, error) {
	ms, _, err := s.LoadMatches(ctx, slug)
	return ms, err
}

// Journal lists a user's recorded bets, newest first.
func (s *Store) Journal(ctx context.Context, user string) ([]canon.JournalEntry, error) {
	rows, err := s.Q.ListJournal(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("journal: %w", err)
	}
	out := make([]canon.JournalEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, canon.JournalEntry{ID: r.ID, GameID: r.GameID, Book: r.Book, Market: r.Market, Selection: r.Selection, Line: r.Line,
			Price: r.Price, Stake: r.Stake, PlacedAt: r.PlacedAt.Time.UTC(), ModelP: r.ModelP, ClosingPrice: r.ClosingPrice, Result: r.Result, Note: r.Note})
	}
	return out, nil
}

// LastOddsCapture is the newest capture of a source (epoch when none).
func (s *Store) LastOddsCapture(ctx context.Context, source string) (time.Time, error) {
	t, err := s.Q.LastOddsCapture(ctx, source)
	if err != nil {
		return time.Time{}, fmt.Errorf("last odds capture: %w", err)
	}
	return t.Time.UTC(), nil
}
