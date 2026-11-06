-- Phase 0 (plan 01 §3, team detail 06 §3, milestone M0 completion):
-- players and their per-game lines, absences, coaches, units (lines, pairs),
-- duos for WOWY, model predictions and the user bet journal.

CREATE TABLE players (
    id          serial PRIMARY KEY,
    name        text NOT NULL,
    name_norm   text NOT NULL,
    birth_date  date,
    nationality text,
    position    text CHECK (position IN ('C', 'LW', 'RW', 'F', 'D', 'G')), -- F = forward, side unknown
    shoots      text CHECK (shoots IN ('L', 'R')),
    created_by  text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX players_name_norm ON players (name_norm);

-- External ids of a player per source (swehockey has none, so name+team+season is the key there).
CREATE TABLE player_sources (
    source           text NOT NULL,
    source_player_id text NOT NULL,
    player_id        integer NOT NULL REFERENCES players (id),
    PRIMARY KEY (source, source_player_id)
);

CREATE TABLE player_match_stats (
    player_id   integer NOT NULL REFERENCES players (id),
    game_id     bigint NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    team_id     integer NOT NULL REFERENCES teams (id),
    side        text NOT NULL CHECK (side IN ('home', 'away')),
    jersey      integer,
    position    text,
    line_no     integer, -- 1..4 forwards, 1..3 defence pairs, as listed in the game sheet
    toi_seconds integer,
    goals       integer,
    assists     integer,
    plus_minus  integer,
    shots       integer,
    pim         integer,
    source      text NOT NULL,
    PRIMARY KEY (player_id, game_id, source)
);
CREATE INDEX player_match_stats_game ON player_match_stats (game_id, side);

CREATE TABLE player_absence_games (
    player_id integer NOT NULL REFERENCES players (id),
    game_id   bigint NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    team_id   integer NOT NULL REFERENCES teams (id),
    reason    text NOT NULL DEFAULT 'unknown' CHECK (reason IN ('injury', 'suspension', 'healthy_scratch', 'other', 'unknown')),
    source    text NOT NULL,
    PRIMARY KEY (player_id, game_id)
);

CREATE TABLE coaches (
    id          serial PRIMARY KEY,
    name        text NOT NULL,
    birth_date  date,
    nationality text
);

CREATE TABLE coach_tenures (
    coach_id  integer NOT NULL REFERENCES coaches (id),
    team_id   integer NOT NULL REFERENCES teams (id),
    role      text NOT NULL CHECK (role IN ('head', 'assistant', 'goalie', 'gm')),
    from_date date NOT NULL,
    to_date   date,
    source    text NOT NULL,
    PRIMARY KEY (coach_id, team_id, role, from_date)
);

-- A unit is a fixed set of players: forward line, defence pair, PP/PK unit, goalie tandem.
CREATE TABLE units (
    id      serial PRIMARY KEY,
    team_id integer NOT NULL REFERENCES teams (id),
    kind    text NOT NULL CHECK (kind IN ('fwd_line', 'd_pair', 'five', 'pp1', 'pp2', 'pk1', 'pk2', 'goalie_tandem')),
    members integer[] NOT NULL, -- sorted player ids
    UNIQUE (team_id, kind, members)
);

CREATE TABLE unit_matches (
    unit_id      integer NOT NULL REFERENCES units (id),
    game_id      bigint NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    toi_together integer, -- NULL = only "listed together in the game sheet"
    gf           integer,
    ga           integer,
    xgf          numeric(6, 3),
    xga          numeric(6, 3),
    cf           integer,
    ca           integer,
    source       text NOT NULL,
    PRIMARY KEY (unit_id, game_id, source)
);

-- Duos across lines for WOWY: together | apart | a_absent | b_absent.
CREATE TABLE duos (
    id       serial PRIMARY KEY,
    team_id  integer NOT NULL REFERENCES teams (id),
    player_a integer NOT NULL REFERENCES players (id),
    player_b integer NOT NULL REFERENCES players (id),
    CHECK (player_a < player_b),
    UNIQUE (team_id, player_a, player_b)
);

CREATE TABLE duo_matches (
    duo_id  integer NOT NULL REFERENCES duos (id),
    game_id bigint NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    state   text NOT NULL CHECK (state IN ('together', 'apart', 'a_absent', 'b_absent')),
    toi     integer,
    gf      integer,
    ga      integer,
    xgf     numeric(6, 3),
    xga     numeric(6, 3),
    cf      integer,
    ca      integer,
    pts_a   integer,
    pts_b   integer,
    source  text NOT NULL,
    PRIMARY KEY (duo_id, game_id, state, source)
);

-- One model output per game, model version and input set. Backtest and production
-- write the same shape; kind tells them apart.
CREATE TABLE predictions (
    id            bigserial PRIMARY KEY,
    game_id       bigint NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    kind          text NOT NULL CHECK (kind IN ('backtest', 'live')),
    model_version text NOT NULL,
    inputs_hash   text NOT NULL,
    computed_at   timestamptz NOT NULL,
    lambda_home   numeric(6, 3) NOT NULL,
    lambda_away   numeric(6, 3) NOT NULL,
    score_matrix  jsonb NOT NULL, -- canon.ScoreMatrix (60 min)
    markets       jsonb NOT NULL, -- canon.MarketProbs
    contributions jsonb,          -- canon.Contribution[]
    UNIQUE (game_id, kind, model_version, inputs_hash)
);

-- User bet journal (plan 01 §1 bod 4). user_key is an anonymous device/user id.
CREATE TABLE journal (
    id            bigserial PRIMARY KEY,
    user_key      text NOT NULL,
    game_id       bigint REFERENCES games (id) ON DELETE SET NULL,
    book          text NOT NULL,
    market        text NOT NULL,
    selection     text NOT NULL,
    line          numeric(5, 2),
    price         numeric(9, 3) NOT NULL CHECK (price > 1),
    stake         numeric(12, 2) NOT NULL CHECK (stake > 0),
    placed_at     timestamptz NOT NULL,
    model_p       numeric(6, 5), -- model probability at placed_at
    closing_price numeric(9, 3),
    result        text CHECK (result IN ('win', 'loss', 'push', 'void', 'half_win', 'half_loss')),
    note          text
);
CREATE INDEX journal_user ON journal (user_key, placed_at);
