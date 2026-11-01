-- +goose Up
ALTER TABLE database_services
    ADD COLUMN image VARCHAR(300) NOT NULL DEFAULT '' AFTER image_version,
    ADD COLUMN port INT NOT NULL DEFAULT 0 AFTER image,
    ADD COLUMN data_path VARCHAR(255) NOT NULL DEFAULT '' AFTER port;

-- +goose Down
ALTER TABLE database_services
    DROP COLUMN data_path,
    DROP COLUMN port,
    DROP COLUMN image;
