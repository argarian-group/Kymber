-- +goose Up
CREATE TABLE task_lists (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    project_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(80) NOT NULL,
    is_done BOOLEAN NOT NULL DEFAULT FALSE,
    position INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_task_lists_project (project_id, position),
    CONSTRAINT fk_task_lists_project FOREIGN KEY (project_id)
        REFERENCES projects (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO task_lists (id, project_id, name, is_done, position)
SELECT LEFT(MD5(CONCAT(p.id, s.status)), 6), p.id, s.name, s.is_done, s.position
FROM projects p
CROSS JOIN (
    SELECT 'todo' AS status, 'To do' AS name, FALSE AS is_done, 1 AS position
    UNION ALL SELECT 'in_progress', 'In progress', FALSE, 2
    UNION ALL SELECT 'in_review', 'In review', FALSE, 3
    UNION ALL SELECT 'done', 'Done', TRUE, 4
) s;

ALTER TABLE project_tasks
    ADD COLUMN list_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NULL AFTER project_id,
    ADD COLUMN position INT NOT NULL DEFAULT 0 AFTER list_id,
    ADD COLUMN description TEXT NOT NULL DEFAULT '' AFTER title;

UPDATE project_tasks
SET list_id = LEFT(MD5(CONCAT(project_id, status)), 6), updated_at = updated_at;

UPDATE project_tasks t
JOIN (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY list_id ORDER BY created_at, id) AS row_position
    FROM project_tasks
) ranked ON ranked.id = t.id
SET t.position = ranked.row_position, t.updated_at = t.updated_at;

CREATE TABLE task_assignees (
    task_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (task_id, user_id),
    CONSTRAINT fk_task_assignees_task FOREIGN KEY (task_id)
        REFERENCES project_tasks (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_assignees_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO task_assignees (task_id, user_id)
SELECT id, assignee_id FROM project_tasks WHERE assignee_id IS NOT NULL;

ALTER TABLE project_tasks
    DROP FOREIGN KEY fk_project_tasks_assignee,
    DROP COLUMN assignee_id,
    DROP COLUMN status,
    MODIFY COLUMN list_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    ADD KEY idx_project_tasks_list (list_id, position),
    ADD CONSTRAINT fk_project_tasks_list FOREIGN KEY (list_id)
        REFERENCES task_lists (id) ON DELETE CASCADE;

CREATE TABLE project_labels (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    project_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(40) NOT NULL,
    color VARCHAR(16) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_project_labels_project (project_id),
    CONSTRAINT fk_project_labels_project FOREIGN KEY (project_id)
        REFERENCES projects (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE task_labels (
    task_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    label_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    PRIMARY KEY (task_id, label_id),
    CONSTRAINT fk_task_labels_task FOREIGN KEY (task_id)
        REFERENCES project_tasks (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_labels_label FOREIGN KEY (label_id)
        REFERENCES project_labels (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE task_checklists (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    task_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    title VARCHAR(120) NOT NULL,
    position INT NOT NULL,
    PRIMARY KEY (id),
    KEY idx_task_checklists_task (task_id, position),
    CONSTRAINT fk_task_checklists_task FOREIGN KEY (task_id)
        REFERENCES project_tasks (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE task_checklist_items (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    checklist_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    title VARCHAR(300) NOT NULL,
    done BOOLEAN NOT NULL DEFAULT FALSE,
    position INT NOT NULL,
    PRIMARY KEY (id),
    KEY idx_task_checklist_items_checklist (checklist_id, position),
    CONSTRAINT fk_task_checklist_items_checklist FOREIGN KEY (checklist_id)
        REFERENCES task_checklists (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE task_comments (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    task_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    author_id BIGINT UNSIGNED NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_task_comments_task (task_id, created_at),
    CONSTRAINT fk_task_comments_task FOREIGN KEY (task_id)
        REFERENCES project_tasks (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_comments_author FOREIGN KEY (author_id)
        REFERENCES users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE task_comments;
DROP TABLE task_checklist_items;
DROP TABLE task_checklists;
DROP TABLE task_labels;
DROP TABLE project_labels;

ALTER TABLE project_tasks
    ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'todo' AFTER title,
    ADD COLUMN assignee_id BIGINT UNSIGNED NULL AFTER priority;

UPDATE project_tasks t
JOIN task_lists l ON l.id = t.list_id
SET t.status = IF(l.is_done, 'done', 'todo'), t.updated_at = t.updated_at;

UPDATE project_tasks t
JOIN (SELECT task_id, MIN(user_id) AS user_id FROM task_assignees GROUP BY task_id) a ON a.task_id = t.id
SET t.assignee_id = a.user_id, t.updated_at = t.updated_at;

ALTER TABLE project_tasks
    DROP FOREIGN KEY fk_project_tasks_list,
    DROP KEY idx_project_tasks_list,
    DROP COLUMN list_id,
    DROP COLUMN position,
    DROP COLUMN description,
    ALTER COLUMN status DROP DEFAULT,
    ADD CONSTRAINT fk_project_tasks_assignee FOREIGN KEY (assignee_id)
        REFERENCES users (id) ON DELETE SET NULL;

DROP TABLE task_assignees;
DROP TABLE task_lists;
