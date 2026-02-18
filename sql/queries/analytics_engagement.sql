-- name: GetGlobalPostTypeAnalytics :many
SELECT post_type,
    count(*) as post_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes
FROM posts p
    JOIN sources s ON p.source_id = s.id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
WHERE s.user_id = $1
GROUP BY post_type
ORDER BY avg_likes DESC;
-- name: GetNetworkEfficiency :many
SELECT s.network,
    count(*) as post_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.reposts), 0)::BIGINT as avg_reposts
FROM posts p
    JOIN sources s ON p.source_id = s.id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            reposts
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
WHERE s.user_id = $1
GROUP BY s.network
ORDER BY avg_likes DESC;
-- name: GetMentionsAnalytics :many
SELECT regexp_replace(
        substring(
            word
            from 2
        ),
        '[^a-z0-9_.]',
        '',
        'g'
    ) as mention,
    count(*) as usage_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes
FROM (
        SELECT regexp_split_to_table(lower(content), '\s+') as word,
            posts.id as post_id
        FROM posts
            JOIN sources s ON posts.source_id = s.id
        WHERE s.user_id = $1
            AND content IS NOT NULL
    ) t
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON t.post_id = prh.post_id
WHERE word LIKE '@%'
GROUP BY mention
HAVING length(
        regexp_replace(
            substring(
                word
                from 2
            ),
            '[^a-z0-9_.]',
            '',
            'g'
        )
    ) > 1
ORDER BY avg_likes DESC
LIMIT 20;
-- name: GetCollaborationsData :many
SELECT collaborator,
    COUNT(*) as collaboration_count,
    COALESCE(AVG(likes), 0)::BIGINT as avg_likes
FROM (
        SELECT p.author as collaborator,
            COALESCE(prh.likes, 0) as likes
        FROM posts p
            JOIN sources s ON p.source_id = s.id
            LEFT JOIN (
                SELECT DISTINCT ON (post_id) post_id,
                    likes
                FROM posts_reactions_history
                ORDER BY post_id,
                    synced_at DESC
            ) prh ON p.id = prh.post_id
        WHERE s.user_id = $1
            AND p.post_type IN ('repost', 'tag')
            AND p.author IS NOT NULL
            AND p.author != ''
    ) combined_collaborations
GROUP BY collaborator
ORDER BY avg_likes DESC
LIMIT 50;
-- name: GetEngagementRateData :many
SELECT p.id,
    p.network_internal_id,
    p.created_at,
    s.network,
    COALESCE(prh.likes, 0)::BIGINT as likes,
    COALESCE(prh.reposts, 0)::BIGINT as reposts,
    COALESCE(ss.followers_count, 0)::BIGINT as followers_count
FROM posts p
    JOIN sources s ON p.source_id = s.id
    LEFT JOIN (
        SELECT DISTINCT ON (post_id) post_id,
            likes,
            reposts
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON p.id = prh.post_id
    LEFT JOIN LATERAL (
        SELECT followers_count
        FROM sources_stats
        WHERE source_id = s.id
            AND date <= p.created_at
        ORDER BY date DESC
        LIMIT 1
    ) ss ON true
WHERE s.user_id = $1
    AND p.created_at > NOW() - INTERVAL '6 months'
    AND ss.followers_count > 0;
-- name: GetFollowRatioData :many
SELECT s.network,
    s.user_name,
    COALESCE(ss.followers_count, 0)::BIGINT as followers_count,
    COALESCE(ss.following_count, 0)::BIGINT as following_count
FROM sources s
    JOIN LATERAL (
        SELECT followers_count,
            following_count
        FROM sources_stats
        WHERE source_id = s.id
        ORDER BY date DESC
        LIMIT 1
    ) ss ON true
WHERE s.user_id = $1
    AND ss.followers_count > 0
    AND ss.following_count > 0;