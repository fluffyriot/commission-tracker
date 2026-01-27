-- +goose Up
ALTER TABLE sources_on_target
RENAME CONSTRAINT fk_posts TO fk_sources;

-- +goose Down
ALTER TABLE sources_on_target
RENAME CONSTRAINT fk_sources TO fk_posts;