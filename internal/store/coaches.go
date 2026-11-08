package store

import (
	"context"
	"fmt"

	"github.com/lioilsources/BestBet/internal/features"
)

// LoadCoachTenures returns a league's head-coach tenures (empty when not ingested).
func (s *Store) LoadCoachTenures(ctx context.Context, slug string) ([]features.CoachTenure, error) {
	rows, err := s.Q.ListLeagueCoachTenures(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("coach tenures: %w", err)
	}
	out := make([]features.CoachTenure, 0, len(rows))
	for _, r := range rows {
		out = append(out, features.CoachTenure{Team: r.TeamID, Coach: r.Name, From: r.FromDate.Time, To: r.ToDate.Time})
	}
	return out, nil
}
