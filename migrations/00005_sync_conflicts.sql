-- +goose Up
-- +goose StatementBegin
CREATE TABLE sync_conflicts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    record_id BIGINT NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    local_revision BIGINT NOT NULL,
    server_revision BIGINT NOT NULL,
    resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolution TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sync_conflicts;
-- +goose StatementEnd
