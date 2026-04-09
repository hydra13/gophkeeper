-- +goose Up
-- +goose StatementBegin
ALTER TABLE key_versions
    ADD COLUMN encrypted_key BYTEA,
    ADD COLUMN key_nonce BYTEA;

UPDATE key_versions
SET encrypted_key = '\\x',
    key_nonce = '\\x'
WHERE encrypted_key IS NULL OR key_nonce IS NULL;

ALTER TABLE key_versions
    ALTER COLUMN encrypted_key SET NOT NULL,
    ALTER COLUMN key_nonce SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE key_versions
    DROP COLUMN IF EXISTS encrypted_key,
    DROP COLUMN IF EXISTS key_nonce;
-- +goose StatementEnd
