-- name: CoverageByLeagueSeason :many
WITH fin AS (
    SELECT g.id, g.league_id, g.season
    FROM games g
    WHERE g.status = 'finished'
),
odds_games AS (
    SELECT DISTINCT o.game_id FROM odds_snapshots o WHERE o.game_id IS NOT NULL
),
goalie_games AS (
    SELECT gl.game_id
    FROM goalies gl
    WHERE gl.starter
    GROUP BY gl.game_id
    HAVING count(DISTINCT gl.side) = 2
),
skater_games AS (
    SELECT pm.game_id
    FROM player_match_stats pm
    GROUP BY pm.game_id
    HAVING count(DISTINCT pm.side) = 2
)
SELECT
    l.slug,
    l.name,
    l.role,
    l.sport,
    fin.season,
    count(*)::bigint AS games,
    count(og.game_id)::bigint AS games_with_odds,
    count(gg.game_id)::bigint AS games_with_goalies,
    count(sg.game_id)::bigint AS games_with_skaters,
    count(*) FILTER (WHERE g.phase <> 'regular')::bigint AS games_non_regular,
    count(*) FILTER (WHERE g.decided_in IN ('ot', 'so'))::bigint AS games_past_60
FROM fin
JOIN leagues l ON l.id = fin.league_id
JOIN games g ON g.id = fin.id
LEFT JOIN odds_games og ON og.game_id = fin.id
LEFT JOIN goalie_games gg ON gg.game_id = fin.id
LEFT JOIN skater_games sg ON sg.game_id = fin.id
GROUP BY l.slug, l.name, l.role, l.sport, fin.season
ORDER BY l.slug, fin.season;

-- name: MarginByLeague :many
-- Overround of complete books: 1X2 needs home/draw/away, ML needs home/away, in one capture.
WITH books AS (
    SELECT g.league_id, o.game_id, o.source, o.book, o.market, o.captured_at,
           sum(1.0 / o.price) - 1 AS margin,
           count(DISTINCT o.selection) AS n_sel
    FROM odds_snapshots o
    JOIN games g ON g.id = o.game_id
    WHERE o.market IN ('1x2', 'ml') AND o.selection IN ('home', 'draw', 'away')
    GROUP BY g.league_id, o.game_id, o.source, o.book, o.market, o.captured_at
)
SELECT
    l.slug,
    b.market,
    avg(b.margin)::float8 AS avg_margin,
    count(DISTINCT b.book)::bigint AS books,
    count(DISTINCT b.game_id)::bigint AS games
FROM books b
JOIN leagues l ON l.id = b.league_id
WHERE (b.market = '1x2' AND b.n_sel = 3) OR (b.market = 'ml' AND b.n_sel = 2)
GROUP BY l.slug, b.market
ORDER BY l.slug, b.market;

-- name: MarketsByLeague :many
SELECT l.slug, o.market, count(DISTINCT o.game_id)::bigint AS games, count(DISTINCT o.book)::bigint AS books
FROM odds_snapshots o
JOIN games g ON g.id = o.game_id
JOIN leagues l ON l.id = g.league_id
GROUP BY l.slug, o.market
ORDER BY l.slug, o.market;

-- name: JoinRateBySource :many
-- A join counts only against another source: a swehockey row is matched when its
-- game also has an API-HOCKEY id, and is counted only once the league has API games.
SELECT l.slug, f.source,
       count(*)::bigint AS fixtures,
       count(*) FILTER (
           WHERE f.game_id IS NOT NULL AND (f.source <> 'swehockey' OR g.apihockey_id IS NOT NULL)
       )::bigint AS matched
FROM fixture_links f
JOIN leagues l ON l.id = f.league_id
LEFT JOIN games g ON g.id = f.game_id
WHERE f.source <> 'swehockey'
   OR EXISTS (SELECT 1 FROM games g2 WHERE g2.league_id = f.league_id AND g2.apihockey_id IS NOT NULL)
GROUP BY l.slug, f.source
ORDER BY l.slug, f.source;

-- name: UnmatchedFixtures :many
SELECT l.slug, f.source, f.source_fixture_id, coalesce(f.detail, '')::text AS detail
FROM fixture_links f
JOIN leagues l ON l.id = f.league_id
WHERE f.game_id IS NULL
ORDER BY l.slug, f.source, f.source_fixture_id
LIMIT 200;

-- name: TeamsCreatedBySecondary :many
-- Teams that only a secondary source knows about: usually a missing alias.
SELECT l.slug, t.name, t.created_by
FROM teams t
JOIN leagues l ON l.id = t.league_id
WHERE t.apihockey_id IS NULL
  AND EXISTS (SELECT 1 FROM teams t2 WHERE t2.league_id = t.league_id AND t2.apihockey_id IS NOT NULL)
ORDER BY l.slug, t.name;

-- name: SoccerSeasons :many
-- Plan 05 S1 gate Z1: halves, football-data.co.uk join and Pinnacle closing per soccer season.
WITH pin AS (
    SELECT o.game_id
    FROM odds_snapshots o
    WHERE o.book = 'pinnacle' AND o.market = '1x2' AND o.snapshot_kind = 'closing' AND o.game_id IS NOT NULL
    GROUP BY o.game_id
    HAVING count(DISTINCT o.selection) = 3
),
fd AS (
    SELECT DISTINCT fl.game_id FROM fixture_links fl WHERE fl.source = 'fdcouk' AND fl.game_id IS NOT NULL
)
SELECT
    l.slug,
    g.season,
    count(*)::bigint AS games,
    count(*) FILTER (WHERE jsonb_array_length(g.period_scores) = 2)::bigint AS with_halves,
    count(*) FILTER (WHERE g.phase <> 'regular')::bigint AS knockout,
    count(*) FILTER (WHERE g.decided_in = 'ot')::bigint AS extra_time,
    count(*) FILTER (WHERE g.decided_in = 'so')::bigint AS penalties,
    count(fd.game_id)::bigint AS with_fdcouk,
    count(pin.game_id)::bigint AS with_pinnacle
FROM games g
JOIN leagues l ON l.id = g.league_id
LEFT JOIN fd ON fd.game_id = g.id
LEFT JOIN pin ON pin.game_id = g.id
WHERE l.sport = 'soccer' AND g.status = 'finished'
GROUP BY l.slug, g.season
ORDER BY l.slug, g.season;

-- name: SoccerPinnacleClosing :many
-- Closing Pinnacle 1X2 and the 90-minute result of finished soccer games (market log-loss, gate Z1).
SELECT
    l.slug,
    g.season,
    g.home_60::int AS home_60,
    g.away_60::int AS away_60,
    (max(o.price) FILTER (WHERE o.selection = 'home'))::float8 AS price_home,
    (max(o.price) FILTER (WHERE o.selection = 'draw'))::float8 AS price_draw,
    (max(o.price) FILTER (WHERE o.selection = 'away'))::float8 AS price_away
FROM games g
JOIN leagues l ON l.id = g.league_id
JOIN odds_snapshots o ON o.game_id = g.id
WHERE l.sport = 'soccer' AND g.status = 'finished' AND g.home_60 IS NOT NULL AND g.away_60 IS NOT NULL
  AND o.book = 'pinnacle' AND o.market = '1x2' AND o.snapshot_kind = 'closing' AND o.captured_at < g.start_at
GROUP BY l.slug, g.season, g.id, g.home_60, g.away_60
HAVING count(DISTINCT o.selection) = 3
ORDER BY l.slug, g.season, g.id;
