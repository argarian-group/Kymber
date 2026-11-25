-- +goose Up
CREATE TABLE service_jobs (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    service_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    schedule VARCHAR(100) NOT NULL,
    command VARCHAR(2000) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    next_run_at DATETIME(6) NOT NULL,
    running_since DATETIME(6) NULL,
    last_run_at DATETIME(6) NULL,
    last_exit_code INT NULL,
    last_output TEXT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_service_jobs_due (enabled, next_run_at),
    CONSTRAINT fk_service_jobs_service FOREIGN KEY (service_id) REFERENCES database_services (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE service_jobs;
