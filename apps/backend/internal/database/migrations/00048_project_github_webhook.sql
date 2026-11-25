-- +goose Up
ALTER TABLE projects ADD COLUMN github_webhook_secret VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL DEFAULT NULL;

-- +goose Down
ALTER TABLE projects DROP COLUMN github_webhook_secret;
