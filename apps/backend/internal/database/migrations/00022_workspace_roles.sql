-- +goose Up
CREATE TABLE workspace_roles (
    workspace_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(32) NOT NULL,
    permissions VARCHAR(512) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, name),
    CONSTRAINT fk_workspace_roles_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE workspace_members MODIFY role VARCHAR(32) NOT NULL;
ALTER TABLE workspace_invitations MODIFY role VARCHAR(32) NOT NULL;

-- +goose Down
DROP TABLE workspace_roles;
