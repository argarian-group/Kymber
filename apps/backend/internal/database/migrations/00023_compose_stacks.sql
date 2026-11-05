-- +goose Up
ALTER TABLE database_services
    ADD COLUMN IF NOT EXISTS compose_file MEDIUMBLOB NULL,
    ADD COLUMN IF NOT EXISTS compose_project VARCHAR(63) NULL;

-- +goose Down
ALTER TABLE database_services
    DROP COLUMN compose_project,
    DROP COLUMN compose_file;
