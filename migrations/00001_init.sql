-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE key_versions (
    id BIGSERIAL PRIMARY KEY,
    version BIGINT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active', 'deprecated', 'retired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deprecated_at TIMESTAMPTZ,
    retired_at TIMESTAMPTZ
);

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

CREATE TABLE record_revisions (
    id BIGSERIAL PRIMARY KEY,
    record_id BIGINT NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    revision BIGINT NOT NULL,
    device_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, revision)
);

CREATE INDEX idx_record_revisions_user_id ON record_revisions(user_id);

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

CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id TEXT NOT NULL,
    device_name TEXT NOT NULL,
    client_type TEXT NOT NULL,
    refresh_token TEXT NOT NULL UNIQUE,
    last_seen_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);

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

CREATE TABLE upload_chunks (
    upload_id BIGINT NOT NULL REFERENCES upload_sessions(id) ON DELETE CASCADE,
    chunk_index BIGINT NOT NULL,
    size BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (upload_id, chunk_index)
);

CREATE TABLE payloads (
    id BIGSERIAL PRIMARY KEY,
    record_id BIGINT NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    version BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    size BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (record_id, version)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS payloads;
DROP TABLE IF EXISTS upload_chunks;
DROP TABLE IF EXISTS upload_sessions;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS sync_conflicts;
DROP TABLE IF EXISTS record_revisions;
DROP TABLE IF EXISTS records;
DROP TABLE IF EXISTS key_versions;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
