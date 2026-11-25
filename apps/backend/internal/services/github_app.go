package services

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
	"kymber/backend/internal/secrets"
)

var gitHubAppNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.-]{1,33}$`)
var setupStatePattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func gitHubAppJWT(appID int64, privateKeyPEM string, now time.Time) (string, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", errors.New("invalid GitHub App private key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsed, pkcs8Err := x509.ParsePKCS8PrivateKey(block.Bytes)
		rsaKey, isRSA := parsed.(*rsa.PrivateKey)
		if pkcs8Err != nil || !isRSA {
			return "", errors.New("invalid GitHub App private key")
		}
		key = rsaKey
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]int64{"iat": now.Add(-time.Minute).Unix(), "exp": now.Add(9 * time.Minute).Unix(), "iss": appID})
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (p *gitProvider) appAuthorization() (string, error) {
	token, err := gitHubAppJWT(p.credentials.AppID, p.credentials.PrivateKey, time.Now())
	return "Bearer " + token, err
}

func (p *gitProvider) installationToken(ctx context.Context, installationID int64, scope map[string]any) (string, error) {
	authorization, err := p.appAuthorization()
	if err != nil {
		return "", err
	}
	var token struct {
		Token string `json:"token"`
	}
	if scope == nil {
		scope = map[string]any{}
	}
	if err := providerAPI(ctx, http.MethodPost, fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID), authorization, scope, &token); err != nil {
		return "", fmt.Errorf("could not get a GitHub App token: %w", err)
	}
	if token.Token == "" {
		return "", errors.New("GitHub returned an empty installation token")
	}
	return token.Token, nil
}

func (h *Handler) panelURL(ctx context.Context) (string, error) {
	var domain string
	if err := h.db.QueryRowContext(ctx, "SELECT COALESCE(panel_domain, '') FROM platform_settings WHERE id = 1").Scan(&domain); err != nil {
		return "", err
	}
	if domain == "" {
		return "", errors.New("set the panel domain in System settings first; GitHub needs a public HTTPS address")
	}
	return "https://" + domain, nil
}

func (h *Handler) startGitHubApp(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.WorkspaceManage) {
		return
	}
	var input struct {
		Name         string `json:"name"`
		Organization string `json:"organization"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Organization = strings.TrimSpace(input.Organization)
	if !gitHubAppNamePattern.MatchString(input.Name) {
		writeError(w, http.StatusBadRequest, "GitHub App names have 2 to 34 letters, numbers, spaces, dots, dashes or underscores")
		return
	}
	if input.Organization != "" && !gitHubOwnerPattern.MatchString(input.Organization) {
		writeError(w, http.StatusBadRequest, "invalid GitHub organization")
		return
	}
	panel, err := h.panelURL(r.Context())
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create GitHub App state")
		return
	}
	state := hex.EncodeToString(stateBytes)
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create Git provider")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO git_providers (id, workspace_id, kind, name, setup_state, created_by) VALUES (?, ?, 'github_app', ?, ?, ?)", id, workspaceID, input.Name, state, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save Git provider")
		return
	}
	manifest, _ := json.Marshal(map[string]any{
		"name":            input.Name,
		"url":             panel,
		"hook_attributes": map[string]any{"url": panel + "/api/webhooks/github", "active": true},
		"redirect_url":    panel + "/api/git-providers/github-app/callback",
		"setup_url":       panel + "/api/git-providers/github-app/setup",
		"setup_on_update": true,
		"public":          false,
		"default_permissions": map[string]string{
			"contents": "read", "metadata": "read", "statuses": "write", "pull_requests": "write",
		},
		"default_events": []string{"push", "pull_request"},
	})
	target := "https://github.com/settings/apps/new?state=" + state
	if input.Organization != "" {
		target = "https://github.com/organizations/" + url.PathEscape(input.Organization) + "/settings/apps/new?state=" + state
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "url": target, "manifest": string(manifest)})
}

func (h *Handler) providerBySetupState(r *http.Request) (*gitProvider, error) {
	state := r.URL.Query().Get("state")
	if !setupStatePattern.MatchString(state) {
		return nil, errors.New("invalid state")
	}
	var id, workspaceID string
	if err := h.db.QueryRowContext(r.Context(), "SELECT id, workspace_id FROM git_providers WHERE setup_state = ? AND kind = 'github_app'", state).Scan(&id, &workspaceID); err != nil {
		return nil, errors.New("unknown state")
	}
	return h.loadGitProvider(r.Context(), workspaceID, id)
}

