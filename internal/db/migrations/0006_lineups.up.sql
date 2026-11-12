-- Plan 04 L2: pre-game lineups as published by the source. confirmed_at is
-- the first poll that listed the player; a poll replaces the team's set.
CREATE TABLE lineups (
    game_id      bigint NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    team_id      integer NOT NULL REFERENCES teams (id),
    player_id    integer NOT NULL REFERENCES players (id),
    position     text,
    confirmed_at timestamptz NOT NULL,
    source       text NOT NULL,
    PRIMARY KEY (game_id, team_id, player_id)
);
