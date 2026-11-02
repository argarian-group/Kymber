-- +goose Up
ALTER TABLE platform_settings ADD COLUMN panel_domain VARCHAR(253) NULL DEFAULT NULL AFTER registration_enabled;

-- +goose Down
ALTER TABLE platform_settings DROP COLUMN panel_domain;
