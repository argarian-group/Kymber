package project

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
)

type projectEnvironment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func environmentName(raw string) string {
	name := strings.TrimSpace(raw)
	if utf8.RuneCountInString(name) > 64 || strings.ContainsFunc(name, unicode.IsControl) {
		return ""
	}
	return name
}

func (h *Handler) listEnvironments(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, name FROM project_environments WHERE project_id = ? ORDER BY created_at, id", projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list environments")
		return
	}
	defer rows.Close()
	environments := make([]projectEnvironment, 0)
	for rows.Next() {
		var environment projectEnvironment
		if err := rows.Scan(&environment.ID, &environment.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read environments")
			return
		}
		environments = append(environments, environment)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read environments")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]projectEnvironment{"environments": environments})
}

func (h *Handler) createEnvironment(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, permissions.ProjectsManage)
	if !ok {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	name := environmentName(input.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "environment name must be 1 to 64 characters")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate environment id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO project_environments (id, project_id, name) VALUES (?, ?, ?)", id, projectID, name); err != nil {
		writeError(w, http.StatusConflict, "could not create environment; its name may already be used")
		return
	}
	writeJSON(w, http.StatusCreated, projectEnvironment{ID: id, Name: name})
}

func (h *Handler) cloneEnvironment(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, permissions.ProjectsManage)
	if !ok {
		return
	}
	sourceID, ok := pathID(w, r, "environmentID")
	if !ok {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	name := environmentName(input.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "environment name must be 1 to 64 characters")
		return
	}
	user, _ := auth.UserFromContext(r)
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not clone environment")
		return
	}
	defer tx.Rollback()
	var found string
	if err := tx.QueryRowContext(r.Context(), "SELECT id FROM project_environments WHERE id = ? AND project_id = ?", sourceID, projectID).Scan(&found); err != nil {
		writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate environment id")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO project_environments (id, project_id, name) VALUES (?, ?, ?)", id, projectID, name); err != nil {
		writeError(w, http.StatusConflict, "could not create environment; its name may already be used")
		return
	}
	if err := cloneEnvironmentApps(r.Context(), tx, projectID, sourceID, id, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not clone the environment's apps")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not clone environment")
		return
	}
	writeJSON(w, http.StatusCreated, projectEnvironment{ID: id, Name: name})
}

// Clones start as undeployed drafts without domains, published ports or auto-deploy, so nothing collides with the source apps.
func cloneEnvironmentApps(ctx context.Context, tx *sql.Tx, projectID, sourceID, targetID string, userID uint64) error {
	rows, err := tx.QueryContext(ctx, "SELECT id FROM database_services WHERE project_id = ? AND environment_id = ? AND status <> 'deleted' AND preview_of IS NULL ORDER BY created_at", projectID, sourceID)
	if err != nil {
		return err
	}
	var serviceIDs []string
	for rows.Next() {
		var serviceID string
		if err := rows.Scan(&serviceID); err != nil {
			rows.Close()
			return err
		}
		serviceIDs = append(serviceIDs, serviceID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, sourceService := range serviceIDs {
		cloneID, err := database.NewShortID()
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO database_services (id, workspace_id, project_id, engine, name, image_version, image, port, data_path, logo_url, category, environment_id, database_name, database_user, encrypted_config, compose_file, compose_project, created_by, cpu_limit, memory_limit_mb, health_cmd, backup_schedule, backup_keep, backup_next_run_at)
			SELECT ?, workspace_id, project_id, engine, name, IF(image = 'docker.io/kymber/build-artifacts', 'draft', image_version), image, port, data_path, logo_url, category, ?, database_name, database_user, encrypted_config, compose_file, IF(compose_file IS NULL, NULL, CONCAT(name, '-', ?)), ?, cpu_limit, memory_limit_mb, health_cmd, backup_schedule, backup_keep, backup_next_run_at
			FROM database_services WHERE id = ?`, cloneID, targetID, cloneID, userID, sourceService); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO service_git_sources (service_id, provider_id, installation_id, repository_url, git_ref, base_directory, dockerfile_path, build_pack, install_command, build_command, start_command, publish_directory, encrypted_build_args, auto_deploy, previews_enabled, preview_domain)
			SELECT ?, provider_id, installation_id, repository_url, git_ref, base_directory, dockerfile_path, build_pack, install_command, build_command, start_command, publish_directory, encrypted_build_args, FALSE, FALSE, preview_domain FROM service_git_sources WHERE service_id = ?`, cloneID, sourceService); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO service_storages (id, service_id, kind, source, mount_path, encrypted_content)
			SELECT LOWER(HEX(RANDOM_BYTES(3))), ?, kind, source, mount_path, encrypted_content FROM service_storages WHERE service_id = ?`, cloneID, sourceService); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO service_jobs (id, service_id, schedule, command, enabled, next_run_at)
			SELECT LOWER(HEX(RANDOM_BYTES(3))), ?, schedule, command, enabled, next_run_at FROM service_jobs WHERE service_id = ?`, cloneID, sourceService); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) deleteEnvironment(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, permissions.ProjectsManage)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "environmentID")
	if !ok {
		return
	}
	result, err := h.db.ExecContext(r.Context(), "DELETE FROM project_environments WHERE id = ? AND project_id = ? AND NOT EXISTS (SELECT 1 FROM database_services d WHERE d.environment_id = ? AND d.status <> 'deleted')", id, projectID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete environment")
		return
	}
	if deleted, _ := result.RowsAffected(); deleted == 0 {
		writeError(w, http.StatusConflict, "environment not found or still has apps")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
