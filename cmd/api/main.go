// Command api serves the read-only API (plan 01 fáze 4, internal/api).
//
// Environment: DATABASE_URL, API_ADDR (127.0.0.1:8096), API_BOOKS (allowed CZ
// books, comma list), API_CLV_MARKETS (markets with proven CLV; empty = nothing
// is green), API_TOKENS ("user:token,…" for /journal), API_HELP (help contact
// shown with the § 5j warning), API_TZ (Europe/Prague).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/lioilsources/BestBet/internal/api"
	"github.com/lioilsources/BestBet/internal/canon"
	"github.com/lioilsources/BestBet/internal/edge"
	"github.com/lioilsources/BestBet/internal/store"
)

// source adapts store errors to the API.
type source struct{ *store.Store }

func (s source) Game(ctx context.Context, id int64) (canon.GameInfo, error) {
	g, err := s.Store.Game(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return g, api.ErrNotFound
	}
	return g, err
}

func main() {
	if err := run(); err != nil {
		slog.Error("api failed", "err", err)
		os.Exit(1)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func set(list string) map[string]bool {
	out := map[string]bool{}
	for _, x := range strings.Split(list, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out[x] = true
		}
	}
	return out
}

func run() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}
	loc, err := time.LoadLocation(env("API_TZ", "Europe/Prague"))
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer st.Close()
	tokens := map[string]string{}
	for _, kv := range strings.Split(os.Getenv("API_TOKENS"), ",") {
		if u, t, ok := strings.Cut(strings.TrimSpace(kv), ":"); ok && u != "" && t != "" {
			tokens[t] = u
		}
	}
	pol := edge.DefaultPolicy()
	pol.CLVMarkets = set(os.Getenv("API_CLV_MARKETS"))
	srv := &api.Server{Src: source{st}, Now: time.Now, Books: set(env("API_BOOKS", "tipsport,fortuna,chance,betano,synottip,sazka,merkur")),
		Policy: pol, Tokens: tokens, HelpContact: os.Getenv("API_HELP"), Loc: loc}
	hs := &http.Server{Addr: env("API_ADDR", "127.0.0.1:8096"), Handler: srv.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		sh, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = hs.Shutdown(sh)
	}()
	slog.Info("api listening", "addr", hs.Addr)
	if err := hs.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
