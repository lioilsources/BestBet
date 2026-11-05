-- M0 hockey schema: leagues, teams, games (with periods, OT/SO, play-off fields),
-- goalies per game, odds snapshots, and fixture links used to measure join quality.

CREATE TABLE leagues (
    id                     serial PRIMARY KEY,
    slug                   text NOT NULL UNIQUE,
    name                   text NOT NULL,
    country                text NOT NULL,
    sport                  text NOT NULL DEFAULT 'hockey',
    role                   text NOT NULL DEFAULT 'fringe' CHECK (role IN ('fringe', 'semi_efficient', 'control')),
    apihockey_id           integer UNIQUE,
    oddspapi_tournament_id integer UNIQUE,
    created_at             timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE teams (
    id           serial PRIMARY KEY,
    league_id    integer NOT NULL REFERENCES leagues (id),
    name         text NOT NULL,
    name_norm    text NOT NULL,
    apihockey_id integer,
    created_by   text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (league_id, name_norm)
);
CREATE UNIQUE INDEX teams_apihockey_id ON teams (league_id, apihockey_id) WHERE apihockey_id IS NOT NULL;

-- Source-specific spellings of a team (swehockey "Östersunds IK" vs API-HOCKEY "Ostersund").
CREATE TABLE team_aliases (
    source     text NOT NULL,
    league_id  integer NOT NULL REFERENCES leagues (id),
    alias_norm text NOT NULL,
    team_id    integer NOT NULL REFERENCES teams (id),
    PRIMARY KEY (source, league_id, alias_norm)
);

CREATE TABLE games (
    id                  bigserial PRIMARY KEY,
    league_id           integer NOT NULL REFERENCES leagues (id),
    season              integer NOT NULL, -- start year: 2025 = 2025/26
    start_at            timestamptz NOT NULL,
    status              text NOT NULL, -- scheduled | finished | postponed | cancelled | live | other
    home_team_id        integer NOT NULL REFERENCES teams (id),
    away_team_id        integer NOT NULL REFERENCES teams (id),
    -- Final score including OT and the shootout winner goal, as books settle ML.
    home_score          integer,
    away_score          integer,
    -- Regulation (60 min) score, as EU books settle 1X2.
    home_60             integer,
    away_60             integer,
    -- [[home, away], ...] for periods 1..3 only.
    period_scores       jsonb,
    ot_home             integer,
    ot_away             integer,
    so_winner           text CHECK (so_winner IN ('home', 'away')),
    decided_in          text CHECK (decided_in IN ('reg', 'ot', 'so')),
    ot_type             text CHECK (ot_type IN ('none', '3v3_5', '5v5_20', 'shootout', 'unknown')),
    phase               text NOT NULL DEFAULT 'regular'
                        CHECK (phase IN ('regular', 'pre_playoff', 'playoff', 'relegation_playoff', 'promotion_qual')),
    stage_raw           text,
    series_id           text,
    game_no             integer,
    series_score_before text, -- "home_wins-away_wins" from the home team's perspective
    apihockey_id        integer UNIQUE,
    swehockey_id        integer UNIQUE,
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX games_league_start ON games (league_id, start_at);
CREATE INDEX games_match_key ON games (league_id, home_team_id, away_team_id, start_at);

CREATE TABLE goalies (
    id             bigserial PRIMARY KEY,
    game_id        bigint NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    team_id        integer NOT NULL REFERENCES teams (id),
    side           text NOT NULL CHECK (side IN ('home', 'away')),
    player_name    text NOT NULL,
    jersey         integer,
    starter        boolean NOT NULL,
    shots_against  integer,
    saves          integer,
    goals_against  integer,
    save_pct       numeric(6, 4),
    toi_seconds    integer,
    confirm_status text NOT NULL DEFAULT 'CONFIRMED' CHECK (confirm_status IN ('UNCONFIRMED', 'EXPECTED', 'CONFIRMED')),
    confirmed_at   timestamptz,
    source         text NOT NULL,
    UNIQUE (game_id, side, player_name, source)
);

-- Every price we ever saw. Rows are never updated: a new capture is a new row.
CREATE TABLE odds_snapshots (
    id                bigserial PRIMARY KEY,
    game_id           bigint REFERENCES games (id) ON DELETE SET NULL,
    source            text NOT NULL, -- apihockey | oddspapi | tipsport_xml | betexplorer
    source_fixture_id text NOT NULL,
    book              text NOT NULL,
    market            text NOT NULL, -- ml | 1x2 | puck_line | total | p1_1x2 | p1_total | correct_score | other
    selection         text NOT NULL, -- home | draw | away | over | under | "3-2" ...
    line              numeric(5, 2),
    price             numeric(9, 3) NOT NULL CHECK (price > 1),
    raw_market        text NOT NULL,
    snapshot_kind     text NOT NULL CHECK (snapshot_kind IN ('t24h', 't1h', 'closing', 'historical', 'adhoc')),
    captured_at       timestamptz NOT NULL,
    UNIQUE NULLS NOT DISTINCT (source, source_fixture_id, book, raw_market, selection, line, captured_at)
);
CREATE INDEX odds_snapshots_game ON odds_snapshots (game_id, market);

-- One row per external fixture we tried to attach to a game; NULL game_id = join failed.
CREATE TABLE fixture_links (
    source            text NOT NULL,
    source_fixture_id text NOT NULL,
    league_id         integer NOT NULL REFERENCES leagues (id),
    game_id           bigint REFERENCES games (id) ON DELETE SET NULL,
    match_method      text NOT NULL, -- external_id | date_teams | none
    detail            text,
    seen_at           timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (source, source_fixture_id)
);
