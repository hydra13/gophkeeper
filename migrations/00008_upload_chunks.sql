-- +goose Up
-- +goose StatementBegin
CREATE TABLE upload_chunks (
    upload_id BIGINT NOT NULL REFERENCES upload_sessions(id) ON DELETE CASCADE,
    chunk_index BIGINT NOT NULL,
    size BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (upload_id, chunk_index)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS upload_chunks;
-- +goose StatementEnd
