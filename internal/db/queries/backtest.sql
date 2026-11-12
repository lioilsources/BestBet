-- name: ListLeagueGames :many
SELECT g.id, g.season, g.start_at, g.home_team_id, g.away_team_id, g.phase, g.status,
       g.period_scores, g.home_60, g.away_60, g.home_score, g.away_score, g.decided_in
FROM games g
JOIN leagues l ON l.id = g.league_id
WHERE l.slug = $1
ORDER BY g.start_at, g.id;

-- name: ListLeagueOdds :many
-- Pre-start prices only: a capture after the start is never a closing price.
SELECT o.game_id, o.book, o.market, o.selection, o.line, o.price, o.captured_at, o.snapshot_kind
FROM odds_snapshots o
JOIN games g ON g.id = o.game_id
JOIN leagues l ON l.id = g.league_id
WHERE l.slug = $1 AND o.captured_at < g.start_at AND o.market IN ('1x2', 'ml')
ORDER BY o.game_id, o.captured_at;

-- name: ListLeagueGoalies :many
SELECT gl.game_id, gl.side, gl.player_name, gl.starter, gl.shots_against, gl.saves, gl.toi_seconds, gl.confirm_status
FROM goalies gl
JOIN games g ON g.id = gl.game_id
JOIN leagues l ON l.id = g.league_id
WHERE l.slug = $1
ORDER BY gl.game_id, gl.side, gl.starter DESC;

-- name: ListLeagueCoachTenures :many
SELECT ct.team_id, c.name, ct.from_date, ct.to_date
FROM coach_tenures ct
JOIN coaches c ON c.id = ct.coach_id
JOIN teams t ON t.id = ct.team_id
JOIN leagues l ON l.id = t.league_id
WHERE l.slug = $1 AND ct.role = 'head'
ORDER BY ct.team_id, ct.from_date;

-- name: ListLeagueSkaterLines :many
-- Every skater line of a league (absences backtest, plan 04 L4).
SELECT pm.game_id, pm.side, pm.player_id, pm.position, pm.toi_seconds
FROM player_match_stats pm
JOIN games g ON g.id = pm.game_id
JOIN leagues l ON l.id = g.league_id
WHERE l.slug = $1
ORDER BY pm.game_id, pm.side, pm.player_id;
