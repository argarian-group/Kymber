-- +goose Up
ALTER TABLE database_services
    ADD COLUMN applied_domain VARCHAR(253) NULL DEFAULT NULL;

UPDATE database_services SET applied_domain = domain;

-- +goose Down
ALTER TABLE database_services DROP COLUMN applied_domain;
