-- +goose Up
ALTER TABLE workspaces ADD COLUMN created_by BIGINT UNSIGNED NULL AFTER name;
UPDATE workspaces w
SET created_by = (
    SELECT wm.user_id
    FROM workspace_members wm
    WHERE wm.workspace_id = w.id
    ORDER BY (wm.role = 'owner') DESC, wm.created_at, wm.user_id
    LIMIT 1
);
ALTER TABLE workspaces ADD CONSTRAINT fk_workspaces_creator FOREIGN KEY (created_by)
    REFERENCES users (id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE workspaces DROP FOREIGN KEY fk_workspaces_creator;
ALTER TABLE workspaces DROP COLUMN created_by;
