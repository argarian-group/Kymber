-- +goose Up
ALTER TABLE project_tasks ADD COLUMN archived_at TIMESTAMP NULL DEFAULT NULL;

-- +goose Down
ALTER TABLE project_tasks DROP COLUMN archived_at;
