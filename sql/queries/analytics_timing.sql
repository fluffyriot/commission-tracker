-- name: GetTimePerformance :many
SELECT EXTRACT(
        DOW
        FROM p.created_at
    )::INT as day_of_week,
    EXTRACT(
        HOUR
        FROM p.created_at
    )::INT as hour_of_day,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.views), 0)::BIGINT as avg_views
FROM posts p
    JOIN sources s ON p.source_id = s.id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            views
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
WHERE s.user_id = $1
GROUP BY day_of_week,
    hour_of_day
ORDER BY day_of_week,
    hour_of_day;
-- name: GetPostingConsistency :many
SELECT TO_CHAR(p.created_at, 'YYYY-MM-DD') as date_str,
    count(*) as post_count
FROM posts p
    JOIN sources s ON p.source_id = s.id
WHERE s.user_id = $1
    AND p.created_at > NOW() - INTERVAL '1 year'
GROUP BY date_str
ORDER BY date_str;
-- name: GetTimePerformanceFiltered :many
SELECT EXTRACT(
        DOW
        FROM p.created_at
    )::INT as day_of_week,
    EXTRACT(
        HOUR
        FROM p.created_at
    )::INT as hour_of_day,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.views), 0)::BIGINT as avg_views
FROM posts p
    JOIN sources s ON p.source_id = s.id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            views
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
WHERE s.user_id = @user_id
    AND (sqlc.narg('start_date')::date IS NULL OR p.created_at >= sqlc.narg('start_date')::date)
    AND (sqlc.narg('end_date')::date IS NULL OR p.created_at < sqlc.narg('end_date')::date + INTERVAL '1 day')
    AND (array_length(@post_types::text[], 1) IS NULL OR p.post_type = ANY(@post_types::text[]))
GROUP BY day_of_week,
    hour_of_day
ORDER BY day_of_week,
    hour_of_day;
-- name: GetPostingConsistencyFiltered :many
SELECT TO_CHAR(p.created_at, 'YYYY-MM-DD') as date_str,
    count(*) as post_count
FROM posts p
    JOIN sources s ON p.source_id = s.id
WHERE s.user_id = @user_id
    AND p.created_at > NOW() - INTERVAL '1 year'
    AND (sqlc.narg('start_date')::date IS NULL OR p.created_at >= sqlc.narg('start_date')::date)
    AND (sqlc.narg('end_date')::date IS NULL OR p.created_at < sqlc.narg('end_date')::date + INTERVAL '1 day')
    AND (array_length(@post_types::text[], 1) IS NULL OR p.post_type = ANY(@post_types::text[]))
GROUP BY date_str
ORDER BY date_str;