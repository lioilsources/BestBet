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
