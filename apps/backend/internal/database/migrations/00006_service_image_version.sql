-- +goose Up
ALTER TABLE database_services
    ADD COLUMN image_version VARCHAR(128) NOT NULL DEFAULT 'latest' AFTER name;

UPDATE database_services SET image_version = 'latest' WHERE image_version = '';

-- +goose Down
ALTER TABLE database_services DROP COLUMN image_version;
