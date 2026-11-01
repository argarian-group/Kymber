-- +goose Up
CREATE TABLE workspace_invitations (
    token_hash BINARY(32) NOT NULL,
    workspace_id BIGINT UNSIGNED NOT NULL,
    email VARCHAR(254) NOT NULL,
    role VARCHAR(16) NOT NULL,
    invited_by BIGINT UNSIGNED NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (token_hash),
    UNIQUE KEY uq_workspace_invitations_email (workspace_id, email),
    KEY idx_workspace_invitations_expires_at (expires_at),
    CONSTRAINT fk_workspace_invitations_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE CASCADE,
    CONSTRAINT fk_workspace_invitations_inviter FOREIGN KEY (invited_by)
        REFERENCES users (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE workspace_invitations;
