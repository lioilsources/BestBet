package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/db/dbq"
)

// LoadMatches converts a league's games, goalies and pre-start odds into canonical
// structures for the model and the backtest.
func (s *Store) LoadMatches(ctx context.Context, slug string) ([]canon.Match, map[int64][]canon.OddsQuote, error) {
	rows, err := s.Q.ListLeagueGames(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("games: %w", err)
	}
	out := make([]canon.Match, 0, len(rows))
	pos := map[int64]int{}
	for _, r := range rows {
		m, err := matchFromRow(slug, r)
		if err != nil {
			return nil, nil, err
		}
		pos[m.ID] = len(out)
		out = append(out, m)
	}
	gl, err := s.Q.ListLeagueGoalies(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("goalies: %w", err)
	}
	for _, g := range gl {
		i, ok := pos[g.GameID]
		if !ok {
			continue
		}
		out[i].Goalies = append(out[i].Goalies, canon.GoalieStart{
			Side: canon.Side(g.Side), Player: g.PlayerName, Starter: g.Starter,
			ShotsAgainst: deref(g.ShotsAgainst), Saves: deref(g.Saves), TOISeconds: deref(g.ToiSeconds), Confirm: g.ConfirmStatus,
		})
	}
	od, err := s.Q.ListLeagueOdds(ctx, slug)
	if err != nil {
		return nil, nil, fmt.Errorf("odds: %w", err)
	}
	odds := map[int64][]canon.OddsQuote{}
	for _, o := range od {
		if o.GameID == nil {
			continue
		}
		odds[*o.GameID] = append(odds[*o.GameID], canon.OddsQuote{
			Book: o.Book, Market: o.Market, Selection: o.Selection, Line: o.Line, Price: o.Price,
			CapturedAt: o.CapturedAt.Time, Kind: o.SnapshotKind,
		})
	}
	return out, odds, nil
}

func matchFromRow(slug string, r dbq.ListLeagueGamesRow) (canon.Match, error) {
	m := canon.Match{
		ID: r.ID, League: slug, Season: int(r.Season), StartAt: r.StartAt.Time.UTC(),
		HomeTeam: r.HomeTeamID, AwayTeam: r.AwayTeamID, Phase: r.Phase,
	}
	if r.Status != "finished" || r.Home60 == nil || r.Away60 == nil || len(r.PeriodScores) == 0 {
		return m, nil
	}
	var ps [][2]int
	if err := json.Unmarshal(r.PeriodScores, &ps); err != nil {
		return m, fmt.Errorf("game %d periods: %w", r.ID, err)
	}
	for _, p := range ps {
		m.Periods = append(m.Periods, canon.Score{Home: p[0], Away: p[1]})
	}
	m.Reg = &canon.Score{Home: int(*r.Home60), Away: int(*r.Away60)}
	if r.HomeScore != nil && r.AwayScore != nil {
		m.Final = &canon.Score{Home: int(*r.HomeScore), Away: int(*r.AwayScore)}
	}
	if r.DecidedIn != nil {
		m.DecidedIn = *r.DecidedIn
	}
	return m, nil
}

func deref(p *int32) int {
	if p == nil {
		return 0
	}
	return int(*p)
}
