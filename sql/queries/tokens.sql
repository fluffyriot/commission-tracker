-- name: CreateToken :one
INSERT INTO tokens (id, user_id, encrypted_access_token, nonce, created_at, updated_at, network)
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

-- name: GetTokenByNetworkAndUser :one
SELECT * FROM tokens
where user_id = $1 and network = $2;