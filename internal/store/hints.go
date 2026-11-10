package store

import (
	"context"
	"fmt"
	"time"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/hint"
)

// RecordHint writes the first qualifying price of a game; false when one exists.
func (s *Store) RecordHint(ctx context.Context, gameID int64, rule string, h hint.Hint, at time.Time) (bool, error) {
	n, err := s.Q.InsertHint(ctx, dbq.InsertHintParams{GameID: gameID, RuleVersion: rule, Book: h.Book, Market: h.Market,
		Selection: h.Selection, Price: h.Price, PSharp: h.PSharp, PSoft: h.PSoft, CapturedAt: ts(h.CapturedAt), RecordedAt: ts(at)})
	if err != nil {
		return false, fmt.Errorf("record hint %d: %w", gameID, err)
	}
	return n > 0, nil
}

// Hints lists the ledger of one rule, newest first.
func (s *Store) Hints(ctx context.Context, rule string) ([]hint.Entry, error) {
	rows, err := s.Q.ListHints(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("hints: %w", err)
	}
	out := make([]hint.Entry, 0, len(rows))
	for _, r := range rows {
		out = append(out, hint.Entry{ID: r.ID, GameID: r.GameID, Rule: r.RuleVersion, RecordedAt: r.RecordedAt.Time.UTC(),
			Hint: hint.Hint{Book: r.Book, Market: r.Market, Selection: r.Selection, Price: r.Price, PSharp: r.PSharp, PSoft: r.PSoft,
				Edge: r.PSharp - r.PSoft, EV: r.PSharp*r.Price - 1, CapturedAt: r.CapturedAt.Time.UTC()}})
	}
	return out, nil
}

// OddsHistory returns every stored pre-start price of a game, oldest first.
func (s *Store) OddsHistory(ctx context.Context, gameID int64) ([]canon.OddsQuote, error) {
	rows, err := s.Q.OddsHistory(ctx, &gameID)
	if err != nil {
		return nil, fmt.Errorf("odds history %d: %w", gameID, err)
	}
	out := make([]canon.OddsQuote, 0, len(rows))
	for _, r := range rows {
		out = append(out, canon.OddsQuote{Book: r.Book, Market: r.Market, Selection: r.Selection, Line: r.Line, Price: r.Price,
			CapturedAt: r.CapturedAt.Time.UTC(), Kind: r.SnapshotKind})
	}
	return out, nil
}

// Coverage is what the data page shows.
type Coverage struct {
	Games       []dbq.CoverageGamesRow       `json:"games"`
	Odds        []dbq.CoverageOddsRow        `json:"odds"`
	Coaches     []dbq.CoverageCoachesRow     `json:"coaches"`
	Predictions []dbq.CoveragePredictionsRow `json:"predictions"`
	At          time.Time                    `json:"at"`
}

// Coverage runs the (slow, full-table) coverage queries.
func (s *Store) Coverage(ctx context.Context) (Coverage, error) {
	var c Coverage
	var err error
	if c.Games, err = s.Q.CoverageGames(ctx); err != nil {
		return c, fmt.Errorf("coverage games: %w", err)
	}
	if c.Odds, err = s.Q.CoverageOdds(ctx); err != nil {
		return c, fmt.Errorf("coverage odds: %w", err)
	}
	if c.Coaches, err = s.Q.CoverageCoaches(ctx); err != nil {
		return c, fmt.Errorf("coverage coaches: %w", err)
	}
	if c.Predictions, err = s.Q.CoveragePredictions(ctx); err != nil {
		return c, fmt.Errorf("coverage predictions: %w", err)
	}
	c.At = time.Now().UTC()
	return c, nil
}
