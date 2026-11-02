-- +goose Up
ALTER TABLE database_services
    ADD COLUMN domain VARCHAR(253) NULL DEFAULT NULL AFTER port,
    ADD UNIQUE KEY uq_database_services_domain (domain);
ALTER TABLE platform_settings ADD COLUMN panel_domain VARCHAR(253) NULL DEFAULT NULL AFTER registration_enabled;

-- +goose Down
ALTER TABLE platform_settings DROP COLUMN panel_domain;
ALTER TABLE database_services
    DROP INDEX uq_database_services_domain,
    DROP COLUMN domain;
