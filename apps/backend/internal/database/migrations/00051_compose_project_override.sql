-- +goose Up
UPDATE database_services SET compose_project = NULL;

-- +goose Down
UPDATE database_services SET compose_project = name WHERE compose_file IS NOT NULL;
