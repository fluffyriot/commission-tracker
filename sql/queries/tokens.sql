-- name: CreateToken :one
INSERT INTO tokens (id, encrypted_access_token, nonce, created_at, updated_at, source_id, profile_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
RETURNING *;

-- name: GetTokenBySource :one
SELECT * FROM tokens
where source_id = $1;

-- name: DeleteTokenById :exec
DELETE FROM tokens
WHERE id = $1;