-- name: GetSiteStatsOverTime :many
SELECT *
FROM (
        SELECT TO_CHAR(DATE_TRUNC('week', date), 'IYYY-"W"IW') as date_str,
            COALESCE(SUM(visitors), 0)::BIGINT as total_visitors,
            COALESCE(AVG(avg_session_duration), 0)::FLOAT as avg_session_duration
        FROM analytics_site_stats ass
            JOIN sources s ON ass.source_id = s.id
        WHERE s.user_id = $1
            AND ass.analytics_type = 'ga'
        GROUP BY DATE_TRUNC('week', date)
        ORDER BY DATE_TRUNC('week', date) DESC
        LIMIT 52
    ) recent_weeks
ORDER BY date_str ASC;

-- name: GetTopPagesByViews :many
SELECT url_path,
    COALESCE(SUM(views), 0)::BIGINT as total_views
FROM analytics_page_stats aps
    JOIN sources s ON aps.source_id = s.id
WHERE s.user_id = $1
    AND aps.analytics_type = 'ga'
GROUP BY url_path
ORDER BY total_views DESC
LIMIT 50;

-- name: GetTotalSiteStats :one
SELECT COALESCE(SUM(visitors), 0)::BIGINT AS total_visitors
FROM analytics_site_stats
    LEFT JOIN sources ON analytics_site_stats.source_id = sources.id
WHERE sources.user_id = $1
    AND analytics_site_stats.analytics_type = 'ga';

-- name: GetTotalPageViews :one
SELECT COALESCE(SUM(views), 0)::BIGINT AS total_page_views
FROM analytics_page_stats
    LEFT JOIN sources ON analytics_page_stats.source_id = sources.id
WHERE sources.user_id = $1
    AND analytics_page_stats.analytics_type = 'ga';

-- name: GetAverageWebsiteSession :one
SELECT COALESCE(AVG(avg_session_duration), 0)::BIGINT AS average_website_session
FROM analytics_site_stats
    LEFT JOIN sources ON analytics_site_stats.source_id = sources.id
WHERE sources.user_id = $1
    AND analytics_site_stats.analytics_type = 'ga';

-- name: GetSiteStatsOverTimeFiltered :many
SELECT *
FROM (
        SELECT TO_CHAR(DATE_TRUNC('week', date), 'IYYY-"W"IW') as date_str,
            COALESCE(SUM(visitors), 0)::BIGINT as total_visitors,
            COALESCE(AVG(avg_session_duration), 0)::FLOAT as avg_session_duration
        FROM analytics_site_stats ass
            JOIN sources s ON ass.source_id = s.id
        WHERE s.user_id = @user_id
            AND ass.analytics_type = 'ga'
            AND (sqlc.narg('start_date')::date IS NULL OR ass.date >= sqlc.narg('start_date')::date)
            AND (sqlc.narg('end_date')::date IS NULL OR ass.date <= sqlc.narg('end_date')::date)
        GROUP BY DATE_TRUNC('week', date)
        ORDER BY DATE_TRUNC('week', date) DESC
        LIMIT 52
    ) recent_weeks
ORDER BY date_str ASC;

-- name: GetTopPagesByViewsFiltered :many
SELECT url_path,
    COALESCE(SUM(views), 0)::BIGINT as total_views
FROM analytics_page_stats aps
    JOIN sources s ON aps.source_id = s.id
WHERE s.user_id = @user_id
    AND aps.analytics_type = 'ga'
    AND (sqlc.narg('start_date')::date IS NULL OR aps.date >= sqlc.narg('start_date')::date)
    AND (sqlc.narg('end_date')::date IS NULL OR aps.date <= sqlc.narg('end_date')::date)
GROUP BY url_path
ORDER BY total_views DESC
LIMIT 50;

-- name: GetGSCSiteStatsOverTime :many
SELECT *
FROM (
        SELECT TO_CHAR(DATE_TRUNC('week', date), 'IYYY-"W"IW') as date_str,
            COALESCE(SUM(visitors), 0)::BIGINT as total_clicks,
            COALESCE(SUM(impressions), 0)::BIGINT as total_impressions
        FROM analytics_site_stats ass
            JOIN sources s ON ass.source_id = s.id
        WHERE s.user_id = $1
            AND ass.analytics_type = 'gsc'
        GROUP BY DATE_TRUNC('week', date)
        ORDER BY DATE_TRUNC('week', date) DESC
        LIMIT 52
    ) recent_weeks
ORDER BY date_str ASC;

-- name: GetGSCTopPagesByClicks :many
SELECT url_path,
    COALESCE(SUM(views), 0)::BIGINT as total_clicks,
    COALESCE(SUM(impressions), 0)::BIGINT as total_impressions
FROM analytics_page_stats aps
    JOIN sources s ON aps.source_id = s.id
WHERE s.user_id = $1
    AND aps.analytics_type = 'gsc'
GROUP BY url_path
ORDER BY total_clicks DESC
LIMIT 50;
