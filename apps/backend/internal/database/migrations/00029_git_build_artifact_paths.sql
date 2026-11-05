-- +goose Up
ALTER TABLE git_build_jobs
    ADD COLUMN artifact_path VARCHAR(255) NULL,
    ADD COLUMN artifact_size BIGINT UNSIGNED NULL,
    ADD COLUMN artifact_expires_at DATETIME(6) NULL,
    ADD KEY idx_git_build_jobs_artifact_expiry (artifact_expires_at);

-- +goose Down
ALTER TABLE git_build_jobs
    DROP KEY idx_git_build_jobs_artifact_expiry,
    DROP COLUMN artifact_expires_at,
    DROP COLUMN artifact_size,
    DROP COLUMN artifact_path;
