-- +goose Up
ALTER TABLE database_services
    ADD COLUMN cpu_limit DECIMAL(5, 2) NOT NULL DEFAULT 0,
    ADD COLUMN memory_limit_mb INT UNSIGNED NOT NULL DEFAULT 0,
    ADD COLUMN health_cmd VARCHAR(512) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE database_services
    DROP COLUMN health_cmd,
    DROP COLUMN memory_limit_mb,
    DROP COLUMN cpu_limit;
