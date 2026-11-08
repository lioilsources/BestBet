-- Thinned odds histories keep the opening price per selection (odds.Thin).
ALTER TABLE odds_snapshots DROP CONSTRAINT odds_snapshots_snapshot_kind_check;
ALTER TABLE odds_snapshots ADD CONSTRAINT odds_snapshots_snapshot_kind_check
    CHECK (snapshot_kind IN ('opening', 't24h', 't1h', 'closing', 'historical', 'adhoc'));
