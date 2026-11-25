-- +goose Up
ALTER TABLE service_git_sources
    ADD COLUMN previews_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN preview_domain VARCHAR(253) NOT NULL DEFAULT '';

ALTER TABLE database_services
    ADD COLUMN preview_of CHAR(6) CHARACTER SET ascii COLLATE ascii_bin NULL,
    ADD COLUMN pr_number INT UNSIGNED NULL,
    ADD UNIQUE KEY uq_database_services_preview (preview_of, pr_number),
    ADD CONSTRAINT fk_database_services_preview FOREIGN KEY (preview_of)
        REFERENCES database_services (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE database_services
    DROP FOREIGN KEY fk_database_services_preview,
    DROP KEY uq_database_services_preview,
    DROP COLUMN pr_number,
    DROP COLUMN preview_of;
ALTER TABLE service_git_sources
    DROP COLUMN preview_domain,
    DROP COLUMN previews_enabled;
