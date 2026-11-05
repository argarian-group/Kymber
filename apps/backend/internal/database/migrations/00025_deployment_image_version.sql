-- +goose Up
ALTER TABLE service_deployments
    ADD COLUMN image_version VARCHAR(128) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE service_deployments
    DROP COLUMN image_version;
