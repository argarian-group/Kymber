-- +goose Up
ALTER TABLE project_tasks ADD COLUMN start_date DATE NULL DEFAULT NULL AFTER priority;

-- +goose Down
ALTER TABLE project_tasks DROP COLUMN start_date;