func (h *Handler) gitHubAppCallback(w http.ResponseWriter, r *http.Request) {
	provider, err := h.providerBySetupState(r)
	if err != nil {
		http.Error(w, "GitHub App setup link is invalid or expired", http.StatusBadRequest)
		return
	}
	if provider.Ready {
		http.Redirect(w, r, provider.InstallURL, http.StatusFound)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" || len(code) > 128 || strings.ContainsAny(code, "/?#") {
		http.Error(w, "missing GitHub manifest code", http.StatusBadRequest)
		return
	}
	var app struct {
		ID            int64  `json:"id"`
		Slug          string `json:"slug"`
		PEM           string `json:"pem"`
		WebhookSecret string `json:"webhook_secret"`
	}
	if err := providerAPI(r.Context(), http.MethodPost, "https://api.github.com/app-manifests/"+url.PathEscape(code)+"/conversions", "", nil, &app); err != nil {
		http.Error(w, "GitHub did not create the App: "+err.Error(), http.StatusBadGateway)
		return
	}
	if app.ID == 0 || app.Slug == "" || app.PEM == "" || app.WebhookSecret == "" {
		http.Error(w, "GitHub returned an incomplete App", http.StatusBadGateway)
		return
	}
	plaintext, _ := json.Marshal(gitCredentials{AppID: app.ID, PrivateKey: app.PEM, WebhookSecret: app.WebhookSecret})
	encrypted, err := secrets.Encrypt(plaintext)
	if err != nil {
		http.Error(w, "service encryption is not configured", http.StatusInternalServerError)
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE git_providers SET app_slug = ?, encrypted_credentials = ? WHERE id = ? AND encrypted_credentials IS NULL", app.Slug, encrypted, provider.ID); err != nil {
		http.Error(w, "could not save the GitHub App", http.StatusInternalServerError)
		return
	}
	provider.AppSlug = app.Slug
	http.Redirect(w, r, "https://github.com/apps/"+url.PathEscape(app.Slug)+"/installations/new?state="+r.URL.Query().Get("state"), http.StatusFound)
}

func (h *Handler) gitHubAppSetup(w http.ResponseWriter, r *http.Request) {
	provider, err := h.providerBySetupState(r)
	if err != nil || !provider.Ready {
		http.Error(w, "GitHub App setup link is invalid or expired", http.StatusBadRequest)
		return
	}
	if err := h.syncInstallations(r.Context(), provider); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/settings/git", http.StatusFound)
}

func (h *Handler) syncInstallations(ctx context.Context, provider *gitProvider) error {
	authorization, err := provider.appAuthorization()
	if err != nil {
		return err
	}
	var installations []struct {
		ID      int64 `json:"id"`
		Account struct {
			Login string `json:"login"`
		} `json:"account"`
	}
	if err := providerAPI(ctx, http.MethodGet, "https://api.github.com/app/installations?per_page=100", authorization, nil, &installations); err != nil {
		return fmt.Errorf("could not list GitHub App installations: %w", err)
	}
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM git_provider_installations WHERE provider_id = ?", provider.ID); err != nil {
		return err
	}
	for _, installation := range installations {
		if _, err := tx.ExecContext(ctx, "INSERT INTO git_provider_installations (provider_id, installation_id, account_login) VALUES (?, ?, ?)", provider.ID, installation.ID, installation.Account.Login); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (h *Handler) syncGitHubApp(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
		return
	}
	providerID, ok := pathID(w, r, "providerID")
	if !ok {
		return
	}
	provider, err := h.loadGitProvider(r.Context(), workspaceID, providerID)
	if err != nil || provider.Kind != "github_app" || !provider.Ready {
		writeError(w, http.StatusNotFound, "GitHub App not found")
		return
	}
	if err := h.syncInstallations(r.Context(), provider); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	installations, err := h.readInstallations(r.Context(), providerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read installations")
		return
	}
	writeJSON(w, http.StatusOK, installations)
}
