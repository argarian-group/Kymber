-- +goose Up
ALTER TABLE servers ADD COLUMN IF NOT EXISTS monitoring_installed_at DATETIME(6) NULL DEFAULT NULL;

-- +goose Down
ALTER TABLE servers DROP COLUMN IF EXISTS monitoring_installed_at;
