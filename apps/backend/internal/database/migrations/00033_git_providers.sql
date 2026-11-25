-- +goose Up
CREATE TABLE git_providers (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    workspace_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    kind VARCHAR(16) NOT NULL,
    name VARCHAR(120) NOT NULL,
    base_url VARCHAR(255) NOT NULL DEFAULT '',
    app_slug VARCHAR(120) NOT NULL DEFAULT '',
    public_key TEXT NULL,
    encrypted_credentials BLOB NULL,
    setup_state CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uq_git_providers_setup_state (setup_state),
    KEY idx_git_providers_workspace (workspace_id, created_at),
    CONSTRAINT fk_git_providers_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE git_provider_installations (
    provider_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    installation_id BIGINT UNSIGNED NOT NULL,
    account_login VARCHAR(120) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (provider_id, installation_id),
    KEY idx_git_provider_installations_id (installation_id),
    CONSTRAINT fk_git_provider_installations_provider FOREIGN KEY (provider_id)
        REFERENCES git_providers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE service_git_sources
    ADD COLUMN provider_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NULL AFTER service_id,
    ADD COLUMN installation_id BIGINT UNSIGNED NULL AFTER provider_id,
    ADD CONSTRAINT fk_service_git_sources_provider FOREIGN KEY (provider_id)
        REFERENCES git_providers (id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE service_git_sources
    DROP FOREIGN KEY fk_service_git_sources_provider,
    DROP COLUMN installation_id,
    DROP COLUMN provider_id;
DROP TABLE git_provider_installations;
DROP TABLE git_providers;
