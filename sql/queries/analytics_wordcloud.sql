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