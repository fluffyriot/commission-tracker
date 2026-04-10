-- name: BackupGetUser :one
SELECT id, username, created_at, updated_at, sync_period, profile_image,
       last_seen_version, intro_completed
FROM users WHERE id = $1;

-- name: BackupGetSourcesForUser :many
SELECT * FROM sources WHERE user_id = $1 ORDER BY created_at;

-- name: BackupGetTargetsForUser :many
SELECT * FROM targets WHERE user_id = $1 ORDER BY created_at;

-- name: BackupGetTokensForUser :many
SELECT t.* FROM tokens t
LEFT JOIN sources s ON t.source_id = s.id
LEFT JOIN targets tgt ON t.target_id = tgt.id
WHERE s.user_id = $1 OR tgt.user_id = $1
ORDER BY t.created_at;

-- name: BackupGetPostsForUser :many
SELECT p.* FROM posts p
JOIN sources s ON p.source_id = s.id
WHERE s.user_id = $1
ORDER BY p.created_at;

-- name: BackupGetReactionsForUser :many
SELECT prh.* FROM posts_reactions_history prh
JOIN posts p ON prh.post_id = p.id
JOIN sources s ON p.source_id = s.id
WHERE s.user_id = $1
ORDER BY prh.synced_at;

-- name: BackupGetTagClassificationsForUser :many
SELECT * FROM tag_classifications WHERE user_id = $1 ORDER BY created_at;

-- name: BackupGetTagsForUser :many
SELECT * FROM tags WHERE user_id = $1 ORDER BY created_at;

-- name: BackupGetPostTagsForUser :many
SELECT pt.* FROM post_tags pt
JOIN posts p ON pt.post_id = p.id
JOIN sources s ON p.source_id = s.id
WHERE s.user_id = $1;

-- name: BackupGetExclusionsForUser :many
SELECT e.* FROM exclusions e
JOIN sources s ON e.source_id = s.id
WHERE s.user_id = $1;

-- name: BackupGetRedirectsForUser :many
SELECT r.* FROM redirects r
JOIN sources s ON r.source_id = s.id
WHERE s.user_id = $1;

-- name: BackupGetSourcesStatsForUser :many
SELECT ss.* FROM sources_stats ss
JOIN sources s ON ss.source_id = s.id
WHERE s.user_id = $1
ORDER BY ss.date;

-- name: BackupGetAnalyticsPageStatsForUser :many
SELECT aps.* FROM analytics_page_stats aps
JOIN sources s ON aps.source_id = s.id
WHERE s.user_id = $1
ORDER BY aps.date;

-- name: BackupGetAnalyticsSiteStatsForUser :many
SELECT ass.* FROM analytics_site_stats ass
JOIN sources s ON ass.source_id = s.id
WHERE s.user_id = $1
ORDER BY ass.date;

-- name: BackupDeletePostTagsForUser :exec
DELETE FROM post_tags WHERE id IN (
    SELECT pt.id FROM post_tags pt
    JOIN posts p ON pt.post_id = p.id
    JOIN sources s ON p.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteReactionsForUser :exec
DELETE FROM posts_reactions_history WHERE id IN (
    SELECT prh.id FROM posts_reactions_history prh
    JOIN posts p ON prh.post_id = p.id
    JOIN sources s ON p.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteExclusionsForUser :exec
DELETE FROM exclusions WHERE id IN (
    SELECT e.id FROM exclusions e
    JOIN sources s ON e.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteRedirectsForUser :exec
DELETE FROM redirects WHERE id IN (
    SELECT r.id FROM redirects r
    JOIN sources s ON r.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeletePostsForUser :exec
DELETE FROM posts WHERE id IN (
    SELECT p.id FROM posts p
    JOIN sources s ON p.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteSourcesStatsForUser :exec
DELETE FROM sources_stats WHERE id IN (
    SELECT ss.id FROM sources_stats ss
    JOIN sources s ON ss.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteAnalyticsPageStatsForUser :exec
DELETE FROM analytics_page_stats WHERE id IN (
    SELECT aps.id FROM analytics_page_stats aps
    JOIN sources s ON aps.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteAnalyticsSiteStatsForUser :exec
DELETE FROM analytics_site_stats WHERE id IN (
    SELECT ass.id FROM analytics_site_stats ass
    JOIN sources s ON ass.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteTokensForUser :exec
DELETE FROM tokens WHERE id IN (
    SELECT t.id FROM tokens t
    LEFT JOIN sources s ON t.source_id = s.id
    LEFT JOIN targets tgt ON t.target_id = tgt.id
    WHERE s.user_id = $1 OR tgt.user_id = $1
);

-- name: BackupDeleteTagsForUser :exec
DELETE FROM tags WHERE user_id = $1;

-- name: BackupDeleteTagClassificationsForUser :exec
DELETE FROM tag_classifications WHERE user_id = $1;

-- name: BackupDeletePostsOnTargetForUser :exec
DELETE FROM posts_on_target WHERE post_id IN (
    SELECT p.id FROM posts p
    JOIN sources s ON p.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteSourcesOnTargetForUser :exec
DELETE FROM sources_on_target WHERE source_id IN (
    SELECT s.id FROM sources s
    WHERE s.user_id = $1
);

-- name: BackupDeleteSourcesStatsOnTargetForUser :exec
DELETE FROM sources_stats_on_target WHERE stat_id IN (
    SELECT ss.id FROM sources_stats ss
    JOIN sources s ON ss.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteAnalyticsPageStatsOnTargetForUser :exec
DELETE FROM analytics_page_stats_on_target WHERE stat_id IN (
    SELECT aps.id FROM analytics_page_stats aps
    JOIN sources s ON aps.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteAnalyticsSiteStatsOnTargetForUser :exec
DELETE FROM analytics_site_stats_on_target WHERE stat_id IN (
    SELECT ass.id FROM analytics_site_stats ass
    JOIN sources s ON ass.source_id = s.id
    WHERE s.user_id = $1
);

-- name: BackupDeleteTableMappingsForUser :exec
DELETE FROM table_mappings WHERE target_id IN (
    SELECT t.id FROM targets t
    WHERE t.user_id = $1
);

-- name: BackupDeleteTargetsForUser :exec
DELETE FROM targets WHERE user_id = $1;

-- name: BackupDeleteSourcesForUser :exec
DELETE FROM sources WHERE user_id = $1;

-- name: BackupDeleteExportsForUser :exec
DELETE FROM exports WHERE user_id = $1;

-- name: BackupDeleteLogsForUser :exec
DELETE FROM logs WHERE source_id IN (
    SELECT s.id FROM sources s WHERE s.user_id = $1
) OR target_id IN (
    SELECT t.id FROM targets t WHERE t.user_id = $1
);
