-- +goose Up

ALTER TABLE tokens
ADD COLUMN profile_id TEXT;


-- +goose Down

ALTER TABLE tokens
DROP COLUMN profile_id;