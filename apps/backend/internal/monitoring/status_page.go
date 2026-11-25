package monitoring

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"

	"kymber/backend/internal/logger"
)

var statusTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{22}$`)

type publicApp struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// publicStatus maps an app's state to what a status page shows: operational, degraded, down or deploying.
func publicStatus(status string, alerting bool) string {
	switch {
	case status == "failed":
		return "down"
	case status == "queued" || status == "deploying":
		return "deploying"
	case alerting:
		return "degraded"
	default:
		return "operational"
	}
}

func (h *Handler) statusPage(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if !statusTokenPattern.MatchString(token) {
		writeError(w, http.StatusNotFound, "status page not found")
		return
	}
	var projectID, name string
	err := h.db.QueryRowContext(r.Context(), "SELECT id, name FROM projects WHERE status_page_token = ?", token).Scan(&projectID, &name)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "status page not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read status page")
		return
	}
	alerting := map[string]bool{}
	if firing, err := firingAlerts(r.Context()); err == nil {
		for _, alert := range firing {
			if alert.Name == "ServerDown" {
				alerting["server:"+alert.ServerID] = true
			} else if alert.ServiceID != "" {
				alerting[alert.ServiceID] = true
			}
		}
	} else {
		logger.Warn("status page could not read alerts: %v", err)
	}
	rows, err := h.db.QueryContext(r.Context(), `SELECT id, name, status, COALESCE(server_id, '') FROM database_services
		WHERE project_id = ? AND preview_of IS NULL AND status IN ('running', 'failed', 'queued', 'deploying') ORDER BY category, name`, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read status page")
		return
	}
	defer rows.Close()
	apps := make([]publicApp, 0)
	for rows.Next() {
		var id, appName, status, serverID string
		if err := rows.Scan(&id, &appName, &status, &serverID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read status page")
			return
		}
		if alerting["server:"+serverID] {
			status = "failed"
		}
		apps = append(apps, publicApp{Name: appName, Status: publicStatus(status, alerting[id])})
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read status page")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=30")
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "apps": apps})
}
