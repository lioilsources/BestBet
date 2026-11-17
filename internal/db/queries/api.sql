-- name: InsertPrediction :exec
-- Same inputs → same row: re-running the cron is idempotent.
INSERT INTO predictions (game_id, kind, model_version, inputs_hash, computed_at, lambda_home, lambda_away,
                         score_matrix, markets, contributions)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (game_id, kind, model_version, inputs_hash) DO NOTHING;

-- name: ListGamesBetween :many
SELECT g.id, l.slug AS league_slug, l.name AS league_name, g.season, g.start_at, g.status, g.phase,
       g.home_team_id, ht.name AS home_name, g.away_team_id, at.name AS away_name,
       g.home_60, g.away_60, g.home_score, g.away_score, g.decided_in
FROM games g
JOIN leagues l ON l.id = g.league_id
JOIN teams ht ON ht.id = g.home_team_id
JOIN teams at ON at.id = g.away_team_id
WHERE g.start_at >= sqlc.arg(from_ts) AND g.start_at < sqlc.arg(to_ts)
ORDER BY g.start_at, g.id;

-- name: GetGame :one
SELECT g.id, l.slug AS league_slug, l.name AS league_name, g.season, g.start_at, g.status, g.phase,
       g.home_team_id, ht.name AS home_name, g.away_team_id, at.name AS away_name,
       g.home_60, g.away_60, g.home_score, g.away_score, g.decided_in
FROM games g
JOIN leagues l ON l.id = g.league_id
JOIN teams ht ON ht.id = g.home_team_id
JOIN teams at ON at.id = g.away_team_id
WHERE g.id = $1;

-- name: LatestPrediction :one
SELECT game_id, model_version, inputs_hash, computed_at, lambda_home, lambda_away, score_matrix, markets, contributions
FROM predictions
WHERE game_id = $1 AND kind = 'live'
ORDER BY computed_at DESC, id DESC
LIMIT 1;

-- name: LatestOdds :many
-- Latest pre-start price per book, market, selection and line.
SELECT DISTINCT ON (o.book, o.market, o.selection, o.line)
       o.book, o.market, o.selection, o.line, o.price, o.captured_at
FROM odds_snapshots o
JOIN games g ON g.id = o.game_id
WHERE o.game_id = $1 AND o.captured_at < g.start_at
ORDER BY o.book, o.market, o.selection, o.line, o.captured_at DESC;

-- name: ListJournal :many
SELECT j.id, j.game_id, j.book, j.market, j.selection, j.line, j.price, j.stake, j.placed_at,
       j.model_p, j.closing_price, j.result, j.note
FROM journal j
WHERE j.user_key = $1
ORDER BY j.placed_at DESC
LIMIT 500;

-- name: LastOddsCapture :one
-- Monitoring: newest odds capture per source (gaps over 24 h are an alert, 01 fáze 4).
SELECT COALESCE(max(captured_at), 'epoch'::timestamptz)::timestamptz AS last_capture FROM odds_snapshots WHERE source = $1;

-- name: InsertHint :execrows
-- First qualifying price per game and rule; later prices never overwrite it.
INSERT INTO hints (game_id, rule_version, book, market, selection, price, p_sharp, p_soft, captured_at, recorded_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (game_id, rule_version) DO NOTHING;

-- name: ListHints :many
SELECT h.id, h.game_id, h.rule_version, h.book, h.market, h.selection, h.price, h.p_sharp, h.p_soft,
       h.captured_at, h.recorded_at
FROM hints h
WHERE h.rule_version = $1
ORDER BY h.recorded_at DESC;

-- name: OddsHistory :many
-- Every stored pre-start price of a game (thinned kinds plus provisional reads).
SELECT o.book, o.market, o.selection, o.line, o.price, o.captured_at, o.snapshot_kind
FROM odds_snapshots o
JOIN games g ON g.id = o.game_id
WHERE o.game_id = $1 AND o.captured_at < g.start_at
ORDER BY o.captured_at, o.book, o.market, o.selection;

-- name: CoverageGames :many
SELECT l.slug, l.name, g.season, count(*) AS games,
       count(*) FILTER (WHERE g.status = 'finished') AS finished,
       count(*) FILTER (WHERE g.status = 'scheduled' AND g.start_at > now()) AS upcoming,
       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM goalies x WHERE x.game_id = g.id AND x.starter)) AS with_goalies,
       count(*) FILTER (WHERE EXISTS (SELECT 1 FROM odds_snapshots o WHERE o.game_id = g.id)) AS with_odds,
       min(g.start_at)::timestamptz AS first_at, max(g.start_at)::timestamptz AS last_at
FROM games g
JOIN leagues l ON l.id = g.league_id
GROUP BY l.slug, l.name, g.season
ORDER BY l.slug, g.season;

-- name: CoverageOdds :many
SELECT l.slug, o.source, o.book, o.snapshot_kind, count(*) AS quotes, count(DISTINCT o.game_id) AS games,
       max(o.captured_at)::timestamptz AS last_capture
FROM odds_snapshots o
JOIN games g ON g.id = o.game_id
JOIN leagues l ON l.id = g.league_id
WHERE o.market = '1x2'
GROUP BY l.slug, o.source, o.book, o.snapshot_kind
ORDER BY l.slug, o.source, o.book, o.snapshot_kind;

-- name: CoverageCoaches :many
SELECT l.slug, count(*) AS tenures, count(DISTINCT t.coach_id) AS coaches
FROM coach_tenures t
JOIN teams tm ON tm.id = t.team_id
JOIN leagues l ON l.id = tm.league_id
GROUP BY l.slug;

-- name: CoveragePredictions :many
SELECT model_version, count(*) AS predictions, max(computed_at)::timestamptz AS last_at
FROM predictions
WHERE kind = 'live'
GROUP BY model_version
ORDER BY model_version;

-- name: CoveragePlayers :many
SELECT l.slug, g.season, count(DISTINCT pm.game_id) AS games, count(*) AS lines,
       count(DISTINCT pm.player_id) AS players
FROM player_match_stats pm
JOIN games g ON g.id = pm.game_id
JOIN leagues l ON l.id = g.league_id
GROUP BY l.slug, g.season
ORDER BY l.slug, g.season;
