-- +goose Up
CREATE TABLE project_logos (
    project_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    content_type VARCHAR(32) NOT NULL,
    version CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    data MEDIUMBLOB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id),
    CONSTRAINT fk_project_logos_project FOREIGN KEY (project_id)
        REFERENCES projects (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE project_logos;
