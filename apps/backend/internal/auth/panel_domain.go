package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

var panelDomainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

func (h *Handler) panelRoot(w http.ResponseWriter, r *http.Request) bool {
	user, ok := h.currentUser(w, r)
	if !ok {
		return false
	}
	if !user.IsRoot {
		writeError(w, http.StatusForbidden, "root account required")
		return false
	}
	return true
}

func validPanelDomain(raw string) (string, bool) {
	domain := strings.ToLower(strings.TrimSpace(raw))
	return domain, domain == "" || len(domain) <= 253 && panelDomainPattern.MatchString(domain) && net.ParseIP(domain) == nil
}

func (h *Handler) getPanelDomain(w http.ResponseWriter, r *http.Request) {
	if !h.panelRoot(w, r) {
		return
	}
	var domain sql.NullString
	if err := h.db.QueryRowContext(r.Context(), "SELECT panel_domain FROM platform_settings WHERE id = 1").Scan(&domain); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read panel domain")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"domain": domain.String})
}

func (h *Handler) setPanelDomain(w http.ResponseWriter, r *http.Request) {
	if !h.panelRoot(w, r) {
		return
	}
	var input struct {
		Domain string `json:"domain"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	domain, valid := validPanelDomain(input.Domain)
	if !valid {
		writeError(w, http.StatusBadRequest, "enter a valid public domain name")
		return
	}
	if domain != "" {
		var used bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM database_services WHERE domain = ?) OR EXISTS(SELECT 1 FROM platform_settings WHERE panel_domain = ? AND id = 1)", domain, domain).Scan(&used); err != nil {
			writeError(w, http.StatusInternalServerError, "could not check domain")
			return
		}
		if used {
			writeError(w, http.StatusConflict, "domain is already assigned to an app")
			return
		}
	}
	var savedDomain any
	if domain != "" {
		savedDomain = domain
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE platform_settings SET panel_domain = ? WHERE id = 1", savedDomain); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save panel domain")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"domain": domain})
}

func (h *Handler) checkPanelDomainDNS(w http.ResponseWriter, r *http.Request) {
	if !h.panelRoot(w, r) {
		return
	}
	domain, valid := validPanelDomain(r.URL.Query().Get("domain"))
	if !valid || domain == "" {
		writeError(w, http.StatusBadRequest, "enter a valid public domain name")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	addresses, err := net.DefaultResolver.LookupHost(ctx, domain)
	writeJSON(w, http.StatusOK, map[string]any{"resolves": err == nil && len(addresses) > 0, "addresses": addresses})
}

func (h *Handler) panelTraefikConfig(w http.ResponseWriter, r *http.Request) {
	token := os.Getenv("TRAEFIK_CONFIG_TOKEN")
	if token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid Traefik provider token")
		return
	}
	var domain sql.NullString
	if err := h.db.QueryRowContext(r.Context(), "SELECT panel_domain FROM platform_settings WHERE id = 1").Scan(&domain); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read panel domain")
		return
	}
	if !domain.Valid || domain.String == "" {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	apiTarget, err := panelTarget("TRAEFIK_PANEL_API_TARGET")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "panel API target is not configured")
		return
	}
	webTarget, err := panelTarget("TRAEFIK_PANEL_WEB_TARGET")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "panel web target is not configured")
		return
	}
	host := "Host(`" + domain.String + "`)"
	apiRule := host + " && (Path(`/api`) || PathPrefix(`/api/`) || Path(`/health`))"
	webRule := host + " && !Path(`/api`) && !PathPrefix(`/api/`) && !Path(`/health`)"
	writeJSON(w, http.StatusOK, map[string]any{
		"http": map[string]any{
			"routers": map[string]any{
				"panel-api":      map[string]any{"entryPoints": []string{"websecure"}, "rule": apiRule, "priority": 200, "service": "panel-api", "tls": map[string]string{"certResolver": "letsencrypt"}},
				"panel-web":      map[string]any{"entryPoints": []string{"websecure"}, "rule": webRule, "priority": 200, "service": "panel-web", "tls": map[string]string{"certResolver": "letsencrypt"}},
				"panel-api-http": map[string]any{"entryPoints": []string{"web"}, "rule": apiRule, "priority": 200, "service": "panel-api", "middlewares": []string{"panel-https"}},
				"panel-web-http": map[string]any{"entryPoints": []string{"web"}, "rule": webRule, "priority": 200, "service": "panel-web", "middlewares": []string{"panel-https"}},
			},
			"middlewares": map[string]any{"panel-https": map[string]any{"redirectScheme": map[string]string{"scheme": "https"}}},
			"services": map[string]any{
				"panel-api": map[string]any{"loadBalancer": map[string]any{"servers": []map[string]string{{"url": apiTarget}}}},
				"panel-web": map[string]any{"loadBalancer": map[string]any{"servers": []map[string]string{{"url": webTarget}}}},
			},
		},
	})
}

func panelTarget(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	target, err := url.Parse(value)
	if err != nil || target.Scheme != "http" || target.Host == "" || target.User != nil || target.RawQuery != "" || target.Fragment != "" || target.Path != "" {
		return "", errors.New("invalid panel target URL")
	}
	return value, nil
}
