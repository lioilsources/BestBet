-- Paper ledger of market hints (plan 03 §3): the first time a game meets a
-- rule, at that price. Never rewritten; evaluated against the sharp close.
CREATE TABLE hints (
    id           bigserial PRIMARY KEY,
    game_id      bigint NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    rule_version text NOT NULL,
    book         text NOT NULL,
    market       text NOT NULL,
    selection    text NOT NULL,
    price        numeric(9, 3) NOT NULL CHECK (price > 1),
    p_sharp      numeric(6, 5) NOT NULL,
    p_soft       numeric(6, 5) NOT NULL,
    captured_at  timestamptz NOT NULL, -- when the book showed the price
    recorded_at  timestamptz NOT NULL, -- when the rule saw it
    UNIQUE (game_id, rule_version)
);
