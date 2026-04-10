-- +goose Up

CREATE TABLE api_tokens (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL,
    token_hash TEXT NOT NULL,
    last_used_at TIMESTAMP,
    CONSTRAINT fk_api_tokens_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX idx_api_tokens_user_id ON api_tokens (user_id);
CREATE UNIQUE INDEX idx_api_tokens_token_hash ON api_tokens (token_hash);

-- +goose Down

DROP TABLE api_tokens;
