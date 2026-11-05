-- +goose Up
ALTER TABLE git_build_jobs
    ADD COLUMN artifact_path VARCHAR(255) NULL;

-- +goose Down
ALTER TABLE git_build_jobs
    DROP COLUMN artifact_path;
