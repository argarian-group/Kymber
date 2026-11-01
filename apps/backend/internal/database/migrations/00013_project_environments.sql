-- +goose Up
CREATE TABLE project_environments (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    project_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_project_environments_name (project_id, name),
    CONSTRAINT fk_project_environments_project FOREIGN KEY (project_id)
        REFERENCES projects (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Empty means outside any environment; NOT NULL lets the unique key cover those apps too.
ALTER TABLE database_services
    ADD COLUMN environment_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '' AFTER category,
    ADD UNIQUE KEY uq_database_services_environment_name (project_id, environment_id, name),
    DROP KEY uq_database_services_project_name;

-- +goose Down
ALTER TABLE database_services
    ADD UNIQUE KEY uq_database_services_project_name (project_id, name),
    DROP KEY uq_database_services_environment_name,
    DROP COLUMN environment_id;

DROP TABLE project_environments;
