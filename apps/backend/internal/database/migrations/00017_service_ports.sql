-- +goose Up
CREATE TABLE service_ports (
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
DROP TABLE service_ports;
