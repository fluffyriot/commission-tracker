-- name: GetTagAnalytics :many
SELECT t.id as tag_id,
    t.name as tag_name,
    tc.name as classification_name,
    COUNT(DISTINCT p.id)::BIGINT as post_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.reposts), 0)::BIGINT as avg_reposts,
    COALESCE(AVG(prh.views), 0)::BIGINT as avg_views,
    COALESCE(SUM(prh.likes), 0)::BIGINT as total_likes,
    COALESCE(SUM(prh.views), 0)::BIGINT as total_views
FROM tags t
    LEFT JOIN tag_classifications tc ON t.classification_id = tc.id
    JOIN post_tags pt ON t.id = pt.tag_id
    JOIN posts p ON pt.post_id = p.id
    JOIN sources s ON p.source_id = s.id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            reposts,
            views
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
WHERE s.user_id = $1
GROUP BY t.id, t.name, tc.name
ORDER BY avg_likes DESC;

-- name: GetTagAnalyticsFiltered :many
SELECT t.id as tag_id,
    t.name as tag_name,
    tc.name as classification_name,
    COUNT(DISTINCT p.id)::BIGINT as post_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.reposts), 0)::BIGINT as avg_reposts,
    COALESCE(AVG(prh.views), 0)::BIGINT as avg_views,
    COALESCE(SUM(prh.likes), 0)::BIGINT as total_likes,
    COALESCE(SUM(prh.views), 0)::BIGINT as total_views
FROM tags t
    LEFT JOIN tag_classifications tc ON t.classification_id = tc.id
    JOIN post_tags pt ON t.id = pt.tag_id
    JOIN posts p ON pt.post_id = p.id
    JOIN sources s ON p.source_id = s.id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            reposts,
            views
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
WHERE s.user_id = @user_id
    AND (sqlc.narg('start_date')::date IS NULL OR p.created_at >= sqlc.narg('start_date')::date)
    AND (sqlc.narg('end_date')::date IS NULL OR p.created_at < sqlc.narg('end_date')::date + INTERVAL '1 day')
    AND (array_length(@post_types::text[], 1) IS NULL OR p.post_type = ANY(@post_types::text[]))
    AND (array_length(@tag_ids::uuid[], 1) IS NULL OR pt.tag_id = ANY(@tag_ids::uuid[]))
GROUP BY t.id, t.name, tc.name
ORDER BY avg_likes DESC;

-- name: GetClassificationAnalytics :many
SELECT tc.id as classification_id,
    tc.name as classification_name,
    COUNT(DISTINCT p.id)::BIGINT as post_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.reposts), 0)::BIGINT as avg_reposts,
    COALESCE(AVG(prh.views), 0)::BIGINT as avg_views,
    COALESCE(SUM(prh.likes), 0)::BIGINT as total_likes,
    COALESCE(SUM(prh.views), 0)::BIGINT as total_views
FROM tag_classifications tc
    JOIN tags t ON tc.id = t.classification_id
    JOIN post_tags pt ON t.id = pt.tag_id
    JOIN posts p ON pt.post_id = p.id
    JOIN sources s ON p.source_id = s.id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            reposts,
            views
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
WHERE tc.user_id = $1
GROUP BY tc.id, tc.name
ORDER BY avg_likes DESC;

-- name: GetClassificationAnalyticsFiltered :many
SELECT tc.id as classification_id,
    tc.name as classification_name,
    COUNT(DISTINCT p.id)::BIGINT as post_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.reposts), 0)::BIGINT as avg_reposts,
    COALESCE(AVG(prh.views), 0)::BIGINT as avg_views,
    COALESCE(SUM(prh.likes), 0)::BIGINT as total_likes,
    COALESCE(SUM(prh.views), 0)::BIGINT as total_views
FROM tag_classifications tc
    JOIN tags t ON tc.id = t.classification_id
    JOIN post_tags pt ON t.id = pt.tag_id
    JOIN posts p ON pt.post_id = p.id
    JOIN sources s ON p.source_id = s.id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            reposts,
            views
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
WHERE tc.user_id = @user_id
    AND (sqlc.narg('start_date')::date IS NULL OR p.created_at >= sqlc.narg('start_date')::date)
    AND (sqlc.narg('end_date')::date IS NULL OR p.created_at < sqlc.narg('end_date')::date + INTERVAL '1 day')
    AND (array_length(@post_types::text[], 1) IS NULL OR p.post_type = ANY(@post_types::text[]))
    AND (array_length(@tag_ids::uuid[], 1) IS NULL OR pt.tag_id = ANY(@tag_ids::uuid[]))
GROUP BY tc.id, tc.name
ORDER BY avg_likes DESC;
