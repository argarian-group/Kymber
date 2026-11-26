-- +goose Up
CREATE TABLE platform_backups (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    server_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'running',
    scheduled BOOLEAN NOT NULL DEFAULT FALSE,
    size_bytes BIGINT UNSIGNED NOT NULL DEFAULT 0,
    remote_key VARCHAR(512) NULL,
    error TEXT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    finished_at DATETIME(6) NULL,
    PRIMARY KEY (id),
    KEY idx_platform_backups_created (created_at)
);

ALTER TABLE platform_settings
    ADD COLUMN backup_schedule VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN backup_keep INT UNSIGNED NOT NULL DEFAULT 7,
    ADD COLUMN backup_next_run_at DATETIME(6) NULL;

-- +goose Down
ALTER TABLE platform_settings
    DROP COLUMN backup_next_run_at,
    DROP COLUMN backup_keep,
    DROP COLUMN backup_schedule;
DROP TABLE platform_backups;
