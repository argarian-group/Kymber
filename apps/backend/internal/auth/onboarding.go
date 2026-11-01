package auth

import (
	"net/http"

	"kymber/backend/internal/logger"
)

func (h *Handler) onboardingStatus(w http.ResponseWriter, r *http.Request) {
	var exists bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM users)").Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check onboarding status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"required": !exists})
}

func (h *Handler) onboard(w http.ResponseWriter, r *http.Request) {
	var input credentials
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validateCredentials(&input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	passwordHash, err := hashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start onboarding")
		return
	}
	defer tx.Rollback()
	var settingsID uint8
	if err := tx.QueryRowContext(r.Context(), "SELECT id FROM platform_settings WHERE id = 1 FOR UPDATE").Scan(&settingsID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not lock platform settings")
		return
	}
	var count int
	if err := tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check users")
		return
	}
	if count != 0 {
		writeError(w, http.StatusConflict, "onboarding is already complete")
		return
	}
	userID, err := createUser(r, tx, input.Username, input.Email, passwordHash, true)
	if err != nil {
		writeUserError(w, err)
		return
	}
	if err := createWorkspace(r, tx, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create root workspace")
		return
	}
	token, expiresAt, err := createSession(r, tx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not complete onboarding")
		return
	}
	logger.Info("Root user created: %s <%s>", input.Username, input.Email)
	setSessionCookie(w, token, expiresAt)
	writeJSON(w, http.StatusCreated, map[string]any{"user": User{ID: userID, Username: input.Username, Email: input.Email, IsRoot: true}, "restart_required": h.secureOnboard})
}
