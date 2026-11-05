// Package store wraps the generated queries with the merge rules between sources:
// API-HOCKEY is primary for games, swehockey fills gaps and adds goalies, odds
// feeds only attach to existing games.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // migrate driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lioilsources/BetWave/internal/config"
	"github.com/lioilsources/BetWave/internal/db"
	"github.com/lioilsources/BetWave/internal/db/dbq"
	"github.com/lioilsources/BetWave/internal/hockey"
	"github.com/lioilsources/BetWave/internal/teammatch"
)

// Store is the DB handle.
type Store struct {
	Pool *pgxpool.Pool
	Q    *dbq.Queries
}

// Open connects to Postgres.
func Open(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{Pool: pool, Q: dbq.New(pool)}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.Pool.Close() }

// Migrate applies all up migrations. dsn uses the postgres:// scheme.
func Migrate(dsn string) error {
	src, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5"+dsn[len("postgres"):])
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// SyncLeagues upserts configured leagues and returns slug -> row.
func (s *Store) SyncLeagues(ctx context.Context, cfg config.File) (map[string]dbq.League, error) {
	out := map[string]dbq.League{}
	for _, l := range cfg.Leagues {
		row, err := s.Q.UpsertLeague(ctx, dbq.UpsertLeagueParams{
			Slug: l.Slug, Name: l.Name, Country: l.Country, Role: l.Role,
			ApihockeyID:          nonZero(l.APIHockey.ID),
			OddspapiTournamentID: nonZero(l.OddsPapi.TournamentID),
		})
		if err != nil {
			return nil, fmt.Errorf("league %s: %w", l.Slug, err)
		}
		out[l.Slug] = row
	}
	return out, nil
}

// Teams resolves team names of one league for one source.
type Teams struct {
	s        *Store
	leagueID int32
	source   string
	m        *teammatch.Matcher
	byAPI    map[int32]int32
	// Unmatched names created as new teams in this run (for the report/log).
	Created []string
}

// LoadTeams prepares a resolver. cfgAliases are alias -> canonical name for this source+league.
func (s *Store) LoadTeams(ctx context.Context, leagueID int32, source string, cfgAliases map[string]string) (*Teams, error) {
	rows, err := s.Q.ListTeams(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	var teams []teammatch.Team
	byAPI := map[int32]int32{}
	byNorm := map[string]int32{}
	for _, r := range rows {
		teams = append(teams, teammatch.Team{ID: r.ID, Name: r.Name, Norm: r.NameNorm})
		byNorm[r.NameNorm] = r.ID
		if r.ApihockeyID != nil {
			byAPI[*r.ApihockeyID] = r.ID
		}
	}
	aliases := map[string]int32{}
	dbAliases, err := s.Q.ListAliases(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("list aliases: %w", err)
	}
	for _, a := range dbAliases {
		if a.Source == source {
			aliases[a.AliasNorm] = a.TeamID
		}
	}
	for alias, canonical := range cfgAliases {
		if id, ok := byNorm[teammatch.Norm(canonical)]; ok {
			aliases[teammatch.Norm(alias)] = id
		}
	}
	return &Teams{s: s, leagueID: leagueID, source: source, m: teammatch.New(teams, aliases), byAPI: byAPI}, nil
}

// Resolve returns a team id for a name, creating the team when nothing matches.
// apiID is set only by API-HOCKEY (0 otherwise).
func (t *Teams) Resolve(ctx context.Context, name string, apiID int32) (int32, error) {
	if apiID != 0 {
		if id, ok := t.byAPI[apiID]; ok {
			return id, nil
		}
	}
	id, method := t.m.Match(name)
	if method == teammatch.NoMatch {
		row, err := t.s.Q.InsertTeam(ctx, dbq.InsertTeamParams{
			LeagueID: t.leagueID, Name: name, NameNorm: teammatch.Norm(name),
			ApihockeyID: nonZero(int(apiID)), CreatedBy: t.source,
		})
		if err != nil {
			return 0, fmt.Errorf("insert team %q: %w", name, err)
		}
		t.m.Add(teammatch.Team{ID: row.ID, Name: row.Name, Norm: row.NameNorm})
		t.Created = append(t.Created, name)
		id = row.ID
	} else if method == teammatch.ByFuzzy {
		// Remember the spelling so the next run is exact and auditable.
		if err := t.s.Q.UpsertAlias(ctx, dbq.UpsertAliasParams{Source: t.source, LeagueID: t.leagueID, AliasNorm: teammatch.Norm(name), TeamID: id}); err != nil {
			return 0, fmt.Errorf("alias %q: %w", name, err)
		}
	}
	if apiID != 0 {
		if err := t.s.Q.SetTeamApihockeyID(ctx, dbq.SetTeamApihockeyIDParams{ID: id, ApihockeyID: &apiID}); err != nil {
			return 0, fmt.Errorf("set api id: %w", err)
		}
		t.byAPI[apiID] = id
	}
	return id, nil
}

// GameInput is a source-independent game.
type GameInput struct {
	LeagueID   int32
	Season     int32
	StartAt    time.Time
	Status     string
	HomeTeamID int32
	AwayTeamID int32
	Result     *hockey.Result
	Phase      string
	StageRaw   string
	OTType     string
}

// MatchWindow is how far kick-off times of two sources may differ (time zones,
// postponements announced late).
const MatchWindow = 36 * time.Hour

// FindGame finds the closest game of the same pairing within MatchWindow.
func (s *Store) FindGame(ctx context.Context, leagueID, home, away int32, at time.Time) (*dbq.Game, error) {
	rows, err := s.Q.FindGameByTeams(ctx, dbq.FindGameByTeamsParams{
		LeagueID: leagueID, HomeTeamID: home, AwayTeamID: away,
		FromTs: ts(at.Add(-MatchWindow)), ToTs: ts(at.Add(MatchWindow)),
	})
	if err != nil {
		return nil, fmt.Errorf("find game: %w", err)
	}
	var best *dbq.Game
	for i := range rows {
		if best == nil || absDur(rows[i].StartAt.Time.Sub(at)) < absDur(best.StartAt.Time.Sub(at)) {
			best = &rows[i]
		}
	}
	return best, nil
}

// UpsertAPIHockeyGame stores a primary-source game. A game created earlier by a
// secondary source for the same pairing gets the API id and is then overwritten.
// Returns the game id and how it was linked.
func (s *Store) UpsertAPIHockeyGame(ctx context.Context, apiID int32, in GameInput) (int64, string, error) {
	method := "external_id"
	if _, err := s.Q.GetGameByApihockeyID(ctx, &apiID); errors.Is(err, pgx.ErrNoRows) {
		method = "new"
		g, err := s.FindGame(ctx, in.LeagueID, in.HomeTeamID, in.AwayTeamID, in.StartAt)
		if err != nil {
			return 0, "", err
		}
		if g != nil && g.ApihockeyID == nil {
			if err := s.Q.SetGameApihockeyID(ctx, dbq.SetGameApihockeyIDParams{ID: g.ID, ApihockeyID: &apiID}); err != nil {
				return 0, "", fmt.Errorf("attach api id: %w", err)
			}
			method = "date_teams"
		}
	} else if err != nil {
		return 0, "", fmt.Errorf("get game: %w", err)
	}
	p := dbq.UpsertGameApihockeyParams{
		LeagueID: in.LeagueID, Season: in.Season, StartAt: ts(in.StartAt), Status: in.Status,
		HomeTeamID: in.HomeTeamID, AwayTeamID: in.AwayTeamID, Phase: in.Phase,
		StageRaw: strPtr(in.StageRaw), ApihockeyID: &apiID,
	}
	if err := fillResult(in, &p.HomeScore, &p.AwayScore, &p.Home60, &p.Away60, &p.PeriodScores, &p.OtHome, &p.OtAway, &p.SoWinner, &p.DecidedIn, &p.OtType); err != nil {
		return 0, "", err
	}
	id, err := s.Q.UpsertGameApihockey(ctx, p)
	if err != nil {
		return 0, "", fmt.Errorf("upsert game %d: %w", apiID, err)
	}
	return id, method, nil
}

// UpsertSwehockeyGame attaches a swehockey game to an existing game or creates it.
func (s *Store) UpsertSwehockeyGame(ctx context.Context, sweID int32, in GameInput) (int64, string, error) {
	if g, err := s.Q.GetGameBySwehockeyID(ctx, &sweID); err == nil {
		return g.ID, "external_id", s.attachSwe(ctx, g.ID, sweID, in)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, "", fmt.Errorf("get game: %w", err)
	}
	g, err := s.FindGame(ctx, in.LeagueID, in.HomeTeamID, in.AwayTeamID, in.StartAt)
	if err != nil {
		return 0, "", err
	}
	if g != nil && g.SwehockeyID == nil {
		return g.ID, "date_teams", s.attachSwe(ctx, g.ID, sweID, in)
	}
	p := dbq.InsertGameSwehockeyParams{
		LeagueID: in.LeagueID, Season: in.Season, StartAt: ts(in.StartAt), Status: in.Status,
		HomeTeamID: in.HomeTeamID, AwayTeamID: in.AwayTeamID, Phase: in.Phase,
		StageRaw: strPtr(in.StageRaw), SwehockeyID: &sweID,
	}
	if err := fillResult(in, &p.HomeScore, &p.AwayScore, &p.Home60, &p.Away60, &p.PeriodScores, &p.OtHome, &p.OtAway, &p.SoWinner, &p.DecidedIn, &p.OtType); err != nil {
		return 0, "", err
	}
	id, err := s.Q.InsertGameSwehockey(ctx, p)
	if err != nil {
		return 0, "", fmt.Errorf("insert swehockey game %d: %w", sweID, err)
	}
	return id, "new", nil
}

func (s *Store) attachSwe(ctx context.Context, gameID int64, sweID int32, in GameInput) error {
	p := dbq.AttachSwehockeyParams{ID: gameID, SwehockeyID: &sweID, Status: in.Status, StartAt: ts(in.StartAt)}
	if err := fillResult(in, &p.HomeScore, &p.AwayScore, &p.Home60, &p.Away60, &p.PeriodScores, &p.OtHome, &p.OtAway, &p.SoWinner, &p.DecidedIn, &p.OtType); err != nil {
		return err
	}
	if err := s.Q.AttachSwehockey(ctx, p); err != nil {
		return fmt.Errorf("attach swehockey %d: %w", sweID, err)
	}
	return nil
}

func fillResult(in GameInput, hs, as, h60, a60 **int32, periods *[]byte, otH, otA **int32, so, decided, otType **string) error {
	r := in.Result
	if r == nil {
		return nil
	}
	*hs, *as = i32(r.Final.Home), i32(r.Final.Away)
	*h60, *a60 = i32(r.Reg.Home), i32(r.Reg.Away)
	ps := make([][2]int, len(r.Periods))
	for i, p := range r.Periods {
		ps[i] = [2]int{p.Home, p.Away}
	}
	b, err := json.Marshal(ps)
	if err != nil {
		return fmt.Errorf("periods: %w", err)
	}
	*periods = b
	if r.OT != nil {
		*otH, *otA = i32(r.OT.Home), i32(r.OT.Away)
	}
	*so = strPtr(r.SOWinner)
	*decided = strPtr(r.DecidedIn)
	*otType = strPtr(in.OTType)
	return nil
}

// Link records the outcome of attaching an external fixture.
func (s *Store) Link(ctx context.Context, source, fixtureID string, leagueID int32, gameID int64, method, detail string) error {
	var gid *int64
	if gameID != 0 {
		gid = &gameID
	}
	if err := s.Q.UpsertFixtureLink(ctx, dbq.UpsertFixtureLinkParams{
		Source: source, SourceFixtureID: fixtureID, LeagueID: leagueID, GameID: gid,
		MatchMethod: method, Detail: strPtr(detail),
	}); err != nil {
		return fmt.Errorf("link %s/%s: %w", source, fixtureID, err)
	}
	return nil
}

// DeriveSeries recomputes series_id/game_no/series_score_before for a league season.
func (s *Store) DeriveSeries(ctx context.Context, league dbq.League, season int32) (int, error) {
	rows, err := s.Q.ListSeriesGames(ctx, dbq.ListSeriesGamesParams{LeagueID: league.ID, Season: season})
	if err != nil {
		return 0, fmt.Errorf("list series games: %w", err)
	}
	in := make([]hockey.SeriesGame, len(rows))
	for i, r := range rows {
		in[i] = hockey.SeriesGame{ID: r.ID, HomeTeamID: r.HomeTeamID, AwayTeamID: r.AwayTeamID, StartAt: r.StartAt.Time, HomeScore: r.HomeScore, AwayScore: r.AwayScore}
	}
	states := hockey.DeriveSeries(fmt.Sprintf("%s:%d", league.Slug, season), in, 10*24*time.Hour)
	for id, st := range states {
		if err := s.Q.SetSeries(ctx, dbq.SetSeriesParams{ID: id, SeriesID: &st.SeriesID, GameNo: i32(st.GameNo), SeriesScoreBefore: &st.ScoreBefore}); err != nil {
			return 0, fmt.Errorf("set series: %w", err)
		}
	}
	return len(states), nil
}

func ts(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func i32(v int) *int32 { x := int32(v); return &x }

func nonZero(v int) *int32 {
	if v == 0 {
		return nil
	}
	return i32(v)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// Lookup matches a name without creating a team.
func (t *Teams) Lookup(name string) (int32, bool) {
	id, method := t.m.Match(name)
	return id, method != teammatch.NoMatch
}
