package auth

import "net/http"

func (h *Handler) getRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	var enabled bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT registration_enabled FROM platform_settings WHERE id = 1").Scan(&enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read registration setting")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"registration_enabled": enabled})
}

func (h *Handler) setRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Enabled == nil {
		writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE platform_settings SET registration_enabled = ? WHERE id = 1", *input.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update registration setting")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"registration_enabled": *input.Enabled})
}

func (h *Handler) requireRoot(w http.ResponseWriter, r *http.Request) bool {
	user, ok := h.currentUser(w, r)
	if !ok {
		return false
	}
	if !user.IsRoot {
		writeError(w, http.StatusForbidden, "root access required")
		return false
	}
	return true
}
