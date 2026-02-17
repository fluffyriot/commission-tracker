-- name: GetWordCloudData :many
SELECT cleaned_word as word,
    count(*) as count
FROM (
        SELECT regexp_replace(
                lower(regexp_split_to_table(content, '\s+')),
                '[^a-z0-9]',
                '',
                'g'
            ) as cleaned_word,
            regexp_split_to_table(content, '\s+') as raw_word
        FROM posts
            JOIN sources s ON posts.source_id = s.id
        WHERE s.user_id = $1
            AND content IS NOT NULL
    ) t
WHERE length(cleaned_word) > 3
    AND raw_word NOT LIKE '#%'
    AND raw_word NOT LIKE '@%'
    AND raw_word NOT LIKE 'http%'
    AND raw_word NOT ILIKE '%http%'
    AND raw_word NOT ILIKE '%www.%'
    AND raw_word NOT ILIKE '%.com%'
    AND cleaned_word NOT ILIKE '%iotpho%'
    AND cleaned_word NOT IN (
        'that',
        'have',
        'with',
        'this',
        'from',
        'they',
        'will',
        'would',
        'there',
        'their',
        'about',
        'which',
        'when',
        'make',
        'like',
        'time',
        'just',
        'know',
        'take',
        'what',
        'people',
        'into',
        'year',
        'your',
        'good',
        'some',
        'could',
        'them',
        'other',
        'cant',
        'than',
        'then',
        'look',
        'only',
        'come',
        'over',
        'think',
        'also',
        'back',
        'after',
        'work',
        'first',
        'well',
        'even',
        'want',
        'because',
        'these',
        'give',
        'most',
        'were',
        'been',
        'here',
        'many',
        'dont',
        'does',
        'more',
        'less'
    )
GROUP BY cleaned_word
ORDER BY count DESC
LIMIT 50;
-- name: GetHashtagAnalytics :many
SELECT substring(
        word
        from 2
    ) as tag,
    count(*) as usage_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.views), 0)::BIGINT as avg_views
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
            likes,
            views
        FROM posts_reactions_history
        ORDER BY post_id,
            synced_at DESC
    ) prh ON t.post_id = prh.post_id
WHERE word LIKE '#%'
    AND length(word) > 1
GROUP BY tag
ORDER BY usage_count DESC
LIMIT 20;
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
-- name: GetTimePerformance :many
SELECT EXTRACT(
        DOW
        FROM p.created_at
    )::INT as day_of_week,
    EXTRACT(
        HOUR
        FROM p.created_at
    )::INT as hour_of_day,
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
GROUP BY day_of_week,
    hour_of_day
ORDER BY day_of_week,
    hour_of_day;
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
-- name: GetTopPagesByViews :many
SELECT url_path,
    COALESCE(SUM(views), 0)::BIGINT as total_views
FROM analytics_page_stats aps
    JOIN sources s ON aps.source_id = s.id
WHERE s.user_id = $1
GROUP BY url_path
ORDER BY total_views DESC
LIMIT 50;
-- name: GetSiteStatsOverTime :many
SELECT *
FROM (
        SELECT TO_CHAR(DATE_TRUNC('week', date), 'IYYY-"W"IW') as date_str,
            COALESCE(SUM(visitors), 0)::BIGINT as total_visitors,
            COALESCE(AVG(avg_session_duration), 0)::FLOAT as avg_session_duration
        FROM analytics_site_stats ass
            JOIN sources s ON ass.source_id = s.id
        WHERE s.user_id = $1
        GROUP BY DATE_TRUNC('week', date)
        ORDER BY DATE_TRUNC('week', date) DESC
        LIMIT 52
    ) recent_weeks
ORDER BY date_str ASC;
-- name: GetPostingConsistency :many
SELECT TO_CHAR(p.created_at, 'YYYY-MM-DD') as date_str,
    count(*) as post_count
FROM posts p
    JOIN sources s ON p.source_id = s.id
WHERE s.user_id = $1
    AND p.created_at > NOW() - INTERVAL '1 year'
GROUP BY date_str
ORDER BY date_str;
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
-- name: GetPerformanceDeviationData :many
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
    s.network,
    COALESCE(prh.likes, 0)::BIGINT as likes,
    COALESCE(prh.reposts, 0)::BIGINT as reposts,
    sa.avg_engagement::FLOAT as avg_engagement
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
    AND (
        COALESCE(prh.likes, 0) + COALESCE(prh.reposts, 0)
    ) > (sa.avg_engagement * 1.5)
ORDER BY (
        (
            COALESCE(prh.likes, 0) + COALESCE(prh.reposts, 0)
        ) - sa.avg_engagement
    ) DESC
LIMIT 10;
-- name: GetEngagementVelocityData :many
SELECT prh.post_id,
    prh.synced_at as history_synced_at,
    COALESCE(prh.likes, 0)::BIGINT as likes,
    COALESCE(prh.reposts, 0)::BIGINT as reposts,
    p.created_at as post_created_at,
    COALESCE(p.content, '')::TEXT as content
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
-- name: GetCollaborationsData :many
SELECT collaborator,
    COUNT(*) as collaboration_count,
    COALESCE(AVG(likes), 0)::BIGINT as avg_likes
FROM (
        -- Reposts/Quotes where author is the collaborator
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