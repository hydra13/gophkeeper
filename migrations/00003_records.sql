-- +goose Up
-- +goose StatementBegin
CREATE TABLE records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('login', 'text', 'binary', 'card')),
    name TEXT NOT NULL,
    metadata TEXT NOT NULL DEFAULT '',
    payload JSONB,
    revision BIGINT NOT NULL,
    deleted_at TIMESTAMPTZ,
    device_id TEXT NOT NULL,
    key_version BIGINT NOT NULL REFERENCES key_versions(version),
    payload_version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (payload_version > 0 OR type <> 'binary')
);

CREATE INDEX idx_records_user_id ON records(user_id);
CREATE INDEX idx_records_revision ON records(user_id, revision);
CREATE INDEX idx_records_deleted_at ON records(deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS records;
-- +goose StatementEnd
