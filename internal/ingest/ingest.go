// Package ingest moves data from sources into the store.
package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/lioilsources/BestBet/internal/apihockey"
	"github.com/lioilsources/BestBet/internal/config"
	"github.com/lioilsources/BestBet/internal/db/dbq"
	"github.com/lioilsources/BestBet/internal/hockey"
	"github.com/lioilsources/BestBet/internal/httpcache"
	"github.com/lioilsources/BestBet/internal/odds"
	"github.com/lioilsources/BestBet/internal/oddspapi"
	"github.com/lioilsources/BestBet/internal/store"
	"github.com/lioilsources/BestBet/internal/swehockey"
	"github.com/lioilsources/BestBet/internal/teammatch"
)

// Env is what every ingest step needs.
type Env struct {
	Store *store.Store
	Cfg   config.File
	Now   func() time.Time
}

func (e Env) aliases(source, slug string) map[string]string {
	return e.Cfg.Aliases[source][slug]
}

// Stats counts what one step did.
type Stats struct {
	Seen, New, Linked, Failed, Goalies, Quotes int
}

func (s Stats) String() string {
	return fmt.Sprintf("seen=%d new=%d linked=%d failed=%d goalies=%d quotes=%d", s.Seen, s.New, s.Linked, s.Failed, s.Goalies, s.Quotes)
}

