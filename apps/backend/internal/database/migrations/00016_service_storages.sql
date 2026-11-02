-- +goose Up
CREATE TABLE service_storages (
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

-- +goose Down
DROP TABLE service_storages;
