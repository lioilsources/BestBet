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
)
SELECT
    l.slug,
    l.name,
    l.role,
    fin.season,
    count(*)::bigint AS games,
    count(og.game_id)::bigint AS games_with_odds,
    count(gg.game_id)::bigint AS games_with_goalies,
    count(*) FILTER (WHERE g.phase <> 'regular')::bigint AS games_non_regular,
    count(*) FILTER (WHERE g.decided_in IN ('ot', 'so'))::bigint AS games_past_60
FROM fin
JOIN leagues l ON l.id = fin.league_id
JOIN games g ON g.id = fin.id
LEFT JOIN odds_games og ON og.game_id = fin.id
LEFT JOIN goalie_games gg ON gg.game_id = fin.id
GROUP BY l.slug, l.name, l.role, fin.season
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
SELECT l.slug, f.source,
       count(*)::bigint AS fixtures,
       count(f.game_id)::bigint AS matched
FROM fixture_links f
JOIN leagues l ON l.id = f.league_id
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