// ResolveAPIHockeyLeague finds a configured league in the full API-HOCKEY
// league list (one cached request for all leagues; the search parameter
// rejects names like "1. Liga"). It returns the leagues of the configured
// country as candidates; exactly one name match (search or alt) is best.
func ResolveAPIHockeyLeague(ctx context.Context, c *apihockey.Client, l config.League) (*apihockey.League, []apihockey.League, error) {
	names := append([]string{l.APIHockey.Search}, l.APIHockey.Alt...)
	list, err := c.Leagues(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	var all []apihockey.League
	for _, c := range list {
		if strings.EqualFold(teammatch.Norm(c.Country.Name), teammatch.Norm(l.APIHockey.Country)) {
			all = append(all, c)
		}
	}
	var best []apihockey.League
	for _, c := range all {
		if !strings.EqualFold(teammatch.Norm(c.Country.Name), teammatch.Norm(l.APIHockey.Country)) {
			continue
		}
		for _, n := range names {
			if strings.EqualFold(teammatch.Norm(c.Name), teammatch.Norm(n)) {
				best = append(best, c)
				break
			}
		}
	}
	if len(best) == 1 {
		return &best[0], all, nil
	}
	return nil, all, nil
}

// APIHockeyGames ingests one league season from API-HOCKEY.
func APIHockeyGames(ctx context.Context, e Env, c *apihockey.Client, l config.League, row dbq.League, season int) (Stats, error) {
	var st Stats
	if row.ApihockeyID == nil {
		return st, fmt.Errorf("league %s has no apihockey id; run `ingest-hockey leagues` first", l.Slug)
	}
	ttl := time.Duration(0)
	if seasonOngoing(season, e.Now()) {
		ttl = 6 * time.Hour
	}
	games, err := c.Games(ctx, int(*row.ApihockeyID), season, ttl)
	if err != nil {
		return st, err
	}
	teams, err := e.Store.LoadTeams(ctx, row.ID, "apihockey", e.aliases("apihockey", l.Slug))
	if err != nil {
		return st, err
	}
	for _, g := range games {
		st.Seen++
		home, err := teams.Resolve(ctx, g.Teams.Home.Name, int32(g.Teams.Home.ID))
		if err != nil {
			return st, err
		}
		away, err := teams.Resolve(ctx, g.Teams.Away.Name, int32(g.Teams.Away.ID))
		if err != nil {
			return st, err
		}
		start, err := g.StartAt()
		if err != nil {
			st.Failed++
			slog.Warn("skip game", "err", err)
			continue
		}
		stage := ""
		if g.Week != nil {
			stage = *g.Week
		}
		in := store.GameInput{
			LeagueID: row.ID, Season: int32(season), StartAt: start, Status: apihockey.Status(g.Status.Short),
			HomeTeamID: home, AwayTeamID: away, Phase: hockey.DetectPhase(stage), StageRaw: stage,
		}
		if in.Status == "finished" {
			r, err := g.Result()
			if err != nil {
				st.Failed++
				slog.Warn("bad result", "game", g.ID, "err", err)
			} else {
				in.Result = &r
				in.OTType = hockey.OTType(r, in.Phase, l.OT)
				if g.Scores.Home != nil && g.Scores.Away != nil && (*g.Scores.Home != r.Final.Home || *g.Scores.Away != r.Final.Away) {
					slog.Warn("final score differs from periods", "game", g.ID, "scores", fmt.Sprintf("%d-%d", *g.Scores.Home, *g.Scores.Away), "derived", fmt.Sprintf("%d-%d", r.Final.Home, r.Final.Away))
				}
			}
		}
		_, method, err := e.Store.UpsertAPIHockeyGame(ctx, int32(g.ID), in)
		if err != nil {
			return st, err
		}
		switch method {
		case "new":
			st.New++
		case "date_teams":
			st.Linked++
		}
	}
	if len(teams.Created) > 0 {
		slog.Info("teams created", "league", l.Slug, "names", teams.Created)
	}
	n, err := e.Store.DeriveSeries(ctx, row, int32(season))
	if err != nil {
		return st, err
	}
	slog.Info("api-hockey games", "league", l.Slug, "season", season, "stats", st.String(), "series_games", n)
	return st, nil
}

// APIHockeyOdds captures current pre-match odds of one league season. Run it on a
// schedule (06:00, T-24h, T-1h, closing); each run is a new snapshot.
func APIHockeyOdds(ctx context.Context, e Env, c *apihockey.Client, l config.League, row dbq.League, season int) (Stats, error) {
	var st Stats
	if row.ApihockeyID == nil {
		return st, fmt.Errorf("league %s has no apihockey id", l.Slug)
	}
	captured := e.Now().UTC().Truncate(time.Minute)
	for page := 1; ; page++ {
		items, total, err := c.Odds(ctx, int(*row.ApihockeyID), season, page)
		if err != nil {
			return st, err
		}
		for _, it := range items {
			st.Seen++
			fid := strconv.Itoa(it.Game.ID)
			apiID := int32(it.Game.ID)
			g, err := e.Store.Q.GetGameByApihockeyID(ctx, &apiID)
			var gameID int64
			kind := "adhoc"
			if err == nil {
				gameID = g.ID
				kind = odds.SnapshotKind(captured, g.StartAt.Time)
				st.Linked++
			} else {
				st.Failed++
			}
			method := "external_id"
			if gameID == 0 {
				method = "none"
			}
			if err := e.Store.Link(ctx, "apihockey_odds", fid, row.ID, gameID, method, ""); err != nil {
				return st, err
			}
			for _, bm := range it.Bookmakers {
				for _, bet := range bm.Bets {
					for _, v := range bet.Values {
						price, err := odds.ParsePrice(v.Odd)
						if err != nil {
							continue
						}
						q := odds.NormalizeAPIHockey(bet.Name, v.Value, price)
						if err := insertQuote(ctx, e.Store, gameID, "apihockey", fid, bm.Name, q, kind, captured); err != nil {
							return st, err
						}
						st.Quotes++
					}
				}
			}
		}
		if page >= total {
			break
		}
	}
	slog.Info("api-hockey odds", "league", l.Slug, "season", season, "stats", st.String())
	return st, nil
}

func quoteParams(gameID int64, source, fixtureID, book string, q odds.Quote, kind string, at time.Time) dbq.InsertOddsParams {
	var gid *int64
	if gameID != 0 {
		gid = &gameID
	}
	return dbq.InsertOddsParams{
		GameID: gid, Source: source, SourceFixtureID: fixtureID, Book: strings.ToLower(book),
		Market: q.Market, Selection: q.Selection, Line: q.Line, Price: q.Price,
		RawMarket: q.RawMarket, SnapshotKind: kind, CapturedAt: pgTime(at),
	}
}

func insertQuote(ctx context.Context, s *store.Store, gameID int64, source, fixtureID, book string, q odds.Quote, kind string, at time.Time) error {
	if err := s.Q.InsertOdds(ctx, quoteParams(gameID, source, fixtureID, book, q, kind, at)); err != nil {
		return fmt.Errorf("insert odds: %w", err)
	}
	return nil
}

// insertQuotes sends a fixture's odds history as one batch.
func insertQuotes(ctx context.Context, s *store.Store, rows []dbq.InsertOddsBatchParams) error {
	if len(rows) == 0 {
		return nil
	}
	var first error
	s.Q.InsertOddsBatch(ctx, rows).Exec(func(_ int, err error) {
		if err != nil && first == nil {
			first = err
		}
	})
	if first != nil {
		return fmt.Errorf("insert odds batch: %w", first)
	}
	return nil
}

// Swehockey ingests one schedule page and, when withGoalies, every played game's
// events page (one polite request per game, cached forever once final).
func Swehockey(ctx context.Context, e Env, hc *httpcache.Client, l config.League, row dbq.League, sch config.SwehockeySchedule, withGoalies bool) (Stats, error) {
	var st Stats
	loc, err := l.Location()
	if err != nil {
		return st, err
	}
	ongoing := seasonOngoing(sch.Season, e.Now())
	ttl := time.Duration(0)
	if ongoing {
		ttl = 6 * time.Hour
	}
	u := fmt.Sprintf("https://stats.swehockey.se/ScheduleAndResults/Schedule/%d", sch.ScheduleID)
	page, _, err := hc.Get(ctx, httpcache.Request{URL: u, Key: u, TTL: ttl})
	if err != nil {
		return st, err
	}
	phase := hockey.DetectPhase(sch.Stage)
	games, err := swehockey.ParseSchedule(page, loc, phase == hockey.PhaseRegular)
	if err != nil {
		return st, err
	}
	teams, err := e.Store.LoadTeams(ctx, row.ID, "swehockey", e.aliases("swehockey", l.Slug))
	if err != nil {
		return st, err
	}
	for _, g := range games {
		st.Seen++
		fid := strconv.Itoa(g.GameID)
		home, err := teams.Resolve(ctx, g.Home, 0)
		if err != nil {
			return st, err
		}
		away, err := teams.Resolve(ctx, g.Away, 0)
		if err != nil {
			return st, err
		}
		in := store.GameInput{
			LeagueID: row.ID, Season: int32(sch.Season), StartAt: g.StartAt, Status: "scheduled",
			HomeTeamID: home, AwayTeamID: away, Phase: phase, StageRaw: strings.TrimSuffix(sch.Stage+" / "+g.Label, " / "),
		}
		if g.Played {
			in.Status = "finished"
			if g.Result != nil {
				in.Result = g.Result
				in.OTType = hockey.OTType(*g.Result, phase, l.OT)
			} else {
				slog.Warn("swehockey result not parsed", "game", g.GameID, "err", g.ParseErr)
			}
		}
		gameID, method, err := e.Store.UpsertSwehockeyGame(ctx, int32(g.GameID), in)
		if err != nil {
			return st, err
		}
		switch method {
		case "new":
			st.New++
		default:
			st.Linked++
		}
		if err := e.Store.Link(ctx, "swehockey", fid, row.ID, gameID, method, g.Home+" - "+g.Away); err != nil {
			return st, err
		}
		if !withGoalies || !g.Played || !g.HasEvents {
			continue
		}
		n, err := swehockeyGoalies(ctx, e, hc, gameID, home, away, g.GameID)
		if errors.Is(err, httpcache.ErrBudget) {
			return st, err
		}
		if err != nil {
			st.Failed++
			slog.Warn("goalies", "game", g.GameID, "err", err)
			continue
		}
		st.Goalies += n
	}
	if len(teams.Created) > 0 {
		slog.Info("teams created", "league", l.Slug, "names", teams.Created)
	}
	if phase != hockey.PhaseRegular {
		if _, err := e.Store.DeriveSeries(ctx, row, int32(sch.Season)); err != nil {
			return st, err
		}
	}
	slog.Info("swehockey", "league", l.Slug, "schedule", sch.ScheduleID, "stats", st.String())
	return st, nil
}

func swehockeyGoalies(ctx context.Context, e Env, hc *httpcache.Client, gameID int64, home, away int32, sweID int) (int, error) {
	u := fmt.Sprintf("https://stats.swehockey.se/Game/Events/%d", sweID)
	page, _, err := hc.Get(ctx, httpcache.Request{URL: u, Key: u})
	if err != nil {
		return 0, err
	}
	ev, err := swehockey.ParseEvents(page)
	if err != nil {
		return 0, err
	}
	for _, g := range ev.Goalies {
		team := home
		if g.Side == "away" {
			team = away
		}
		pct := g.SavePct
		jersey := int32(g.Jersey)
		sa, sv, ga := int32(g.ShotsAgainst), int32(g.Saves), int32(g.GoalsAgainst)
		var toi *int32
		if g.TOISeconds > 0 {
			t := int32(g.TOISeconds)
			toi = &t
		}
		if err := e.Store.Q.UpsertGoalie(ctx, dbq.UpsertGoalieParams{
			GameID: gameID, TeamID: team, Side: g.Side, PlayerName: g.Name, Jersey: &jersey,
			Starter: g.Starter, ShotsAgainst: &sa, Saves: &sv, GoalsAgainst: &ga, SavePct: &pct,
			ToiSeconds: toi, ConfirmStatus: "CONFIRMED", Source: "swehockey",
		}); err != nil {
			return 0, fmt.Errorf("goalie: %w", err)
		}
	}
	return len(ev.Goalies), nil
}

// OddsPapi pulls fixtures of a tournament in [from, to] and the historical odds of
// each fixture for up to 3 books (endpoint cooldown 5 s, set via httpcache.MinGap).
func OddsPapi(ctx context.Context, e Env, c *oddspapi.Client, l config.League, row dbq.League, from, to time.Time, books []string) (Stats, error) {
	var st Stats
	if row.OddspapiTournamentID == nil {
		return st, fmt.Errorf("league %s has no oddspapi tournament id; run `ingest-hockey leagues`", l.Slug)
	}
	markets, err := c.Markets(ctx)
	if err != nil {
		return st, err
	}
	fixtures, err := c.Fixtures(ctx, int(*row.OddspapiTournamentID), from, to, 6*time.Hour)
	if err != nil {
		return st, err
	}
	teams, err := e.Store.LoadTeams(ctx, row.ID, "oddspapi", e.aliases("oddspapi", l.Slug))
	if err != nil {
		return st, err
	}
	for _, f := range fixtures {
		// The fixtures endpoint also returns older rescheduled games; stay in range.
		if !f.StartTime.IsZero() && (f.StartTime.Before(from) || f.StartTime.After(to.Add(24*time.Hour))) {
			continue
		}
		st.Seen++
		gameID, detail, err := linkByTeams(ctx, e.Store, teams, row.ID, f.Home, f.Away, f.StartTime)
		if err != nil {
			return st, err
		}
		method := "date_teams"
		if gameID == 0 {
			method = "none"
			st.Failed++
		} else {
			st.Linked++
		}
		if err := e.Store.Link(ctx, "oddspapi", f.FixtureID, row.ID, gameID, method, detail); err != nil {
			return st, err
		}
		finished := f.StartTime.Before(e.Now().Add(-6 * time.Hour))
		pts, err := c.HistoricalOdds(ctx, f.FixtureID, books, finished)
		if errors.Is(err, httpcache.ErrBudget) {
			return st, err
		}
		if err != nil {
			slog.Warn("historical odds", "fixture", f.FixtureID, "err", err)
			continue
		}
		// Only modelled markets, thinned to opening/t24h/t1h/closing per
		// selection (odds.Thin); the full history stays in the HTTP cache.
		qs := make([]odds.Quote, len(pts))
		var keep []odds.Point
		for i, p := range pts {
			def, ok := markets[p.MarketID]
			if !ok {
				def = oddspapi.MarketDef{Name: "market " + p.MarketID}
			}
			qs[i] = oddspapi.Normalize(def, p.OutcomeID, p.Price)
			if qs[i].Market == odds.Other {
				continue
			}
			line := ""
			if qs[i].Line != nil {
				line = strconv.FormatFloat(*qs[i].Line, 'f', -1, 64)
			}
			keep = append(keep, odds.Point{Key: p.Book + "|" + qs[i].Market + "|" + qs[i].Selection + "|" + line, At: p.CreatedAt, I: i})
		}
		kinds := odds.Thin(keep, f.StartTime)
		rows := make([]dbq.InsertOddsBatchParams, 0, len(kinds))
		for i, kind := range kinds {
			p := pts[i]
			rows = append(rows, dbq.InsertOddsBatchParams(quoteParams(gameID, "oddspapi", f.FixtureID, p.Book, qs[i], kind, p.CreatedAt)))
		}
		if err := insertQuotes(ctx, e.Store, rows); err != nil {
			return st, err
		}
		st.Quotes += len(rows)
	}
	slog.Info("oddspapi", "league", l.Slug, "stats", st.String())
	return st, nil
}

// linkByTeams matches a secondary fixture to a game without creating teams: a
// name we do not know is a failed join, reported for an alias.
func linkByTeams(ctx context.Context, s *store.Store, teams *store.Teams, leagueID int32, home, away string, at time.Time) (int64, string, error) {
	h, okH := teams.Lookup(home)
	a, okA := teams.Lookup(away)
	detail := home + " - " + away
	if !okH || !okA {
		return 0, "unknown team: " + detail, nil
	}
	g, err := s.FindGame(ctx, leagueID, h, a, at)
	if err != nil {
		return 0, "", err
	}
	if g == nil {
		return 0, "no game in window: " + detail, nil
	}
	return g.ID, detail, nil
}

// seasonOngoing: hockey seasons run Aug..Jun of start year +1.
func seasonOngoing(season int, now time.Time) bool {
	end := time.Date(season+1, time.July, 1, 0, 0, 0, 0, time.UTC)
	start := time.Date(season, time.August, 1, 0, 0, 0, 0, time.UTC)
	return now.After(start) && now.Before(end)
}

func pgTime(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }
