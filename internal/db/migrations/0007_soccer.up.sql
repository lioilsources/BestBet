-- Plan 05 S1: soccer (MLS) next to hockey. American Soccer Analysis owns soccer games.
ALTER TABLE games ADD COLUMN asa_id text UNIQUE;

-- Soccer knockouts: 2 x 15 min extra time; a draw after 90 min that goes straight
-- to penalties keeps ot_type 'none' and decided_in 'so'.
ALTER TABLE games DROP CONSTRAINT games_ot_type_check;
ALTER TABLE games ADD CONSTRAINT games_ot_type_check
    CHECK (ot_type IN ('none', '3v3_5', '5v5_20', 'shootout', 'unknown', 'et_2x15'));
