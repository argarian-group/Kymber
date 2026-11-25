-- +goose Up
CREATE TABLE task_time_entries (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    task_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP NULL DEFAULT NULL,
    running_user_id BIGINT UNSIGNED AS (IF(ended_at IS NULL, user_id, NULL)) PERSISTENT,
    PRIMARY KEY (id),
    UNIQUE KEY uq_task_time_entries_running (running_user_id),
    KEY idx_task_time_entries_task (task_id, started_at),
    CONSTRAINT fk_task_time_entries_task FOREIGN KEY (task_id) REFERENCES project_tasks (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_time_entries_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE task_time_entries;
