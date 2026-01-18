-- name: AddSourceToTarget :one
INSERT INTO sources_on_target (id, source_id, target_id, target_source_id)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetTargetSources :many
SELECT * FROM sources_on_target
where target_id = $1;

-- name: GetTargetSourceBySource :one
SELECT * FROM sources_on_target
where target_id = $1 and source_id = $2
LIMIT 1;

-- name: RemoveTargetSourceById :exec
DELETE FROM sources_on_target
where id = $1;

-- name: GetSourcesOfTarget :many
SELECT * FROM sources_on_target
where source_id = $1;

-- name: DeleteSourceTarget :exec
DELETE FROM sources_on_target
where target_id = $1 and source_id = $2;
