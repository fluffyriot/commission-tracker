-- +goose Up
ALTER TABLE tokens
ADD COLUMN source_app_data JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE tokens DROP COLUMN source_app_data;