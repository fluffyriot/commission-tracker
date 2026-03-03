-- name: AddTagToPost :one
INSERT INTO
    post_tags (id, created_at, post_id, tag_id)
VALUES ($1, $2, $3, $4)
RETURNING
    *;

-- name: RemoveTagFromPost :exec
DELETE FROM post_tags WHERE post_id = $1 AND tag_id = $2;

-- name: GetTagsForPost :many
SELECT t.id,
    t.name,
    t.classification_id,
    tc.name as classification_name
FROM post_tags pt
    JOIN tags t ON pt.tag_id = t.id
    LEFT JOIN tag_classifications tc ON t.classification_id = tc.id
WHERE
    pt.post_id = $1
ORDER BY tc.name ASC NULLS LAST, t.name ASC;

-- name: ClearTagsForPost :exec
DELETE FROM post_tags WHERE post_id = $1;

-- name: GetAllPostTagsForUser :many
SELECT pt.post_id,
    t.id as tag_id,
    t.name as tag_name,
    tc.name as classification_name
FROM post_tags pt
    JOIN tags t ON pt.tag_id = t.id
    LEFT JOIN tag_classifications tc ON t.classification_id = tc.id
    JOIN posts p ON pt.post_id = p.id
    JOIN sources s ON p.source_id = s.id
WHERE
    s.user_id = $1
ORDER BY tc.name ASC NULLS LAST, t.name ASC;

-- name: GetPostIdsWithNoTags :many
SELECT p.id as post_id
FROM posts p
    JOIN sources s ON p.source_id = s.id
    LEFT JOIN post_tags pt ON p.id = pt.post_id
WHERE
    s.user_id = $1
    AND pt.id IS NULL;
