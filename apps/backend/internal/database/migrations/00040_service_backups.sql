-- +goose Up
CREATE TABLE service_backups (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    service_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    server_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'running',
    scheduled BOOLEAN NOT NULL DEFAULT FALSE,
    size_bytes BIGINT UNSIGNED NOT NULL DEFAULT 0,
    error TEXT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    finished_at DATETIME(6) NULL,
    PRIMARY KEY (id),
    KEY idx_service_backups_service (service_id, created_at),
    CONSTRAINT fk_service_backups_service FOREIGN KEY (service_id) REFERENCES database_services (id) ON DELETE CASCADE
);

ALTER TABLE database_services
    ADD COLUMN backup_schedule VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN backup_keep INT UNSIGNED NOT NULL DEFAULT 7,
    ADD COLUMN backup_next_run_at DATETIME(6) NULL;

-- +goose Down
ALTER TABLE database_services
    DROP COLUMN backup_next_run_at,
    DROP COLUMN backup_keep,
    DROP COLUMN backup_schedule;
DROP TABLE service_backups;
