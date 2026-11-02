-- +goose Up
ALTER TABLE servers
    MODIFY token_hash BINARY(32) NULL,
    ADD COLUMN IF NOT EXISTS connection_type VARCHAR(8) NOT NULL DEFAULT 'local' AFTER name,
    ADD COLUMN IF NOT EXISTS host VARCHAR(255) NULL AFTER connection_type,
    ADD COLUMN IF NOT EXISTS ssh_port SMALLINT UNSIGNED NOT NULL DEFAULT 22 AFTER host,
    ADD COLUMN IF NOT EXISTS ssh_user VARCHAR(64) NULL AFTER ssh_port,
    ADD COLUMN IF NOT EXISTS encrypted_ssh_key MEDIUMBLOB NULL AFTER ssh_user,
    ADD COLUMN IF NOT EXISTS ssh_host_key VARBINARY(4096) NULL AFTER encrypted_ssh_key;

-- +goose Down
ALTER TABLE servers
    DROP COLUMN ssh_host_key,
    DROP COLUMN encrypted_ssh_key,
    DROP COLUMN ssh_user,
    DROP COLUMN ssh_port,
    DROP COLUMN host,
    DROP COLUMN connection_type,
    MODIFY token_hash BINARY(32) NOT NULL;
