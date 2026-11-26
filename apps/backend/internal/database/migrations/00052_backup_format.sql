-- +goose Up
ALTER TABLE service_backups ADD COLUMN format VARCHAR(8) CHARACTER SET ascii NOT NULL DEFAULT 'tar';
UPDATE service_backups SET format = 'cp';

-- +goose Down
ALTER TABLE service_backups DROP COLUMN format;
