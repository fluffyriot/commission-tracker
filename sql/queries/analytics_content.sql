-- name: GetHashtagAnalytics :many
SELECT lower(matches [1]) as tag,
    count(*) as usage_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.views), 0)::BIGINT as avg_views
FROM (
        SELECT regexp_matches(content, '#([[:alnum:]_]+)', 'g') as matches,
            posts.id as post_id
        FROM posts
            JOIN sources s ON posts.source_id = s.id
        WHERE s.user_id = $1
            AND content IS NOT NULL
    ) t
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            views
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON t.post_id = prh.post_id
GROUP BY tag
ORDER BY avg_likes DESC
LIMIT 20;
-- name: GetPerformanceDeviationPositive :many
WITH SourceAverages AS (
    SELECT p.source_id,
        AVG(
            COALESCE(prh.likes, 0) + COALESCE(prh.reposts, 0)
        ) as avg_engagement
    FROM posts p
        LEFT JOIN (
            SELECT DISTINCT ON (post_id) post_id,
                likes,
                reposts
            FROM posts_reactions_history
            ORDER BY post_id,
                synced_at DESC
        ) prh ON p.id = prh.post_id
    GROUP BY p.source_id
)
SELECT p.id,
    p.network_internal_id,
    COALESCE(p.content, '')::TEXT as content,
    p.created_at,
    p.author,
    s.network,
    COALESCE(prh.likes, 0)::BIGINT as likes,
    COALESCE(prh.reposts, 0)::BIGINT as reposts,
    (
        sa.avg_engagement * LEAST(
            1.0,
            EXTRACT(
                EPOCH
                FROM (NOW() - p.created_at)
            ) / 86400.0
        )
    )::FLOAT as expected_engagement
FROM posts p
    JOIN sources s ON p.source_id = s.id
    JOIN SourceAverages sa ON p.source_id = sa.source_id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            reposts
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
WHERE s.user_id = $1
    AND p.post_type NOT IN ('tag', 'repost', 'quote')
    AND (COALESCE(prh.likes, 0) + COALESCE(prh.reposts, 0)) > (
        sa.avg_engagement * LEAST(
            1.0,
            EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 86400.0
        )
    )
    AND p.created_at > NOW() - INTERVAL '90 days'
ORDER BY (
        (COALESCE(prh.likes, 0) + COALESCE(prh.reposts, 0)) - (
            sa.avg_engagement * LEAST(
                1.0,
                EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 86400.0
            )
        )
    ) DESC
LIMIT 7;
-- name: GetPerformanceDeviationNegative :many
WITH SourceAverages AS (
    SELECT p.source_id,
        AVG(
            COALESCE(prh.likes, 0) + COALESCE(prh.reposts, 0)
        ) as avg_engagement
    FROM posts p
        LEFT JOIN (
            SELECT DISTINCT ON (post_id) post_id,
                likes,
                reposts
            FROM posts_reactions_history
            ORDER BY post_id,
                synced_at DESC
        ) prh ON p.id = prh.post_id
    GROUP BY p.source_id
)
SELECT p.id,
    p.network_internal_id,
    COALESCE(p.content, '')::TEXT as content,
    p.created_at,
    p.author,
    s.network,
    COALESCE(prh.likes, 0)::BIGINT as likes,
    COALESCE(prh.reposts, 0)::BIGINT as reposts,
    (
        sa.avg_engagement * LEAST(
            1.0,
            EXTRACT(
                EPOCH
                FROM (NOW() - p.created_at)
            ) / 86400.0
        )
    )::FLOAT as expected_engagement
FROM posts p
    JOIN sources s ON p.source_id = s.id
    JOIN SourceAverages sa ON p.source_id = sa.source_id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            reposts
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
WHERE s.user_id = $1
    AND p.post_type NOT IN ('tag', 'repost', 'quote')
    AND (COALESCE(prh.likes, 0) + COALESCE(prh.reposts, 0)) < (
        sa.avg_engagement * LEAST(
            1.0,
            EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 86400.0
        )
    )
    AND p.created_at > NOW() - INTERVAL '90 days'
ORDER BY (
        (COALESCE(prh.likes, 0) + COALESCE(prh.reposts, 0)) - (
            sa.avg_engagement * LEAST(
                1.0,
                EXTRACT(EPOCH FROM (NOW() - p.created_at)) / 86400.0
            )
        )
    ) ASC
LIMIT 7;
-- name: GetEngagementVelocityData :many
SELECT prh.post_id,
    prh.synced_at as history_synced_at,
    COALESCE(prh.likes, 0)::BIGINT as likes,
    COALESCE(prh.reposts, 0)::BIGINT as reposts,
    p.created_at as post_created_at,
    COALESCE(p.content, '')::TEXT as content,
    p.author,
    p.network_internal_id,
    s.network
FROM posts_reactions_history prh
    JOIN posts p ON prh.post_id = p.id
    JOIN sources s ON p.source_id = s.id
    JOIN (
        SELECT post_id,
            MIN(synced_at) as first_synced
        FROM posts_reactions_history
        GROUP BY post_id
    ) first_sync ON p.id = first_sync.post_id
WHERE s.user_id = $1
    AND p.created_at > NOW() - INTERVAL '30 days'
    AND p.post_type NOT IN ('tag', 'repost', 'quote')
    AND DATE(p.created_at) = DATE(first_sync.first_synced)
ORDER BY prh.post_id,
    prh.synced_at ASC;