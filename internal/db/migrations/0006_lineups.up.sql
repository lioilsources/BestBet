-- Plan 04 L2: pre-game lineups as published by the source. status is
-- 'available' (the list of players who can play, about 90 min before) or
-- 'confirmed' (the dressed lineup, minutes before). confirmed_at is the
-- first poll that listed the player with that status; a poll replaces the
-- team's set of that status.
CREATE TABLE lineups (
    game_id      bigint NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    team_id      integer NOT NULL REFERENCES teams (id),
    status       text NOT NULL CHECK (status IN ('available', 'confirmed')),
    player_id    integer NOT NULL REFERENCES players (id),
    position     text,
    confirmed_at timestamptz NOT NULL,
    source       text NOT NULL,
    PRIMARY KEY (game_id, team_id, status, player_id)
);
