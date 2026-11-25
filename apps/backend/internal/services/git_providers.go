package services

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
	"kymber/backend/internal/secrets"
)

type gitInstallation struct {
	ID           int64  `json:"id"`
	AccountLogin string `json:"account_login"`
}

type gitProvider struct {
	ID            string            `json:"id"`
	Kind          string            `json:"kind"`
	Name          string            `json:"name"`
	BaseURL       string            `json:"base_url,omitempty"`
	AppSlug       string            `json:"app_slug,omitempty"`
	PublicKey     string            `json:"public_key,omitempty"`
	InstallURL    string            `json:"install_url,omitempty"`
	Ready         bool              `json:"ready"`
	Installations []gitInstallation `json:"installations,omitempty"`
	CreatedAt     string            `json:"created_at"`
	workspaceID   string
	credentials   gitCredentials
}

type gitCredentials struct {
	Token         string `json:"token,omitempty"`
	Username      string `json:"username,omitempty"`
	AppID         int64  `json:"app_id,omitempty"`
	PrivateKey    string `json:"private_key,omitempty"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
	KnownHosts    string `json:"known_hosts,omitempty"`
}

type gitRepository struct {
	provider       *gitProvider
	installationID int64
	url            string
	path           string
}

type gitCloneAuth struct {
	Header     string `json:"header,omitempty"`
	SSHKey     string `json:"ssh_key,omitempty"`
	KnownHosts string `json:"known_hosts,omitempty"`
}

type gitBranch struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

var gitPathSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]{0,99}$`)
var sshUserPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

func (p *gitProvider) kind() string {
	if p == nil {
		return "github_public"
	}
	return p.Kind
}

func (p *gitProvider) host() string {
	switch p.kind() {
	case "gitlab":
		u, _ := url.Parse(p.BaseURL)
		return u.Host
	case "bitbucket":
		return "bitbucket.org"
	case "git_ssh":
		return p.BaseURL
	}
	return "github.com"
}

func (p *gitProvider) apiBase() string {
	switch p.kind() {
	case "gitlab":
		return p.BaseURL + "/api/v4"
	case "bitbucket":
		return "https://api.bitbucket.org/2.0"
	}
	return "https://api.github.com"
}

func (p *gitProvider) repositoryPath(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery != "" || u.Fragment != "" || u.Host != p.host() {
		return "", false
	}
	if p.kind() == "git_ssh" {
		if u.Scheme != "ssh" || u.User == nil || !sshUserPattern.MatchString(u.User.Username()) {
			return "", false
		}
		if _, hasPassword := u.User.Password(); hasPassword {
			return "", false
		}
	} else if u.Scheme != "https" || u.User != nil {
		return "", false
	}
	parts := strings.Split(strings.TrimPrefix(strings.TrimSuffix(u.Path, ".git"), "/"), "/")
	maxParts := 2
	if p.kind() == "gitlab" || p.kind() == "git_ssh" {
		maxParts = 20
	}
	if len(parts) < 2 || len(parts) > maxParts {
		return "", false
	}
	for _, part := range parts {
		if !gitPathSegmentPattern.MatchString(part) {
			return "", false
		}
	}
	if p.kind() == "github_public" || p.kind() == "github_token" || p.kind() == "github_app" {
		if !gitHubOwnerPattern.MatchString(parts[0]) {
			return "", false
		}
	}
	return strings.Join(parts, "/"), true
}

var gitAPIClient = &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{
	DialContext: publicDialer().DialContext,
}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func publicDialer() *net.Dialer {
	return &net.Dialer{Timeout: 10 * time.Second, Control: func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return err
		}
		ip := net.ParseIP(host)
		if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
			return fmt.Errorf("refusing to connect to non-public address %s", host)
		}
		return nil
	}}
}

func (r gitRepository) apiAuthorization(ctx context.Context) (string, error) {
	p := r.provider
	switch p.kind() {
	case "github_public":
		return "", nil
	case "github_token", "gitlab":
		return "Bearer " + p.credentials.Token, nil
	case "bitbucket":
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(p.credentials.Username+":"+p.credentials.Token)), nil
	case "github_app":
		token, err := p.installationToken(ctx, r.installationID, nil)
		return "Bearer " + token, err
	}
	return "", errors.New("this provider has no API")
}

