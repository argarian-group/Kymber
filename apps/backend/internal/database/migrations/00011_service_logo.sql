-- +goose Up
ALTER TABLE database_services ADD COLUMN logo_url VARCHAR(512) NOT NULL DEFAULT '' AFTER data_path;

-- +goose Down
ALTER TABLE database_services DROP COLUMN logo_url;
