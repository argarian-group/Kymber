-- +goose Up
ALTER TABLE service_git_sources
    ADD COLUMN encrypted_webhook_secret BLOB NULL;

-- +goose Down
ALTER TABLE service_git_sources
    DROP COLUMN encrypted_webhook_secret;