func providerAPI(ctx context.Context, method, target, authorization string, body, output any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if strings.Contains(target, "api.github.com") {
		request.Header.Set("Accept", "application/vnd.github+json")
	}
	request.Header.Set("User-Agent", "Kymber-build-controller")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response, err := gitAPIClient.Do(request)
	if err != nil {
		return fmt.Errorf("could not reach the Git provider: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return err
	}
	if response.StatusCode == http.StatusNotFound {
		return errors.New("repository or ref was not found, or the credentials cannot access it")
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return fmt.Errorf("the Git provider rejected the credentials (HTTP %d)", response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Errorf("the Git provider returned HTTP %d", response.StatusCode)
	}
	if output != nil && len(data) > 0 {
		if err := json.Unmarshal(data, output); err != nil {
			return errors.New("the Git provider returned an invalid response")
		}
	}
	return nil
}

func (r gitRepository) api(ctx context.Context, method, path string, body, output any) error {
	authorization, err := r.apiAuthorization(ctx)
	if err != nil {
		return err
	}
	return providerAPI(ctx, method, r.provider.apiBase()+path, authorization, body, output)
}

func (r gitRepository) apiPath() string {
	if r.provider.kind() == "gitlab" {
		return "/projects/" + url.PathEscape(r.path)
	}
	if r.provider.kind() == "bitbucket" {
		return "/repositories/" + r.path
	}
	return "/repos/" + r.path
}

func (r gitRepository) resolveCommit(ctx context.Context, ref string) (string, error) {
	var commit struct {
		SHA  string `json:"sha"`
		ID   string `json:"id"`
		Hash string `json:"hash"`
	}
	switch r.provider.kind() {
	case "git_ssh":
		return "", nil
	case "gitlab":
		if err := r.api(ctx, http.MethodGet, r.apiPath()+"/repository/commits/"+url.PathEscape(ref), nil, &commit); err != nil {
			return "", err
		}
	case "bitbucket":
		if err := r.api(ctx, http.MethodGet, r.apiPath()+"/commit/"+url.PathEscape(ref), nil, &commit); err != nil {
			return "", err
		}
	default:
		if err := r.api(ctx, http.MethodGet, r.apiPath()+"/commits/"+url.PathEscape(ref), nil, &commit); err != nil {
			return "", err
		}
	}
	sha := commit.SHA + commit.ID + commit.Hash
	if !gitCommitPattern.MatchString(sha) {
		return "", errors.New("the Git provider returned an invalid commit")
	}
	return sha, nil
}

func (r gitRepository) branches(ctx context.Context) ([]gitBranch, error) {
	branches := []gitBranch{}
	switch r.provider.kind() {
	case "git_ssh":
		return branches, nil
	case "gitlab":
		var found []struct {
			Name   string `json:"name"`
			Commit struct {
				ID string `json:"id"`
			} `json:"commit"`
		}
		if err := r.api(ctx, http.MethodGet, r.apiPath()+"/repository/branches?per_page=100", nil, &found); err != nil {
			return nil, err
		}
		for _, item := range found {
			branch := gitBranch{Name: item.Name}
			branch.Commit.SHA = item.Commit.ID
			branches = append(branches, branch)
		}
	case "bitbucket":
		var found struct {
			Values []struct {
				Name   string `json:"name"`
				Target struct {
					Hash string `json:"hash"`
				} `json:"target"`
			} `json:"values"`
		}
		if err := r.api(ctx, http.MethodGet, r.apiPath()+"/refs/branches?pagelen=100", nil, &found); err != nil {
			return nil, err
		}
		for _, item := range found.Values {
			branch := gitBranch{Name: item.Name}
			branch.Commit.SHA = item.Target.Hash
			branches = append(branches, branch)
		}
	default:
		if err := r.api(ctx, http.MethodGet, r.apiPath()+"/branches?per_page=100", nil, &branches); err != nil {
			return nil, err
		}
	}
	if len(branches) > 100 {
		branches = branches[:100]
	}
	return branches, nil
}

func (r gitRepository) cloneAuth(ctx context.Context) (gitCloneAuth, error) {
	p := r.provider
	basic := func(user, password string) string {
		return "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
	}
	switch p.kind() {
	case "github_token":
		return gitCloneAuth{Header: basic("x-access-token", p.credentials.Token)}, nil
	case "github_app":
		name := r.path[strings.Index(r.path, "/")+1:]
		token, err := p.installationToken(ctx, r.installationID, map[string]any{"repositories": []string{name}, "permissions": map[string]string{"contents": "read"}})
		if err != nil {
			return gitCloneAuth{}, err
		}
		return gitCloneAuth{Header: basic("x-access-token", token)}, nil
	case "gitlab":
		return gitCloneAuth{Header: basic("oauth2", p.credentials.Token)}, nil
	case "bitbucket":
		return gitCloneAuth{Header: basic("x-bitbucket-api-token-auth", p.credentials.Token)}, nil
	case "git_ssh":
		return gitCloneAuth{SSHKey: p.credentials.PrivateKey, KnownHosts: p.credentials.KnownHosts}, nil
	}
	return gitCloneAuth{}, nil
}

func (h *Handler) loadGitProvider(ctx context.Context, workspaceID, providerID string) (*gitProvider, error) {
	var provider gitProvider
	var publicKey sql.NullString
	var encrypted []byte
	var setupState sql.NullString
	if err := h.db.QueryRowContext(ctx, `SELECT id, workspace_id, kind, name, base_url, app_slug, public_key, encrypted_credentials, setup_state, created_at FROM git_providers WHERE id = ? AND workspace_id = ?`, providerID, workspaceID).
		Scan(&provider.ID, &provider.workspaceID, &provider.Kind, &provider.Name, &provider.BaseURL, &provider.AppSlug, &publicKey, &encrypted, &setupState, &provider.CreatedAt); err != nil {
		return nil, err
	}
	provider.PublicKey = publicKey.String
	if encrypted != nil {
		plaintext, err := secrets.Decrypt(encrypted)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(plaintext, &provider.credentials); err != nil {
			return nil, err
		}
	}
	provider.Ready = encrypted != nil
	if provider.Kind == "github_app" && provider.AppSlug != "" && setupState.Valid {
		provider.InstallURL = "https://github.com/apps/" + url.PathEscape(provider.AppSlug) + "/installations/new?state=" + setupState.String
	}
	return &provider, nil
}

func (h *Handler) openGitRepository(ctx context.Context, workspaceID string, source gitSource) (gitRepository, error) {
	repository := gitRepository{url: source.RepositoryURL, installationID: source.InstallationID}
	if source.ProviderID != "" {
		provider, err := h.loadGitProvider(ctx, workspaceID, source.ProviderID)
		if err != nil {
			return repository, errors.New("Git provider not found")
		}
		if !provider.Ready {
			return repository, errors.New("finish setting up the Git provider first")
		}
		repository.provider = provider
	}
	path, ok := repository.provider.repositoryPath(source.RepositoryURL)
	if !ok {
		return repository, fmt.Errorf("repository URL must be a %s repository on %s", repository.provider.kind(), repository.provider.host())
	}
	repository.path = path
	if repository.provider.kind() == "github_app" {
		var exists bool
		if err := h.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM git_provider_installations WHERE provider_id = ? AND installation_id = ?)", source.ProviderID, source.InstallationID).Scan(&exists); err != nil || !exists {
			return repository, errors.New("select a GitHub App installation")
		}
	} else {
		repository.installationID = 0
	}
	return repository, nil
}

