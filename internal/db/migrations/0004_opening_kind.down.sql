DELETE FROM odds_snapshots WHERE snapshot_kind = 'opening';
ALTER TABLE odds_snapshots DROP CONSTRAINT odds_snapshots_snapshot_kind_check;
ALTER TABLE odds_snapshots ADD CONSTRAINT odds_snapshots_snapshot_kind_check
    CHECK (snapshot_kind IN ('t24h', 't1h', 'closing', 'historical', 'adhoc'));
