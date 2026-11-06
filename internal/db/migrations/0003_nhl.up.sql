-- NHL official API (api-web.nhle.com) as a source of games: its own id like swehockey.
ALTER TABLE games ADD COLUMN nhl_id bigint UNIQUE;
