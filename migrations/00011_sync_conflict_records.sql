-- +goose Up
-- +goose StatementBegin
ALTER TABLE sync_conflicts
    ADD COLUMN local_record JSONB,
    ADD COLUMN server_record JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sync_conflicts
    DROP COLUMN IF EXISTS local_record,
    DROP COLUMN IF EXISTS server_record;
-- +goose StatementEnd
