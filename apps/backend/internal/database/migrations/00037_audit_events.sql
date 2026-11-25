-- +goose Up
CREATE TABLE audit_events (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    workspace_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NULL,
    user_id BIGINT UNSIGNED NULL,
    method VARCHAR(8) NOT NULL,
    route VARCHAR(255) NOT NULL,
    path VARCHAR(512) NOT NULL,
    status SMALLINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_audit_events_workspace (workspace_id, id),
    CONSTRAINT fk_audit_events_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces (id) ON DELETE CASCADE,
    CONSTRAINT fk_audit_events_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
);

-- +goose Down
DROP TABLE audit_events;
