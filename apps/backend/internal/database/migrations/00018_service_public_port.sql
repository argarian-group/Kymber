-- +goose Up
ALTER TABLE database_services ADD COLUMN public_port SMALLINT UNSIGNED NULL DEFAULT NULL AFTER domain;

-- +goose Down
ALTER TABLE database_services DROP COLUMN public_port;
