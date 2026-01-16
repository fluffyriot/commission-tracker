-- +goose Up
CREATE TABLE tokens (
    id UUID PRIMARY KEY,
    encrypted_access_token BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    profile_id TEXT,
    source_id UUID,
    CONSTRAINT fk_source FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE,
    target_id UUID,
    CONSTRAINT fk_target FOREIGN KEY (target_id) REFERENCES targets(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE tokens;