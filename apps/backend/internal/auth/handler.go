package auth

import (
	"database/sql"
	"net/http"
)

type Handler struct {
	db            *sql.DB
	secureOnboard bool
}

type User struct {
	ID       uint64 `json:"id,string"`
	Username string `json:"username"`
	Email    string `json:"email"`
	IsRoot   bool   `json:"is_root"`
}

func New(db *sql.DB, secureOnboard bool) *Handler {
	return &Handler{db: db, secureOnboard: secureOnboard}
}

func OnboardingPending(db *sql.DB) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users)").Scan(&exists)
	return !exists, err
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/onboarding", h.onboardingStatus)
	mux.HandleFunc("POST /api/onboarding", h.onboard)
	mux.HandleFunc("POST /api/auth/register", h.register)
	mux.HandleFunc("POST /api/invitations/accept", h.acceptInvitation)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", h.me)
	mux.HandleFunc("PUT /api/account/profile", h.updateProfile)
	mux.HandleFunc("PUT /api/account/password", h.updatePassword)
	mux.HandleFunc("GET /api/settings/registration", h.getRegistration)
	mux.HandleFunc("PUT /api/settings/registration", h.setRegistration)
	mux.HandleFunc("GET /api/settings/panel-domain", h.getPanelDomain)
	mux.HandleFunc("PUT /api/settings/panel-domain", h.setPanelDomain)
	mux.HandleFunc("GET /api/settings/panel-domain/dns", h.checkPanelDomainDNS)
	mux.HandleFunc("GET /internal/traefik/panel-domain", h.panelTraefikConfig)
}
