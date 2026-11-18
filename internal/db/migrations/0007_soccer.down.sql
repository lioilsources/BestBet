UPDATE games SET ot_type = 'unknown' WHERE ot_type = 'et_2x15';
ALTER TABLE games DROP CONSTRAINT games_ot_type_check;
ALTER TABLE games ADD CONSTRAINT games_ot_type_check
    CHECK (ot_type IN ('none', '3v3_5', '5v5_20', 'shootout', 'unknown'));
ALTER TABLE games DROP COLUMN asa_id;
