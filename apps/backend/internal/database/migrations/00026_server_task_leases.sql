-- +goose Up
ALTER TABLE server_tasks
    ADD COLUMN lease_expires_at DATETIME(6) NULL,
    ADD KEY idx_server_tasks_lease (server_id, status, lease_expires_at);

-- +goose Down
ALTER TABLE server_tasks
    DROP KEY idx_server_tasks_lease,
    DROP COLUMN lease_expires_at;
