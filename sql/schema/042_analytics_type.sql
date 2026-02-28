-- +goose Up
ALTER TABLE analytics_site_stats
    ADD COLUMN analytics_type TEXT NOT NULL DEFAULT 'ga',
    ADD COLUMN impressions BIGINT;

ALTER TABLE analytics_page_stats
    ADD COLUMN analytics_type TEXT NOT NULL DEFAULT 'ga',
    ADD COLUMN impressions BIGINT;

-- +goose Down
ALTER TABLE analytics_site_stats
    DROP COLUMN analytics_type,
    DROP COLUMN impressions;

ALTER TABLE analytics_page_stats
    DROP COLUMN analytics_type,
    DROP COLUMN impressions;
