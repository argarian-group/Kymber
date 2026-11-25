-- +goose Up
ALTER TABLE projects
    ADD COLUMN status_page_token CHAR(22) CHARACTER SET ascii COLLATE ascii_bin NULL,
    ADD UNIQUE KEY uq_projects_status_page_token (status_page_token);

-- +goose Down
ALTER TABLE projects
    DROP KEY uq_projects_status_page_token,
    DROP COLUMN status_page_token;
