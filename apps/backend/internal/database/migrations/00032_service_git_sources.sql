-- +goose Up
CREATE TABLE service_git_sources (
    service_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    repository_url VARCHAR(512) NOT NULL,
    git_ref VARCHAR(128) NOT NULL,
    base_directory VARCHAR(255) NOT NULL DEFAULT '',
    dockerfile_path VARCHAR(255) NOT NULL DEFAULT 'Dockerfile',
    build_pack VARCHAR(16) NOT NULL DEFAULT 'dockerfile',
    install_command VARCHAR(512) NOT NULL DEFAULT '',
    build_command VARCHAR(512) NOT NULL DEFAULT '',
    start_command VARCHAR(512) NOT NULL DEFAULT '',
    publish_directory VARCHAR(255) NOT NULL DEFAULT '',
    encrypted_build_args BLOB NULL,
    auto_deploy BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (service_id),
    CONSTRAINT fk_service_git_sources_service FOREIGN KEY (service_id)
        REFERENCES database_services (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE git_build_jobs
    ADD COLUMN encrypted_build_config BLOB NULL,
    ADD COLUMN build_trigger VARCHAR(16) NOT NULL DEFAULT 'manual',
    ADD COLUMN deploy_on_success BOOLEAN NOT NULL DEFAULT FALSE;

INSERT IGNORE INTO service_git_sources (service_id, repository_url, git_ref)
SELECT b.service_id, b.repository_url, b.git_ref FROM git_build_jobs b
WHERE b.created_at = (SELECT MAX(l.created_at) FROM git_build_jobs l WHERE l.service_id = b.service_id);

-- +goose Down
ALTER TABLE git_build_jobs
    DROP COLUMN deploy_on_success,
    DROP COLUMN build_trigger,
    DROP COLUMN encrypted_build_config;
DROP TABLE service_git_sources;
