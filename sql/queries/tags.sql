-- name: CreateTag :one
INSERT INTO
    tags (
        id,
        created_at,
        updated_at,
        user_id,
        classification_id,
        name
    )
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING
    *;

-- name: GetTagsForUser :many
SELECT t.id,
    t.created_at,
    t.updated_at,
    t.user_id,
    t.classification_id,
    t.name,
    tc.name as classification_name
FROM tags t
    LEFT JOIN tag_classifications tc ON t.classification_id = tc.id
WHERE
    t.user_id = $1
ORDER BY tc.name ASC NULLS LAST, t.name ASC;

-- name: GetTagsForClassification :many
SELECT *
FROM tags
WHERE
    classification_id = $1
ORDER BY name ASC;

-- name: GetUnclassifiedTagsForUser :many
SELECT *
FROM tags
WHERE
    user_id = $1
    AND classification_id IS NULL
ORDER BY name ASC;

-- name: GetTagById :one
SELECT *
FROM tags
WHERE
    id = $1;

-- name: UpdateTag :one
UPDATE tags
SET
    name = $2,
    updated_at = $3,
    classification_id = $4
WHERE
    id = $1
RETURNING
    *;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = $1;
