-- +goose Up
-- +goose StatementBegin
CREATE TABLE key_versions (
    id BIGSERIAL PRIMARY KEY,
    version BIGINT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('active', 'deprecated', 'retired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deprecated_at TIMESTAMPTZ,
    retired_at TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS key_versions;
-- +goose StatementEnd
