package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lioilsources/BestBet/internal/apihockey"
	"github.com/lioilsources/BestBet/internal/config"
	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/odds"
	"github.com/lioilsources/BestBet/internal/oddspapi"
	"github.com/lioilsources/BestBet/internal/provmock"
	"github.com/lioilsources/BestBet/internal/store"
)

func selectLeagues(cfg config.File, slug string) ([]config.League, error) {
	if slug == "" {
		return cfg.Leagues, nil
	}
	l, ok := cfg.League(slug)
	if !ok {
		return nil, fmt.Errorf("unknown league %q", slug)
	}
	return []config.League{l}, nil
}

// mockBase is the in-process provider mock (MOCK_PROVIDERS=1), empty otherwise.
var mockBase string

// startMock serves provmock on a loopback port. Mock data must never reach the
// real database, so the database name has to end in _mock.
func startMock(dsn string, cfg config.File) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("DATABASE_URL: %w", err)
	}
	if !strings.HasSuffix(strings.TrimPrefix(u.Path, "/"), "_mock") {
		return fmt.Errorf("MOCK_PROVIDERS=1 needs a database whose name ends in _mock, got %q", u.Path)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	w := provmock.New(cfg, 1, time.Now)
	go func() { _ = http.Serve(ln, w.Handler()) }()
	mockBase = "http://" + ln.Addr().String()
	slog.Warn("MOCK PROVIDERS: API-HOCKEY and OddsPapi are simulated", "base", mockBase)
	return nil
}

func apiHockey(cacheDir string) (*apihockey.Client, error) {
	if mockBase != "" {
		c := apihockey.New("mock", &httpcache.Client{Dir: cacheDir + "/apihockey-mock", UserAgent: userAgent})
		c.Base = mockBase + "/apihockey"
		return c, nil
	}
	key := os.Getenv("APIHOCKEY_KEY")
	if key == "" {
		return nil, errors.New("APIHOCKEY_KEY is not set")
	}
	budget := 90
	if v := os.Getenv("APIHOCKEY_BUDGET"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("APIHOCKEY_BUDGET: %w", err)
		}
		budget = n
	}
	hc := &httpcache.Client{Dir: cacheDir + "/apihockey", MinGap: 7 * time.Second, Budget: budget, UserAgent: userAgent}
	return apihockey.New(key, hc), nil
}

func oddsPapi(cacheDir string) (*oddspapi.Client, error) {
	if mockBase != "" {
		return &oddspapi.Client{Base: mockBase + "/oddspapi", Key: "mock", HTTP: &httpcache.Client{Dir: cacheDir + "/oddspapi-mock", UserAgent: userAgent}}, nil
	}
	key := os.Getenv("ODDSPAPI_KEY")
	if key == "" {
		return nil, errors.New("ODDSPAPI_KEY is not set")
	}
	return &oddspapi.Client{Base: oddspapi.DefaultBase, Key: key, HTTP: &httpcache.Client{Dir: cacheDir + "/oddspapi", MinGap: 6 * time.Second, UserAgent: userAgent}}, nil
}

// refresh re-reads a league row: `leagues` may have just stored its API ids.
func refresh(ctx context.Context, st *store.Store, rows map[string]dbq.League, slug string) dbq.League {
	if r, err := st.Q.GetLeagueBySlug(ctx, slug); err == nil {
		return r
	}
	return rows[slug]
}

// currentSeason: a hockey season starting in August belongs to that year.
func currentSeason(now time.Time) int {
	if now.Month() >= time.July {
		return now.Year()
	}
	return now.Year() - 1
}

func oddsKind(now, start time.Time) string { return odds.SnapshotKind(now, start) }

func pgTime(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
