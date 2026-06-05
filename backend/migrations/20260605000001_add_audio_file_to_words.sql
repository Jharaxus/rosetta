-- +goose Up
ALTER TABLE words ADD COLUMN audio_file TEXT;

-- +goose Down
ALTER TABLE words DROP COLUMN IF EXISTS audio_file;
