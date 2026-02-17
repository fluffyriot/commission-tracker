-- +goose Up
ALTER TABLE sources_on_target DROP CONSTRAINT IF EXISTS sources_on_target_target_source_id_key;
ALTER TABLE sources_on_target
ADD CONSTRAINT sources_on_target_target_source_unique UNIQUE (target_id, target_source_id);
ALTER TABLE sources_stats_on_target DROP CONSTRAINT IF EXISTS sources_stats_on_target_target_record_id_key;
ALTER TABLE sources_stats_on_target
ADD CONSTRAINT sources_stats_on_target_target_record_unique UNIQUE (target_id, target_record_id);
ALTER TABLE posts_on_target DROP CONSTRAINT IF EXISTS posts_on_target_target_post_id_key;
ALTER TABLE posts_on_target
ADD CONSTRAINT posts_on_target_target_post_unique UNIQUE (target_id, target_post_id);
-- +goose Down
ALTER TABLE sources_on_target DROP CONSTRAINT IF EXISTS sources_on_target_target_source_unique;
ALTER TABLE sources_on_target
ADD CONSTRAINT sources_on_target_target_source_id_key UNIQUE (target_source_id);
ALTER TABLE sources_stats_on_target DROP CONSTRAINT IF EXISTS sources_stats_on_target_target_record_unique;
ALTER TABLE sources_stats_on_target
ADD CONSTRAINT sources_stats_on_target_target_record_id_key UNIQUE (target_record_id);
ALTER TABLE posts_on_target DROP CONSTRAINT IF EXISTS posts_on_target_target_post_unique;
ALTER TABLE posts_on_target
ADD CONSTRAINT posts_on_target_target_post_id_key UNIQUE (target_post_id);