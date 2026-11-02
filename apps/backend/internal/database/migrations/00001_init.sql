-- +goose Up
CREATE TABLE users (
    id BIGINT UNSIGNED NOT NULL,
    username VARCHAR(32) NOT NULL,
    email VARCHAR(254) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    is_root BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_username (username),
    UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE workspaces (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(120) NOT NULL,
    created_by BIGINT UNSIGNED NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_workspaces_creator FOREIGN KEY (created_by)
        REFERENCES users (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE projects (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    workspace_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(120) NOT NULL,
    slug VARCHAR(120) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_projects_workspace_slug (workspace_id, slug),
    CONSTRAINT fk_projects_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE workspace_members (
    workspace_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    role VARCHAR(16) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, user_id),
    CONSTRAINT fk_workspace_members_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE RESTRICT,
    CONSTRAINT fk_workspace_members_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE user_sessions (
    token_hash BINARY(32) NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (token_hash),
    KEY idx_user_sessions_user_id (user_id),
    KEY idx_user_sessions_expires_at (expires_at),
    CONSTRAINT fk_user_sessions_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE platform_settings (
    id TINYINT UNSIGNED NOT NULL,
    registration_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (id),
    CONSTRAINT chk_platform_settings_singleton CHECK (id = 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO platform_settings (id, registration_enabled) VALUES (1, FALSE);

CREATE TABLE workspace_invitations (
    token_hash BINARY(32) NOT NULL,
    workspace_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
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

CREATE TABLE servers (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    name VARCHAR(120) NOT NULL,
    connection_type VARCHAR(8) NOT NULL DEFAULT 'local',
    host VARCHAR(255) NULL,
    ssh_port SMALLINT UNSIGNED NOT NULL DEFAULT 22,
    ssh_user VARCHAR(64) NULL,
    encrypted_ssh_key MEDIUMBLOB NULL,
    ssh_host_key VARBINARY(4096) NULL,
    token_hash BINARY(32) NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    last_seen DATETIME(6) NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_servers_token_hash (token_hash),
    CONSTRAINT fk_servers_creator FOREIGN KEY (created_by)
        REFERENCES users (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE database_services (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    workspace_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    project_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    server_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NULL,
    engine VARCHAR(16) NOT NULL,
    name VARCHAR(120) NOT NULL,
    image_version VARCHAR(128) NOT NULL DEFAULT 'latest',
    image VARCHAR(300) NOT NULL DEFAULT '',
    port INT NOT NULL DEFAULT 0,
    data_path VARCHAR(255) NOT NULL DEFAULT '',
    logo_url VARCHAR(512) NOT NULL DEFAULT '',
    category VARCHAR(16) NOT NULL DEFAULT 'application',
    environment_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '',
    database_name VARCHAR(64) NOT NULL DEFAULT '',
    database_user VARCHAR(32) NOT NULL DEFAULT '',
    encrypted_config MEDIUMBLOB NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    container_id VARCHAR(128) NULL,
    last_error TEXT NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_database_services_environment_name (project_id, environment_id, name),
    KEY idx_database_services_workspace (workspace_id),
    CONSTRAINT fk_database_services_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE RESTRICT,
    CONSTRAINT fk_database_services_project FOREIGN KEY (project_id)
        REFERENCES projects (id) ON DELETE RESTRICT,
    CONSTRAINT fk_database_services_server FOREIGN KEY (server_id)
        REFERENCES servers (id) ON DELETE RESTRICT,
    CONSTRAINT fk_database_services_creator FOREIGN KEY (created_by)
        REFERENCES users (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE server_tasks (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    server_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    service_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    action VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'queued',
    attempts SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    claimed_at DATETIME(6) NULL,
    result_message TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME(6) NULL,
    PRIMARY KEY (id),
    KEY idx_server_tasks_queue (server_id, status, created_at),
    CONSTRAINT fk_server_tasks_server FOREIGN KEY (server_id)
        REFERENCES servers (id) ON DELETE RESTRICT,
    CONSTRAINT fk_server_tasks_service FOREIGN KEY (service_id)
        REFERENCES database_services (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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

CREATE TABLE project_tasks (
    id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    project_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    list_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    position INT NOT NULL DEFAULT 0,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    priority VARCHAR(16) NOT NULL,
    due_date DATE NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_project_tasks_project (project_id, created_at),
    KEY idx_project_tasks_list (list_id, position),
    CONSTRAINT fk_project_tasks_project FOREIGN KEY (project_id)
        REFERENCES projects (id) ON DELETE CASCADE,
    CONSTRAINT fk_project_tasks_list FOREIGN KEY (list_id)
        REFERENCES task_lists (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE task_assignees (
    task_id CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (task_id, user_id),
    CONSTRAINT fk_task_assignees_task FOREIGN KEY (task_id)
        REFERENCES project_tasks (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_assignees_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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

-- +goose Down
DROP TABLE task_comments;
DROP TABLE task_checklist_items;
DROP TABLE task_checklists;
DROP TABLE task_labels;
DROP TABLE project_labels;
DROP TABLE task_assignees;
DROP TABLE project_tasks;
DROP TABLE task_lists;
DROP TABLE project_logos;
DROP TABLE server_tasks;
DROP TABLE database_services;
DROP TABLE project_environments;
DROP TABLE servers;
DROP TABLE workspace_invitations;
DROP TABLE user_sessions;
DROP TABLE workspace_members;
DROP TABLE platform_settings;
DROP TABLE projects;
DROP TABLE workspaces;
DROP TABLE users;
