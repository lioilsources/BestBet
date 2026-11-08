-- name: UpsertLeague :one
INSERT INTO leagues (slug, name, country, role, apihockey_id, oddspapi_tournament_id)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    country = EXCLUDED.country,
    role = EXCLUDED.role,
    apihockey_id = COALESCE(EXCLUDED.apihockey_id, leagues.apihockey_id),
    oddspapi_tournament_id = COALESCE(EXCLUDED.oddspapi_tournament_id, leagues.oddspapi_tournament_id)
RETURNING *;

-- name: GetLeagueBySlug :one
SELECT * FROM leagues WHERE slug = $1;

-- name: ListLeagues :many
SELECT * FROM leagues ORDER BY id;

-- name: ListTeams :many
SELECT * FROM teams WHERE league_id = $1 ORDER BY id;

-- name: ListAliases :many
SELECT * FROM team_aliases WHERE league_id = $1;

-- name: InsertTeam :one
INSERT INTO teams (league_id, name, name_norm, apihockey_id, created_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (league_id, name_norm) DO UPDATE SET
    apihockey_id = COALESCE(teams.apihockey_id, EXCLUDED.apihockey_id)
RETURNING *;

-- name: UpsertAlias :exec
INSERT INTO team_aliases (source, league_id, alias_norm, team_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (source, league_id, alias_norm) DO UPDATE SET team_id = EXCLUDED.team_id;

-- name: UpsertGameApihockey :one
INSERT INTO games (
    league_id, season, start_at, status, home_team_id, away_team_id,
    home_score, away_score, home_60, away_60, period_scores,
    ot_home, ot_away, so_winner, decided_in, ot_type, phase, stage_raw, apihockey_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
)
ON CONFLICT (apihockey_id) DO UPDATE SET
    start_at = EXCLUDED.start_at,
    status = EXCLUDED.status,
    home_score = EXCLUDED.home_score,
    away_score = EXCLUDED.away_score,
    home_60 = EXCLUDED.home_60,
    away_60 = EXCLUDED.away_60,
    period_scores = EXCLUDED.period_scores,
    ot_home = EXCLUDED.ot_home,
    ot_away = EXCLUDED.ot_away,
    so_winner = EXCLUDED.so_winner,
    decided_in = EXCLUDED.decided_in,
    ot_type = EXCLUDED.ot_type,
    phase = EXCLUDED.phase,
    stage_raw = EXCLUDED.stage_raw,
    updated_at = now()
RETURNING id;

-- name: GetGameBySwehockeyID :one
SELECT * FROM games WHERE swehockey_id = $1;

-- name: FindGameByTeams :many
-- Candidate games of the same pairing within a time window (sources disagree on kick-off and time zone).
SELECT * FROM games
WHERE league_id = $1 AND home_team_id = $2 AND away_team_id = $3
  AND start_at BETWEEN sqlc.arg(from_ts)::timestamptz AND sqlc.arg(to_ts)::timestamptz
ORDER BY start_at;

-- name: AttachSwehockey :exec
-- Secondary source. If API-HOCKEY owns the game (apihockey_id set) only empty
-- fields are filled; otherwise swehockey owns it and overwrites (scheduled -> finished).
UPDATE games SET
    swehockey_id = $2,
    status = CASE WHEN apihockey_id IS NULL THEN sqlc.arg(status)::text ELSE status END,
    start_at = CASE WHEN apihockey_id IS NULL THEN sqlc.arg(start_at)::timestamptz ELSE start_at END,
    home_score = CASE WHEN apihockey_id IS NULL THEN $3 ELSE COALESCE(home_score, $3) END,
    away_score = CASE WHEN apihockey_id IS NULL THEN $4 ELSE COALESCE(away_score, $4) END,
    home_60 = CASE WHEN apihockey_id IS NULL THEN $5 ELSE COALESCE(home_60, $5) END,
    away_60 = CASE WHEN apihockey_id IS NULL THEN $6 ELSE COALESCE(away_60, $6) END,
    period_scores = CASE WHEN apihockey_id IS NULL THEN $7 ELSE COALESCE(period_scores, $7) END,
    ot_home = CASE WHEN apihockey_id IS NULL THEN $8 ELSE COALESCE(ot_home, $8) END,
    ot_away = CASE WHEN apihockey_id IS NULL THEN $9 ELSE COALESCE(ot_away, $9) END,
    so_winner = CASE WHEN apihockey_id IS NULL THEN $10 ELSE COALESCE(so_winner, $10) END,
    decided_in = CASE WHEN apihockey_id IS NULL THEN $11 ELSE COALESCE(decided_in, $11) END,
    ot_type = CASE WHEN apihockey_id IS NULL THEN sqlc.narg(ot_type) ELSE COALESCE(ot_type, sqlc.narg(ot_type)) END,
    updated_at = now()
WHERE id = $1;

-- name: InsertGameSwehockey :one
INSERT INTO games (
    league_id, season, start_at, status, home_team_id, away_team_id,
    home_score, away_score, home_60, away_60, period_scores,
    ot_home, ot_away, so_winner, decided_in, ot_type, phase, stage_raw, swehockey_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
)
RETURNING id;

-- name: ListSeriesGames :many
SELECT id, home_team_id, away_team_id, start_at, home_score, away_score, status
FROM games
WHERE league_id = $1 AND season = $2 AND phase IN ('playoff', 'relegation_playoff', 'promotion_qual')
ORDER BY start_at, id;

-- name: SetSeries :exec
UPDATE games SET series_id = $2, game_no = $3, series_score_before = $4 WHERE id = $1;

-- name: ListGamesStartingBetween :many
SELECT g.*, l.slug AS league_slug
FROM games g JOIN leagues l ON l.id = g.league_id
WHERE g.start_at BETWEEN sqlc.arg(from_ts)::timestamptz AND sqlc.arg(to_ts)::timestamptz
  AND g.status IN ('scheduled', 'live')
ORDER BY g.start_at;

-- name: UpsertGoalie :exec
INSERT INTO goalies (
    game_id, team_id, side, player_name, jersey, starter, shots_against, saves,
    goals_against, save_pct, toi_seconds, confirm_status, confirmed_at, source
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (game_id, side, player_name, source) DO UPDATE SET
    starter = EXCLUDED.starter,
    shots_against = EXCLUDED.shots_against,
    saves = EXCLUDED.saves,
    goals_against = EXCLUDED.goals_against,
    save_pct = EXCLUDED.save_pct,
    toi_seconds = EXCLUDED.toi_seconds,
    confirm_status = EXCLUDED.confirm_status,
    confirmed_at = EXCLUDED.confirmed_at;

-- name: InsertOdds :exec
INSERT INTO odds_snapshots (
    game_id, source, source_fixture_id, book, market, selection, line, price,
    raw_market, snapshot_kind, captured_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT DO NOTHING;

-- InsertOddsBatch is InsertOdds for whole odds histories: one round trip per
-- fixture instead of one per price point (the DB sits on JODA, over the LAN).
-- name: InsertOddsBatch :batchexec
INSERT INTO odds_snapshots (
    game_id, source, source_fixture_id, book, market, selection, line, price,
    raw_market, snapshot_kind, captured_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT DO NOTHING;

-- name: UpsertFixtureLink :exec
INSERT INTO fixture_links (source, source_fixture_id, league_id, game_id, match_method, detail)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (source, source_fixture_id) DO UPDATE SET
    game_id = EXCLUDED.game_id,
    match_method = EXCLUDED.match_method,
    detail = EXCLUDED.detail,
    seen_at = now();

-- name: SetTeamApihockeyID :exec
UPDATE teams SET apihockey_id = $2 WHERE id = $1 AND apihockey_id IS NULL;

-- name: SetGameApihockeyID :exec
UPDATE games SET apihockey_id = $2 WHERE id = $1 AND apihockey_id IS NULL;

-- name: GetGameByApihockeyID :one
SELECT * FROM games WHERE apihockey_id = $1;

-- name: HasSnapshot :one
SELECT EXISTS (
    SELECT 1 FROM odds_snapshots WHERE game_id = $1 AND snapshot_kind = $2
)::bool AS present;

-- name: GetGameByNhlID :one
SELECT * FROM games WHERE nhl_id = $1;

-- name: AttachNHL :exec
-- Same rules as AttachSwehockey: API-HOCKEY owns a game when it has its id.
UPDATE games SET
    nhl_id = $2,
    status = CASE WHEN apihockey_id IS NULL THEN sqlc.arg(status)::text ELSE status END,
    start_at = CASE WHEN apihockey_id IS NULL THEN sqlc.arg(start_at)::timestamptz ELSE start_at END,
    home_score = CASE WHEN apihockey_id IS NULL THEN $3 ELSE COALESCE(home_score, $3) END,
    away_score = CASE WHEN apihockey_id IS NULL THEN $4 ELSE COALESCE(away_score, $4) END,
    home_60 = CASE WHEN apihockey_id IS NULL THEN $5 ELSE COALESCE(home_60, $5) END,
    away_60 = CASE WHEN apihockey_id IS NULL THEN $6 ELSE COALESCE(away_60, $6) END,
    period_scores = CASE WHEN apihockey_id IS NULL THEN $7 ELSE COALESCE(period_scores, $7) END,
    ot_home = CASE WHEN apihockey_id IS NULL THEN $8 ELSE COALESCE(ot_home, $8) END,
    ot_away = CASE WHEN apihockey_id IS NULL THEN $9 ELSE COALESCE(ot_away, $9) END,
    so_winner = CASE WHEN apihockey_id IS NULL THEN $10 ELSE COALESCE(so_winner, $10) END,
    decided_in = CASE WHEN apihockey_id IS NULL THEN $11 ELSE COALESCE(decided_in, $11) END,
    ot_type = CASE WHEN apihockey_id IS NULL THEN sqlc.narg(ot_type) ELSE COALESCE(ot_type, sqlc.narg(ot_type)) END,
    phase = CASE WHEN apihockey_id IS NULL THEN sqlc.arg(phase)::text ELSE phase END,
    updated_at = now()
WHERE id = $1;

-- name: InsertGameNHL :one
INSERT INTO games (
    league_id, season, start_at, status, home_team_id, away_team_id,
    home_score, away_score, home_60, away_60, period_scores,
    ot_home, ot_away, so_winner, decided_in, ot_type, phase, stage_raw, nhl_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
)
RETURNING id;

-- name: ListNHLFinishedGames :many
SELECT g.id, g.nhl_id, g.home_team_id, g.away_team_id, g.start_at
FROM games g
JOIN leagues l ON l.id = g.league_id
WHERE l.slug = $1 AND g.nhl_id IS NOT NULL AND g.status = 'finished'
ORDER BY g.start_at, g.id;

-- name: GetCoachByName :one
SELECT id FROM coaches WHERE name = $1 ORDER BY id LIMIT 1;

-- name: InsertCoach :one
INSERT INTO coaches (name) VALUES ($1) RETURNING id;

-- name: DeleteLeagueCoachTenures :exec
DELETE FROM coach_tenures ct
USING teams t
WHERE t.id = ct.team_id AND t.league_id = $1 AND ct.source = $2;

-- name: InsertCoachTenure :exec
INSERT INTO coach_tenures (coach_id, team_id, role, from_date, to_date, source)
VALUES ($1, $2, 'head', $3, $4, $5)
ON CONFLICT DO NOTHING;
