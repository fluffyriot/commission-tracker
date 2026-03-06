-- name: CreateApiToken :one
INSERT INTO api_tokens (id, created_at, user_id, token_hash)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetApiTokenByHash :one
SELECT * FROM api_tokens WHERE token_hash = $1;

-- name: GetApiTokensByUser :many
SELECT * FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC;

-- name: DeleteApiToken :exec
DELETE FROM api_tokens WHERE id = $1 AND user_id = $2;

-- name: DeleteApiTokensByUser :exec
DELETE FROM api_tokens WHERE user_id = $1;

-- name: UpdateApiTokenLastUsed :exec
UPDATE api_tokens SET last_used_at = $2 WHERE id = $1;
