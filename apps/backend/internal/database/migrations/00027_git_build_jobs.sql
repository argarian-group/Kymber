-- +goose Up
CREATE TABLE git_build_jobs (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    service_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    repository_url VARCHAR(512) NOT NULL,
    git_ref VARCHAR(128) NOT NULL,
    commit_sha CHAR(40) CHARACTER SET ascii COLLATE ascii_bin NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'queued',
    artifact_digest CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    result_message TEXT NULL,
    attempts SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    started_at DATETIME(6) NULL,
    finished_at DATETIME(6) NULL,
    PRIMARY KEY (id),
    KEY idx_git_build_jobs_queue (status, created_at),
    KEY idx_git_build_jobs_service (service_id, created_at),
    CONSTRAINT fk_git_build_jobs_service FOREIGN KEY (service_id)
        REFERENCES database_services (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE git_build_jobs;
