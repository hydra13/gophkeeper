ALTER TABLE sync_conflicts
    ADD COLUMN local_record JSONB,
    ADD COLUMN server_record JSONB;
