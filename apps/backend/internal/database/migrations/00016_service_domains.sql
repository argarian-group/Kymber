-- +goose Up
ALTER TABLE database_services
    ADD COLUMN domain VARCHAR(253) NULL DEFAULT NULL AFTER port,
    ADD UNIQUE KEY uq_database_services_domain (domain);

-- +goose Down
ALTER TABLE database_services
    DROP INDEX uq_database_services_domain,
    DROP COLUMN domain;
