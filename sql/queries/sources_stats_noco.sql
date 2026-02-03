-- name: GetAllSourcesStatsForUser :many
SELECT ss.id, ss.date, ss.source_id, ss.followers_count, ss.following_count, ss.posts_count, ss.average_likes, ss.average_reposts, ss.average_views
FROM sources_stats ss
    LEFT JOIN sources s ON ss.source_id = s.id
WHERE
    s.user_id = $1
ORDER BY ss.date DESC;
-- name: GetUnsyncedSourcesStatsForTarget :many
SELECT ss.*
FROM sources_stats ss
WHERE
    ss.source_id = $1
    AND NOT EXISTS (
        SELECT 1
        FROM sources_stats_on_target ssot
        WHERE
            ssot.stat_id = ss.id
            AND ssot.target_id = $2
    );

-- name: GetAllSourcesStatsWithTargetInfo :many
SELECT ss.*, map.target_record_id
FROM
    sources_stats ss
    LEFT JOIN sources_stats_on_target map ON ss.id = map.stat_id
    AND map.target_id = $1
WHERE
    ss.source_id = $2;

-- name: GetSyncedSourcesStatsForUpdate :many
SELECT ss.*, map.target_record_id
FROM
    sources_stats ss
    JOIN sources_stats_on_target map ON ss.id = map.stat_id
    AND map.target_id = $1
WHERE
    ss.source_id = $2
    AND ss.date >= $3;

-- name: AddSourcesStatToTarget :one
INSERT INTO
    sources_stats_on_target (
        id,
        synced_at,
        stat_id,
        target_id,
        target_record_id
    )
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (stat_id, target_id) DO
UPDATE
SET
    synced_at = $2,
    target_record_id = $5
RETURNING
    *;