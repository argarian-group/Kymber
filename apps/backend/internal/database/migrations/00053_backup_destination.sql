-- +goose Up
ALTER TABLE platform_settings
    ADD COLUMN s3_endpoint VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN s3_region VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN s3_bucket VARCHAR(63) NOT NULL DEFAULT '',
    ADD COLUMN s3_prefix VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN s3_access_key VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN encrypted_s3_secret VARBINARY(1024) NULL;
ALTER TABLE service_backups ADD COLUMN remote_key VARCHAR(512) NULL;

-- +goose Down
ALTER TABLE service_backups DROP COLUMN remote_key;
ALTER TABLE platform_settings
    DROP COLUMN s3_endpoint,
    DROP COLUMN s3_region,
    DROP COLUMN s3_bucket,
    DROP COLUMN s3_prefix,
    DROP COLUMN s3_access_key,
    DROP COLUMN encrypted_s3_secret;
