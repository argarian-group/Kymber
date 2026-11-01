-- +goose Up
CREATE TABLE servers (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(120) NOT NULL,
    token_hash BINARY(32) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    last_seen DATETIME(6) NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_servers_token_hash (token_hash),
    CONSTRAINT fk_servers_creator FOREIGN KEY (created_by)
        REFERENCES users (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE database_services (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    workspace_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    project_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    server_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NULL,
    engine VARCHAR(16) NOT NULL,
    name VARCHAR(120) NOT NULL,
    database_name VARCHAR(64) NOT NULL DEFAULT '',
    database_user VARCHAR(32) NOT NULL DEFAULT '',
    encrypted_config MEDIUMBLOB NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    container_id VARCHAR(128) NULL,
    last_error TEXT NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_database_services_project_name (project_id, name),
    KEY idx_database_services_workspace (workspace_id),
    CONSTRAINT fk_database_services_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE RESTRICT,
    CONSTRAINT fk_database_services_project FOREIGN KEY (project_id)
        REFERENCES projects (id) ON DELETE RESTRICT,
    CONSTRAINT fk_database_services_server FOREIGN KEY (server_id)
        REFERENCES servers (id) ON DELETE RESTRICT,
    CONSTRAINT fk_database_services_creator FOREIGN KEY (created_by)
        REFERENCES users (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE server_tasks (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    server_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    service_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    action VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'queued',
    attempts SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    claimed_at DATETIME(6) NULL,
    result_message TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME(6) NULL,
    PRIMARY KEY (id),
    KEY idx_server_tasks_queue (server_id, status, created_at),
    CONSTRAINT fk_server_tasks_server FOREIGN KEY (server_id)
        REFERENCES servers (id) ON DELETE RESTRICT,
    CONSTRAINT fk_server_tasks_service FOREIGN KEY (service_id)
        REFERENCES database_services (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE server_tasks;
DROP TABLE database_services;
DROP TABLE servers;
