-- +goose Up
ALTER TABLE database_services ADD COLUMN category VARCHAR(16) NOT NULL DEFAULT 'application' AFTER logo_url;

UPDATE database_services SET category = 'database' WHERE engine IN ('mysql', 'redis');

-- +goose Down
ALTER TABLE database_services DROP COLUMN category;
