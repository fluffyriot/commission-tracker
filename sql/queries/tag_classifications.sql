-- name: CreateTagClassification :one
INSERT INTO
    tag_classifications (
        id,
        created_at,
        updated_at,
        user_id,
        name
    )
VALUES ($1, $2, $3, $4, $5)
RETURNING
    *;

-- name: GetTagClassificationsForUser :many
SELECT *
FROM tag_classifications
WHERE
    user_id = $1
ORDER BY name ASC;

-- name: GetTagClassificationById :one
SELECT *
FROM tag_classifications
WHERE
    id = $1;

-- name: UpdateTagClassification :one
UPDATE tag_classifications
SET
    name = $2,
    updated_at = $3
WHERE
    id = $1
RETURNING
    *;

-- name: DeleteTagClassification :exec
DELETE FROM tag_classifications WHERE id = $1;
