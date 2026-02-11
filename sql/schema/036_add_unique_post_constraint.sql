-- +goose Up
ALTER TABLE posts
ADD CONSTRAINT unique_posts_source_id_network_internal_id UNIQUE (
    source_id,
    network_internal_id
);

-- +goose Down
ALTER TABLE posts
DROP CONSTRAINT IF EXISTS unique_posts_source_id_network_internal_id;