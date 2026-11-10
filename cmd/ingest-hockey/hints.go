package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/lioilsources/BestBet/internal/hint"
	"github.com/lioilsources/BestBet/internal/store"
)

// cmdHints records the first qualifying price of every game starting in the
// next 48 h (plan 03 §3). Prices come from the latest stored pre-start quotes,
// so run it right after the odds ingest.
func cmdHints(ctx context.Context, st *store.Store, r hint.Rule, now time.Time) error {
	games, err := st.GamesBetween(ctx, now, now.Add(48*time.Hour))
	if err != nil {
		return err
	}
	seen, recorded := 0, 0
	for _, g := range games {
		if g.Status != "scheduled" {
			continue
		}
		qs, err := st.LatestOdds(ctx, g.ID)
		if err != nil {
			return err
		}
		h := r.Find(qs)
		if h == nil {
			continue
		}
		seen++
		ok, err := st.RecordHint(ctx, g.ID, hint.Version, *h, now)
		if err != nil {
			return err
		}
		if ok {
			recorded++
			slog.Info("hint", "game", g.ID, "match", g.HomeName+" - "+g.AwayName, "book", h.Book, "sel", h.Selection, "price", h.Price, "ev", h.EV)
		}
	}
	slog.Info("hints", "rule", hint.Version, "games", len(games), "qualifying", seen, "new", recorded)
	return nil
}
