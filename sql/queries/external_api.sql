-- name: GetCurrentTotalFollowers :one
SELECT COALESCE(
    (
        SELECT SUM(COALESCE(followers_count, 0))
        FROM (
            SELECT DISTINCT ON (ss.source_id) ss.followers_count
            FROM sources_stats ss
                JOIN sources s ON ss.source_id = s.id
            WHERE s.user_id = $1
                AND s.is_active = TRUE
            ORDER BY ss.source_id, ss.date DESC
        ) as latest
    ), 0
)::BIGINT as total_followers;

-- name: GetTotalFollowersAtDate :one
SELECT COALESCE(
    (
        SELECT SUM(COALESCE(followers_count, 0))
        FROM (
            SELECT DISTINCT ON (ss.source_id) ss.followers_count
            FROM sources_stats ss
                JOIN sources s ON ss.source_id = s.id
            WHERE s.user_id = $1
                AND s.is_active = TRUE
                AND ss.date <= $2
            ORDER BY ss.source_id, ss.date DESC
        ) as latest
    ), 0
)::BIGINT as total_followers;

-- name: GetCurrentTotalStats :one
SELECT
    COALESCE(SUM(prh.likes), 0)::BIGINT AS total_likes,
    COALESCE(SUM(prh.reposts), 0)::BIGINT AS total_reposts,
    COALESCE(SUM(prh.views), 0)::BIGINT AS total_views
FROM (
    SELECT DISTINCT ON (prh.post_id) prh.likes, prh.reposts, prh.views
    FROM posts_reactions_history prh
        JOIN posts p ON prh.post_id = p.id
        JOIN sources s ON p.source_id = s.id
    WHERE s.user_id = $1
        AND s.is_active = TRUE
    ORDER BY prh.post_id, prh.synced_at DESC
) prh;

-- name: GetTotalStatsAtDate :one
SELECT
    COALESCE(SUM(prh.likes), 0)::BIGINT AS total_likes,
    COALESCE(SUM(prh.reposts), 0)::BIGINT AS total_reposts,
    COALESCE(SUM(prh.views), 0)::BIGINT AS total_views
FROM (
    SELECT DISTINCT ON (prh.post_id) prh.likes, prh.reposts, prh.views
    FROM posts_reactions_history prh
        JOIN posts p ON prh.post_id = p.id
        JOIN sources s ON p.source_id = s.id
    WHERE s.user_id = $1
        AND s.is_active = TRUE
        AND prh.synced_at <= $2
    ORDER BY prh.post_id, prh.synced_at DESC
) prh;

-- name: GetSourceStatusCounts :one
SELECT
    COUNT(*) FILTER (WHERE is_active = TRUE AND sync_status NOT IN ('Failed', 'Deactivated'))::BIGINT AS healthy_count,
    COUNT(*) FILTER (WHERE is_active = TRUE)::BIGINT AS enabled_count,
    COUNT(*) FILTER (WHERE is_active = FALSE)::BIGINT AS disabled_count
FROM sources
WHERE user_id = $1;

-- name: GetTargetStatusCounts :one
SELECT
    COUNT(*) FILTER (WHERE is_active = TRUE AND sync_status NOT IN ('Failed', 'Deactivated'))::BIGINT AS healthy_count,
    COUNT(*) FILTER (WHERE is_active = TRUE)::BIGINT AS enabled_count,
    COUNT(*) FILTER (WHERE is_active = FALSE)::BIGINT AS disabled_count
FROM targets
WHERE user_id = $1;

-- name: GetCurrentWebsiteStats :one
SELECT
    COALESCE((SELECT SUM(views) FROM analytics_page_stats aps JOIN sources s ON aps.source_id = s.id WHERE s.user_id = $1 AND aps.analytics_type = 'ga'), 0)::BIGINT AS total_page_views,
    COALESCE((SELECT SUM(visitors) FROM analytics_site_stats ass JOIN sources s ON ass.source_id = s.id WHERE s.user_id = $1 AND ass.analytics_type = 'ga'), 0)::BIGINT AS total_visitors,
    COALESCE((SELECT SUM(impressions) FROM analytics_site_stats ass JOIN sources s ON ass.source_id = s.id WHERE s.user_id = $1 AND ass.analytics_type = 'gsc'), 0)::BIGINT AS total_impressions;

-- name: GetWebsiteStatsAtDate :one
SELECT
    COALESCE((SELECT SUM(views) FROM analytics_page_stats aps JOIN sources s ON aps.source_id = s.id WHERE s.user_id = $1 AND aps.analytics_type = 'ga' AND aps.date <= $2), 0)::BIGINT AS total_page_views,
    COALESCE((SELECT SUM(visitors) FROM analytics_site_stats ass JOIN sources s ON ass.source_id = s.id WHERE s.user_id = $1 AND ass.analytics_type = 'ga' AND ass.date <= $2), 0)::BIGINT AS total_visitors,
    COALESCE((SELECT SUM(impressions) FROM analytics_site_stats ass JOIN sources s ON ass.source_id = s.id WHERE s.user_id = $1 AND ass.analytics_type = 'gsc' AND ass.date <= $2), 0)::BIGINT AS total_impressions;
