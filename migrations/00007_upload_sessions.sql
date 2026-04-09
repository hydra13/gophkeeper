-- +goose Up
-- +goose StatementBegin
CREATE TABLE upload_sessions (
    id BIGSERIAL PRIMARY KEY,
    record_id BIGINT NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'completed', 'aborted')),
    total_chunks BIGINT NOT NULL,
    received_chunks BIGINT NOT NULL DEFAULT 0,
    chunk_size BIGINT NOT NULL,
    total_size BIGINT NOT NULL,
    key_version BIGINT NOT NULL REFERENCES key_versions(version),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_upload_sessions_user_id ON upload_sessions(user_id);
CREATE INDEX idx_upload_sessions_record_id ON upload_sessions(record_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS upload_sessions;
-- +goose StatementEnd
