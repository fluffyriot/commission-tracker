-- name: GetWordCloudData :many
SELECT
    cleaned_word as word,
    count(*) as count
FROM (
    SELECT regexp_replace(lower(regexp_split_to_table(content, '\s+')), '[^a-z0-9]', '', 'g') as cleaned_word,
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
    'that', 'have', 'with', 'this', 'from', 'they',  'will', 'would', 'there', 'their', 
    'about', 'which', 'when', 'make', 'like', 'time', 'just', 'know', 'take', 'what',
    'people', 'into', 'year', 'your', 'good', 'some', 'could', 'them', 'other', 'cant',
    'than', 'then', 'look', 'only', 'come', 'over', 'think', 'also', 'back', 'after', 
    'work', 'first', 'well', 'even', 'want', 'because', 'these', 'give', 'most',
    'were', 'been', 'here',  'many', 'dont', 'does', 'more', 'less'
)
GROUP BY cleaned_word
ORDER BY count DESC
LIMIT 50;

-- name: GetHashtagAnalytics :many
SELECT
    substring(word from 2) as tag,
    count(*) as usage_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.views), 0)::BIGINT as avg_views
FROM (
    SELECT
        regexp_split_to_table(lower(content), '\s+') as word,
        posts.id as post_id
    FROM posts
    JOIN sources s ON posts.source_id = s.id
    WHERE s.user_id = $1
    AND content IS NOT NULL
) t
LEFT JOIN (
    SELECT DISTINCT ON (post_id) post_id, likes, views
    FROM posts_reactions_history
    ORDER BY post_id, synced_at DESC
) prh ON t.post_id = prh.post_id
WHERE word LIKE '#%' AND length(word) > 1
GROUP BY tag
ORDER BY usage_count DESC
LIMIT 20;

-- name: GetMentionsAnalytics :many
SELECT
    regexp_replace(substring(word from 2), '[^a-z0-9_.]', '', 'g') as mention,
    count(*) as usage_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes
FROM (
    SELECT
        regexp_split_to_table(lower(content), '\s+') as word,
        posts.id as post_id
    FROM posts
    JOIN sources s ON posts.source_id = s.id
    WHERE s.user_id = $1
    AND content IS NOT NULL
) t
LEFT JOIN (
    SELECT DISTINCT ON (post_id) post_id, likes
    FROM posts_reactions_history
    ORDER BY post_id, synced_at DESC
) prh ON t.post_id = prh.post_id
WHERE word LIKE '@%'
GROUP BY mention
HAVING length(regexp_replace(substring(word from 2), '[^a-z0-9_.]', '', 'g')) > 1
ORDER BY avg_likes DESC
LIMIT 20;

-- name: GetTimePerformance :many
SELECT
    EXTRACT(DOW FROM p.created_at)::INT as day_of_week,
    EXTRACT(HOUR FROM p.created_at)::INT as hour_of_day,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes
FROM posts p
JOIN sources s ON p.source_id = s.id
LEFT JOIN (
    SELECT DISTINCT ON (post_id) post_id, likes
    FROM posts_reactions_history
    ORDER BY post_id, synced_at DESC
) prh ON p.id = prh.post_id
WHERE s.user_id = $1
GROUP BY day_of_week, hour_of_day
ORDER BY day_of_week, hour_of_day;

-- name: GetGlobalPostTypeAnalytics :many
SELECT
    post_type,
    count(*) as post_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes
FROM posts p
JOIN sources s ON p.source_id = s.id
LEFT JOIN (
    SELECT DISTINCT ON (post_id) post_id, likes
    FROM posts_reactions_history
    ORDER BY post_id, synced_at DESC
) prh ON p.id = prh.post_id
WHERE s.user_id = $1
GROUP BY post_type
ORDER BY avg_likes DESC;

-- name: GetNetworkEfficiency :many
SELECT
    s.network,
    count(*) as post_count,
    COALESCE(AVG(prh.likes), 0)::BIGINT as avg_likes,
    COALESCE(AVG(prh.reposts), 0)::BIGINT as avg_reposts
FROM posts p
JOIN sources s ON p.source_id = s.id
LEFT JOIN (
    SELECT DISTINCT ON (post_id) post_id, likes, reposts
    FROM posts_reactions_history
    ORDER BY post_id, synced_at DESC
) prh ON p.id = prh.post_id
WHERE s.user_id = $1
GROUP BY s.network
ORDER BY avg_likes DESC;

-- name: GetTopPagesByViews :many
SELECT
    url_path,
    COALESCE(SUM(views), 0)::BIGINT as total_views
FROM analytics_page_stats aps
JOIN sources s ON aps.source_id = s.id
WHERE s.user_id = $1
GROUP BY url_path
ORDER BY total_views DESC
LIMIT 50;

-- name: GetSiteStatsOverTime :many
SELECT
    TO_CHAR(date, 'YYYY-MM-DD') as date_str,
    COALESCE(SUM(visitors), 0)::BIGINT as total_visitors,
    COALESCE(AVG(avg_session_duration), 0)::FLOAT as avg_session_duration
FROM analytics_site_stats ass
JOIN sources s ON ass.source_id = s.id
WHERE s.user_id = $1
GROUP BY date_str
ORDER BY date_str ASC
LIMIT 90;

-- name: GetPostingConsistency :many
SELECT
    TO_CHAR(p.created_at, 'YYYY-MM-DD') as date_str,
    count(*) as post_count
FROM posts p
JOIN sources s ON p.source_id = s.id
WHERE s.user_id = $1
AND p.created_at > NOW() - INTERVAL '1 year'
GROUP BY date_str
ORDER BY date_str;