func (h *Handler) listGitProviders(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id FROM git_providers WHERE workspace_id = ? ORDER BY created_at", workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list Git providers")
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "could not list Git providers")
			return
		}
		ids = append(ids, id)
	}
	rows.Close()
	providers := make([]*gitProvider, 0, len(ids))
	for _, id := range ids {
		provider, err := h.loadGitProvider(r.Context(), workspaceID, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not read Git provider")
			return
		}
		if provider.Kind == "github_app" {
			if provider.Installations, err = h.readInstallations(r.Context(), id); err != nil {
				writeError(w, http.StatusInternalServerError, "could not read GitHub App installations")
				return
			}
		}
		providers = append(providers, provider)
	}
	writeJSON(w, http.StatusOK, providers)
}

func (h *Handler) readInstallations(ctx context.Context, providerID string) ([]gitInstallation, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT installation_id, account_login FROM git_provider_installations WHERE provider_id = ? ORDER BY account_login", providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	installations := []gitInstallation{}
	for rows.Next() {
		var installation gitInstallation
		if err := rows.Scan(&installation.ID, &installation.AccountLogin); err != nil {
			return nil, err
		}
		installations = append(installations, installation)
	}
	return installations, rows.Err()
}

func (h *Handler) createGitProvider(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.WorkspaceManage) {
		return
	}
	var input struct {
		Kind     string `json:"kind"`
		Name     string `json:"name"`
		BaseURL  string `json:"base_url"`
		Username string `json:"username"`
		Token    string `json:"token"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Token = strings.TrimSpace(input.Token)
	if input.Name == "" || len(input.Name) > 120 {
		writeError(w, http.StatusBadRequest, "a name of up to 120 characters is required")
		return
	}
	provider := gitProvider{Kind: input.Kind, Name: input.Name}
	var credentials gitCredentials
	switch input.Kind {
	case "github_token", "gitlab", "bitbucket":
		if input.Token == "" || len(input.Token) > 512 {
			writeError(w, http.StatusBadRequest, "an access token is required")
			return
		}
		credentials = gitCredentials{Token: input.Token, Username: strings.TrimSpace(input.Username)}
		if input.Kind == "gitlab" {
			base := strings.TrimSuffix(strings.TrimSpace(input.BaseURL), "/")
			if base == "" {
				base = "https://gitlab.com"
			}
			u, err := url.Parse(base)
			if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
				writeError(w, http.StatusBadRequest, "GitLab URL must look like https://gitlab.example.com")
				return
			}
			provider.BaseURL = "https://" + u.Host
		}
		if input.Kind == "bitbucket" && credentials.Username == "" {
			writeError(w, http.StatusBadRequest, "the Atlassian account email is required for Bitbucket")
			return
		}
		provider.credentials = credentials
		check := map[string]string{"github_token": "/user", "gitlab": "/user", "bitbucket": "/user"}[input.Kind]
		authorization, _ := gitRepository{provider: &provider}.apiAuthorization(r.Context())
		if err := providerAPI(r.Context(), http.MethodGet, provider.apiBase()+check, authorization, nil, nil); err != nil {
			writeError(w, http.StatusBadRequest, "could not verify the token: "+err.Error())
			return
		}
	case "git_ssh":
		host := strings.ToLower(strings.TrimSpace(input.BaseURL))
		hostname, port, err := net.SplitHostPort(host)
		if err != nil {
			hostname, port = host, "22"
		}
		portNumber, portErr := strconv.Atoi(port)
		if !serviceDomainPattern.MatchString(hostname) || portErr != nil || portNumber < 1 || portNumber > 65535 {
			writeError(w, http.StatusBadRequest, "enter the Git server host, for example git.example.com or git.example.com:2222")
			return
		}
		provider.BaseURL = hostname
		if port != "22" {
			provider.BaseURL = net.JoinHostPort(hostname, port)
		}
		privateKey, publicKey, err := newDeployKey()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not generate deploy key")
			return
		}
		knownHosts, err := scanHostKey(r.Context(), net.JoinHostPort(hostname, port))
		if err != nil {
			writeError(w, http.StatusBadGateway, "could not read the Git server host key: "+err.Error())
			return
		}
		credentials = gitCredentials{PrivateKey: privateKey, KnownHosts: knownHosts}
		provider.PublicKey = publicKey
	default:
		writeError(w, http.StatusBadRequest, "unsupported Git provider")
		return
	}
	plaintext, _ := json.Marshal(credentials)
	encrypted, err := secrets.Encrypt(plaintext)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "service encryption is not configured")
		return
	}
	provider.ID, err = database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create Git provider")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `INSERT INTO git_providers (id, workspace_id, kind, name, base_url, public_key, encrypted_credentials, created_by) VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?)`,
		provider.ID, workspaceID, provider.Kind, provider.Name, provider.BaseURL, provider.PublicKey, encrypted, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save Git provider")
		return
	}
	created, err := h.loadGitProvider(r.Context(), workspaceID, provider.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read Git provider")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func newDeployKey() (string, string, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	block, err := ssh.MarshalPrivateKey(private, "kymber deploy key")
	if err != nil {
		return "", "", err
	}
	sshPublic, err := ssh.NewPublicKey(public)
	if err != nil {
		return "", "", err
	}
	return string(pem.EncodeToMemory(block)), strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublic))) + " kymber", nil
}

func scanHostKey(ctx context.Context, address string) (string, error) {
	var line string
	config := &ssh.ClientConfig{User: "git", Timeout: 10 * time.Second, HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
		line = knownhosts.Line([]string{knownhosts.Normalize(address)}, key)
		return errors.New("host key captured")
	}}
	connection, err := publicDialer().DialContext(ctx, "tcp", address)
	if err != nil {
		return "", err
	}
	defer connection.Close()
	_, _, _, _ = ssh.NewClientConn(connection, address, config)
	if line == "" {
		return "", errors.New("the server did not present an SSH host key")
	}
	return line, nil
}

func (h *Handler) deleteGitProvider(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.WorkspaceManage) {
		return
	}
	providerID, ok := pathID(w, r, "providerID")
	if !ok {
		return
	}
	var used bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM service_git_sources WHERE provider_id = ?)", providerID).Scan(&used); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check Git provider usage")
		return
	}
	if used {
		writeError(w, http.StatusConflict, "this provider is used by apps; change their Git source first")
		return
	}
	result, err := h.db.ExecContext(r.Context(), "DELETE FROM git_providers WHERE id = ? AND workspace_id = ?", providerID, workspaceID)
	if err != nil {
		writeError(w, http.StatusConflict, "could not delete Git provider")
		return
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		writeError(w, http.StatusNotFound, "Git provider not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listGitProviderRepositories(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || !provider.Ready {
		writeError(w, http.StatusNotFound, "Git provider not found")
		return
	}
	installationID, _ := strconv.ParseInt(r.URL.Query().Get("installation_id"), 10, 64)
	repository := gitRepository{provider: provider, installationID: installationID}
	if provider.Kind == "github_app" {
		var exists bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM git_provider_installations WHERE provider_id = ? AND installation_id = ?)", providerID, installationID).Scan(&exists); err != nil || !exists {
			writeError(w, http.StatusBadRequest, "select a GitHub App installation")
			return
		}
	}
	type listing struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	repositories := []listing{}
	var names []string
	switch provider.Kind {
	case "github_app":
		var found struct {
			Repositories []struct {
				FullName string `json:"full_name"`
			} `json:"repositories"`
		}
		err = repository.api(r.Context(), http.MethodGet, "/installation/repositories?per_page=100", nil, &found)
		for _, item := range found.Repositories {
			names = append(names, item.FullName)
		}
	case "github_token":
		var found []struct {
			FullName string `json:"full_name"`
		}
		err = repository.api(r.Context(), http.MethodGet, "/user/repos?per_page=100&sort=updated", nil, &found)
		for _, item := range found {
			names = append(names, item.FullName)
		}
	case "gitlab":
		var found []struct {
			Path string `json:"path_with_namespace"`
		}
		err = repository.api(r.Context(), http.MethodGet, "/projects?membership=true&per_page=100&order_by=last_activity_at&simple=true", nil, &found)
		for _, item := range found {
			names = append(names, item.Path)
		}
	case "bitbucket":
		var found struct {
			Values []struct {
				FullName string `json:"full_name"`
			} `json:"values"`
		}
		err = repository.api(r.Context(), http.MethodGet, "/repositories?role=member&pagelen=100&sort=-updated_on", nil, &found)
		for _, item := range found.Values {
			names = append(names, item.FullName)
		}
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	for _, name := range names {
		repositoryURL := "https://" + provider.host() + "/" + name
		if _, ok := provider.repositoryPath(repositoryURL); ok {
			repositories = append(repositories, listing{Name: name, URL: repositoryURL})
		}
	}
	writeJSON(w, http.StatusOK, repositories)
}
