-- +goose Up
ALTER TABLE database_services
    MODIFY COLUMN domain VARCHAR(2048) CHARACTER SET ascii COLLATE ascii_general_ci NULL DEFAULT NULL,
    MODIFY COLUMN applied_domain VARCHAR(2048) CHARACTER SET ascii COLLATE ascii_general_ci NULL DEFAULT NULL;

-- +goose Down
ALTER TABLE database_services
    MODIFY COLUMN domain VARCHAR(253) NULL DEFAULT NULL,
    MODIFY COLUMN applied_domain VARCHAR(253) NULL DEFAULT NULL;
