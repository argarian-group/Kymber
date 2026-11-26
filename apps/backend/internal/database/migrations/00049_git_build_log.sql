-- +goose Up
ALTER TABLE git_build_jobs ADD COLUMN build_log MEDIUMTEXT NULL;

-- +goose Down
ALTER TABLE git_build_jobs DROP COLUMN build_log;
