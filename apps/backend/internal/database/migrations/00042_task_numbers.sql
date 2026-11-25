-- +goose Up
ALTER TABLE projects
    ADD COLUMN task_prefix VARCHAR(10) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT 'TASK',
    ADD COLUMN task_counter INT UNSIGNED NOT NULL DEFAULT 0;

UPDATE projects
SET task_prefix = COALESCE(NULLIF(UPPER(LEFT(REGEXP_REPLACE(name, '[^A-Za-z0-9]', ''), 4)), ''), 'TASK');

UPDATE projects SET task_prefix = CONCAT('P', task_prefix) WHERE task_prefix REGEXP '^[0-9]';

ALTER TABLE project_tasks ADD COLUMN number INT UNSIGNED NULL;

UPDATE project_tasks t
JOIN (SELECT id, ROW_NUMBER() OVER (PARTITION BY project_id ORDER BY created_at, id) AS n FROM project_tasks) numbered ON numbered.id = t.id
SET t.number = numbered.n, t.updated_at = t.updated_at;

UPDATE projects p SET task_counter = (SELECT COALESCE(MAX(number), 0) FROM project_tasks t WHERE t.project_id = p.id);

ALTER TABLE project_tasks
    MODIFY number INT UNSIGNED NOT NULL,
    ADD UNIQUE KEY uq_project_tasks_number (project_id, number);

-- +goose Down
ALTER TABLE project_tasks DROP KEY uq_project_tasks_number, DROP COLUMN number;
ALTER TABLE projects DROP COLUMN task_counter, DROP COLUMN task_prefix;
