-- name: GetWordCloudEngagementData :many
WITH post_engagement AS (
    SELECT p.id,
        p.content,
        COALESCE(prh.likes, 0) + COALESCE(prh.reposts, 0) as engagement
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
        AND p.content IS NOT NULL
),
word_stats AS (
    SELECT regexp_replace(
            lower(regexp_split_to_table(pe.content, '\s+')),
            '[^a-z0-9]',
            '',
            'g'
        ) as cleaned_word,
        regexp_split_to_table(pe.content, '\s+') as raw_word,
        pe.engagement
    FROM post_engagement pe
)
SELECT cleaned_word as word,
    SUM(engagement)::BIGINT as total_engagement,
    (CAST(SUM(engagement) AS FLOAT) / COUNT(*))::FLOAT as avg_engagement,
    COUNT(*) as usage_count
FROM word_stats
WHERE length(cleaned_word) > 3
    AND raw_word NOT LIKE '#%'
    AND raw_word NOT LIKE '@%'
    AND raw_word NOT LIKE 'r/%'
    AND raw_word NOT LIKE 'http%'
    AND raw_word NOT ILIKE '%http%'
    AND raw_word NOT ILIKE '%www.%'
    AND raw_word NOT ILIKE '%.net%'
    AND raw_word NOT ILIKE '%.org%'
    AND raw_word NOT ILIKE '%.co.uk%'
    AND raw_word NOT ILIKE '%.com%'
    AND raw_word NOT ILIKE '%.me/%'
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
ORDER BY avg_engagement DESC
LIMIT 50;
-- name: GetWordCloudData :many
WITH word_stats AS (
    SELECT regexp_replace(
            lower(regexp_split_to_table(p.content, '\s+')),
            '[^a-z0-9]',
            '',
            'g'
        ) as cleaned_word,
        regexp_split_to_table(p.content, '\s+') as raw_word
    FROM posts p
        JOIN sources s ON p.source_id = s.id
    WHERE s.user_id = $1
        AND p.content IS NOT NULL
)
SELECT cleaned_word as word,
    COUNT(*) as usage_count
FROM word_stats
WHERE length(cleaned_word) > 3
    AND raw_word NOT LIKE '#%'
    AND raw_word NOT LIKE '@%'
    AND raw_word NOT LIKE 'r/%'
    AND raw_word NOT LIKE 'http%'
    AND raw_word NOT ILIKE '%http%'
    AND raw_word NOT ILIKE '%www.%'
    AND raw_word NOT ILIKE '%.net%'
    AND raw_word NOT ILIKE '%.org%'
    AND raw_word NOT ILIKE '%.co.uk%'
    AND raw_word NOT ILIKE '%.com%'
    AND raw_word NOT ILIKE '%.me/%'
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
ORDER BY usage_count DESC
LIMIT 50;
-- name: GetWordCloudEngagementDataFiltered :many
WITH post_engagement AS (
    SELECT p.id,
        p.content,
        COALESCE(prh.likes, 0) + COALESCE(prh.reposts, 0) as engagement
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
    WHERE s.user_id = @user_id
        AND p.content IS NOT NULL
        AND (sqlc.narg('start_date')::date IS NULL OR p.created_at >= sqlc.narg('start_date')::date)
        AND (sqlc.narg('end_date')::date IS NULL OR p.created_at < sqlc.narg('end_date')::date + INTERVAL '1 day')
        AND (array_length(@post_types::text[], 1) IS NULL OR p.post_type = ANY(@post_types::text[]))
),
word_stats AS (
    SELECT regexp_replace(
            lower(regexp_split_to_table(pe.content, '\s+')),
            '[^a-z0-9]',
            '',
            'g'
        ) as cleaned_word,
        regexp_split_to_table(pe.content, '\s+') as raw_word,
        pe.engagement
    FROM post_engagement pe
)
SELECT cleaned_word as word,
    SUM(engagement)::BIGINT as total_engagement,
    (CAST(SUM(engagement) AS FLOAT) / COUNT(*))::FLOAT as avg_engagement,
    COUNT(*) as usage_count
FROM word_stats
WHERE length(cleaned_word) > 3
    AND raw_word NOT LIKE '#%'
    AND raw_word NOT LIKE '@%'
    AND raw_word NOT LIKE 'r/%'
    AND raw_word NOT LIKE 'http%'
    AND raw_word NOT ILIKE '%http%'
    AND raw_word NOT ILIKE '%www.%'
    AND raw_word NOT ILIKE '%.net%'
    AND raw_word NOT ILIKE '%.org%'
    AND raw_word NOT ILIKE '%.co.uk%'
    AND raw_word NOT ILIKE '%.com%'
    AND raw_word NOT ILIKE '%.me/%'
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
ORDER BY avg_engagement DESC
LIMIT 50;
-- name: GetWordCloudDataFiltered :many
WITH word_stats AS (
    SELECT regexp_replace(
            lower(regexp_split_to_table(p.content, '\s+')),
            '[^a-z0-9]',
            '',
            'g'
        ) as cleaned_word,
        regexp_split_to_table(p.content, '\s+') as raw_word
    FROM posts p
        JOIN sources s ON p.source_id = s.id
    WHERE s.user_id = @user_id
        AND p.content IS NOT NULL
        AND (sqlc.narg('start_date')::date IS NULL OR p.created_at >= sqlc.narg('start_date')::date)
        AND (sqlc.narg('end_date')::date IS NULL OR p.created_at < sqlc.narg('end_date')::date + INTERVAL '1 day')
        AND (array_length(@post_types::text[], 1) IS NULL OR p.post_type = ANY(@post_types::text[]))
)
SELECT cleaned_word as word,
    COUNT(*) as usage_count
FROM word_stats
WHERE length(cleaned_word) > 3
    AND raw_word NOT LIKE '#%'
    AND raw_word NOT LIKE '@%'
    AND raw_word NOT LIKE 'r/%'
    AND raw_word NOT LIKE 'http%'
    AND raw_word NOT ILIKE '%http%'
    AND raw_word NOT ILIKE '%www.%'
    AND raw_word NOT ILIKE '%.net%'
    AND raw_word NOT ILIKE '%.org%'
    AND raw_word NOT ILIKE '%.co.uk%'
    AND raw_word NOT ILIKE '%.com%'
    AND raw_word NOT ILIKE '%.me/%'
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
ORDER BY usage_count DESC
LIMIT 50;