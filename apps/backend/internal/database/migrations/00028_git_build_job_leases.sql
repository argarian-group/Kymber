-- +goose Up
ALTER TABLE git_build_jobs
    ADD COLUMN lease_expires_at DATETIME(6) NULL,
    ADD KEY idx_git_build_jobs_lease (status, lease_expires_at);

-- +goose Down
ALTER TABLE git_build_jobs
    DROP KEY idx_git_build_jobs_lease,
    DROP COLUMN lease_expires_at;
