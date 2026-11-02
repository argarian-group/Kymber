-- +goose Up
ALTER TABLE platform_settings ADD COLUMN IF NOT EXISTS panel_domain VARCHAR(253) NULL DEFAULT NULL AFTER registration_enabled;

ALTER TABLE database_services
    ADD COLUMN IF NOT EXISTS domain VARCHAR(253) NULL DEFAULT NULL AFTER port,
    ADD UNIQUE KEY IF NOT EXISTS uq_database_services_domain (domain);

CREATE TABLE IF NOT EXISTS service_storages (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    service_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    kind VARCHAR(8) NOT NULL,
    source VARCHAR(255) NULL,
    mount_path VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
    encrypted_content MEDIUMBLOB NULL,
    applied BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_service_storages_mount (service_id, mount_path),
    CONSTRAINT fk_service_storages_service FOREIGN KEY (service_id)
        REFERENCES database_services (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS service_ports (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    service_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    host_port SMALLINT UNSIGNED NOT NULL,
    container_port SMALLINT UNSIGNED NOT NULL,
    protocol VARCHAR(3) NOT NULL,
    bind_address VARCHAR(9) NOT NULL,
    applied BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_service_ports_host (service_id, host_port, protocol),
    KEY idx_service_ports_host (host_port, protocol),
    CONSTRAINT fk_service_ports_service FOREIGN KEY (service_id)
        REFERENCES database_services (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
SELECT 1;
