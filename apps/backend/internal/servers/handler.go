package servers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/permissions"
)

type Handler struct{ db *sql.DB }

func New(db *sql.DB) *Handler { return &Handler{db: db} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	for _, route := range []struct {
		pattern string
		handler http.HandlerFunc
	}{
		{"GET /api/servers", h.listServers},
		{"POST /api/servers", h.createServer},
		{"POST /api/servers/{serverID}/ssh", h.setupLocalSSH},
		{"PUT /api/servers/{serverID}", h.renameServer},
		{"POST /api/servers/{serverID}/token", h.rotateServerToken},
		{"DELETE /api/servers/{serverID}", h.deleteServer},
		{"GET /api/workspaces/{workspaceID}/servers", h.listDeployTargets},
	} {
		mux.Handle(route.pattern, auth.Authenticate(h.db, route.handler))
	}
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

func pathID(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id := r.PathValue(name)
	if !idPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return "", false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain a single JSON value")
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID string, permission string) bool {
	if user.IsRoot {
		var exists bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = ?)", workspaceID).Scan(&exists); err != nil {
			writeError(w, http.StatusInternalServerError, "could not verify workspace")
			return false
		}
		if !exists {
			writeError(w, http.StatusNotFound, "workspace not found")
			return false
		}
		return true
	}
	_, granted, err := permissions.Lookup(r.Context(), h.db, workspaceID, user.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, "workspace not found")
		return false
	}
	if permission != "" && !granted.Has(permission) {
		writeError(w, http.StatusForbidden, "you do not have permission for this action")
		return false
	}
	return true
}
