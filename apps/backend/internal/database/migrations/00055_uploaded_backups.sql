-- +goose Up
ALTER TABLE service_backups
    ADD COLUMN uploaded BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN restore_path VARCHAR(255) NULL;
ALTER TABLE platform_backups ADD COLUMN uploaded BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE platform_backups DROP COLUMN uploaded;
ALTER TABLE service_backups
    DROP COLUMN restore_path,
    DROP COLUMN uploaded;
