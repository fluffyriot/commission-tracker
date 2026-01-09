-- +goose Up
CREATE TABLE tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    encrypted_access_token BYTEA NOT NULL,
    nonce BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    network TEXT NOT NULL,
    CONSTRAINT network_check
        CHECK (network IN ('Instagram'))
);

-- +goose Down
DROP TABLE tokens;