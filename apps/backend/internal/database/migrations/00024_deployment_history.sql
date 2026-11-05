-- +goose Up
CREATE TABLE service_deployments (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    service_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    action VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'deploying',
    output MEDIUMTEXT NOT NULL DEFAULT '',
    last_error TEXT NULL,
    started_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    finished_at DATETIME(6) NULL,
    PRIMARY KEY (id),
    KEY idx_service_deployments_service_started (service_id, started_at),
    CONSTRAINT fk_service_deployments_service FOREIGN KEY (service_id)
        REFERENCES database_services (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE service_deployments;
