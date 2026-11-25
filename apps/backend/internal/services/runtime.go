package services

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/permissions"
)

type runtimeSettings struct {
	CPULimit      float64 `json:"cpu_limit"`
	MemoryLimitMB int     `json:"memory_limit_mb"`
	HealthCmd     string  `json:"health_cmd"`
}

func validateRuntime(settings runtimeSettings) error {
	if settings.CPULimit < 0 || settings.CPULimit > 256 {
		return errors.New("CPU limit must be between 0 and 256 cores")
	}
	if settings.MemoryLimitMB != 0 && (settings.MemoryLimitMB < 6 || settings.MemoryLimitMB > 1<<20) {
		return errors.New("memory limit must be 0 or between 6 MB and 1 TB")
	}
	if len(settings.HealthCmd) > 512 || strings.IndexFunc(settings.HealthCmd, unicode.IsControl) >= 0 {
		return errors.New("health check must be one line of at most 512 characters")
	}
	return nil
}

func runtimeLimitArgs(cpus float64, memoryMB int) string {
	args := ""
	if cpus > 0 {
		args += " --cpus " + strconv.FormatFloat(cpus, 'f', -1, 64)
	}
	if memoryMB > 0 {
		args += " --memory " + strconv.Itoa(memoryMB) + "m"
	}
	return args
}

func (h *Handler) updateServiceRuntime(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return
	}
	serviceID, ok := pathID(w, r, "serviceID")
	if !ok {
		return
	}
	var input runtimeSettings
	if !decodeJSON(w, r, &input) {
		return
	}
	input.HealthCmd = strings.TrimSpace(input.HealthCmd)
	if err := validateRuntime(input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET cpu_limit = ?, memory_limit_mb = ?, health_cmd = ? WHERE id = ? AND workspace_id = ? AND project_id = ? AND compose_file IS NULL", input.CPULimit, input.MemoryLimitMB, input.HealthCmd, serviceID, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save runtime settings")
		return
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		var exists bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM database_services WHERE id = ? AND workspace_id = ? AND project_id = ? AND compose_file IS NULL)", serviceID, workspaceID, projectID).Scan(&exists); err != nil || !exists {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
	}
	writeJSON(w, http.StatusOK, input)
}
