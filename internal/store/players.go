package store

import (
	"context"
	"fmt"

	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/teammatch"
)

// Players resolves a source's player ids to players.id, creating players on
// first sight. The whole mapping of the source is loaded once (one query) so
// a season backfill does not pay a lookup per skater.
type Players struct {
	s      *Store
	source string
	ids    map[string]int32
	New    int
}

// LoadPlayers loads the player mapping of one source.
func (s *Store) LoadPlayers(ctx context.Context, source string) (*Players, error) {
	rows, err := s.Q.ListPlayerSources(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("player sources: %w", err)
	}
	p := &Players{s: s, source: source, ids: make(map[string]int32, len(rows))}
	for _, r := range rows {
		p.ids[r.SourcePlayerID] = r.PlayerID
	}
	return p, nil
}

// Resolve returns the player id for a source id, creating the player if new.
func (p *Players) Resolve(ctx context.Context, sourceID, name, position string) (int32, error) {
	if id, ok := p.ids[sourceID]; ok {
		return id, nil
	}
	var pos *string
	if position != "" {
		pos = &position
	}
	id, err := p.s.Q.InsertPlayer(ctx, dbq.InsertPlayerParams{Name: name, NameNorm: teammatch.Norm(name), Position: pos, CreatedBy: p.source})
	if err != nil {
		return 0, fmt.Errorf("insert player %s: %w", sourceID, err)
	}
	if err := p.s.Q.InsertPlayerSource(ctx, dbq.InsertPlayerSourceParams{Source: p.source, SourcePlayerID: sourceID, PlayerID: id}); err != nil {
		return 0, fmt.Errorf("player source %s: %w", sourceID, err)
	}
	p.ids[sourceID] = id
	p.New++
	return id, nil
}

// UpsertPlayerLines writes one game's skater lines in a single round trip.
func (s *Store) UpsertPlayerLines(ctx context.Context, rows []dbq.UpsertPlayerMatchStatsBatchParams) error {
	if len(rows) == 0 {
		return nil
	}
	var first error
	s.Q.UpsertPlayerMatchStatsBatch(ctx, rows).Exec(func(_ int, err error) {
		if err != nil && first == nil {
			first = err
		}
	})
	if first != nil {
		return fmt.Errorf("player lines: %w", first)
	}
	return nil
}
