package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"regexp"
	"strings"
	"time"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/permissions"
	"kymber/backend/internal/secrets"
)

type gitSource struct {
	ProviderID       string   `json:"provider_id"`
	InstallationID   int64    `json:"installation_id"`
	RepositoryURL    string   `json:"repository_url"`
	Ref              string   `json:"ref"`
	BaseDirectory    string   `json:"base_directory"`
	DockerfilePath   string   `json:"dockerfile_path"`
	BuildPack        string   `json:"build_pack"`
	InstallCommand   string   `json:"install_command"`
	BuildCommand     string   `json:"build_command"`
	StartCommand     string   `json:"start_command"`
	PublishDirectory string   `json:"publish_directory"`
	BuildArgs        []string `json:"build_args"`
	AutoDeploy       bool     `json:"auto_deploy"`
	WebhookURL       string   `json:"webhook_url,omitempty"`
	WebhookSecret    string   `json:"webhook_secret,omitempty"`
}

var repositoryPathPattern = regexp.MustCompile(`^[A-Za-z0-9._@+-]+(/[A-Za-z0-9._@+-]+)*$`)
var buildArgKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,127}$`)

const maxBuildArgs = 50

func validRepositoryPath(value string) bool {
	if len(value) > 255 || !repositoryPathPattern.MatchString(value) || path.Clean(value) != value {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}

func validateGitBuildConfiguration(source *gitSource) error {
	if source.BuildPack == "" {
		source.BuildPack = "dockerfile"
	}
	source.BaseDirectory = strings.Trim(strings.TrimSpace(source.BaseDirectory), "/")
	source.DockerfilePath = strings.TrimPrefix(strings.TrimSpace(source.DockerfilePath), "/")
	if source.DockerfilePath == "" {
		source.DockerfilePath = "Dockerfile"
	}
	if source.BaseDirectory != "" && !validRepositoryPath(source.BaseDirectory) {
		return errors.New("base directory must be a relative path inside the repository")
	}
	if !validRepositoryPath(source.DockerfilePath) {
		return errors.New("Dockerfile path must be a relative path inside the base directory")
	}
	source.PublishDirectory = strings.Trim(strings.TrimSpace(source.PublishDirectory), "/")
	commands := []string{source.InstallCommand, source.BuildCommand, source.StartCommand}
	switch source.BuildPack {
	case "dockerfile":
		if strings.Join(commands, "") != "" || source.PublishDirectory != "" {
			return errors.New("Dockerfile builds take their commands from the Dockerfile")
		}
	case "nixpacks":
		if source.PublishDirectory != "" {
			return errors.New("publish directory only applies to static sites")
		}
		for _, command := range commands {
			if len(command) > 512 || strings.ContainsAny(command, "\x00\r\n") {
				return errors.New("commands must be a single line of up to 512 characters")
			}
		}
	case "static":
		if strings.Join(commands, "") != "" {
			return errors.New("static sites are served as-is; use Nixpacks to run build commands")
		}
		if source.PublishDirectory != "" && !validRepositoryPath(source.PublishDirectory) {
			return errors.New("publish directory must be a relative path inside the base directory")
		}
	default:
		return errors.New("build pack must be dockerfile, nixpacks or static")
	}
	if len(source.BuildArgs) > maxBuildArgs {
		return fmt.Errorf("at most %d build arguments are allowed", maxBuildArgs)
	}
	for _, arg := range source.BuildArgs {
		key, value, found := strings.Cut(arg, "=")
		if !found || !buildArgKeyPattern.MatchString(key) || len(value) > 4096 || strings.ContainsAny(value, "\x00\r\n") {
			return fmt.Errorf("build argument %q must be KEY=VALUE on a single line", key)
		}
	}
	return nil
}

func (h *Handler) readGitSource(ctx context.Context, serviceID string) (gitSource, error) {
	var source gitSource
	var encryptedArgs []byte
	var providerID sql.NullString
	var installationID sql.NullInt64
	err := h.db.QueryRowContext(ctx, `SELECT provider_id, installation_id, repository_url, git_ref, base_directory, dockerfile_path, build_pack, install_command, build_command, start_command, publish_directory, encrypted_build_args, auto_deploy
		FROM service_git_sources WHERE service_id = ?`, serviceID).Scan(&providerID, &installationID, &source.RepositoryURL, &source.Ref, &source.BaseDirectory, &source.DockerfilePath, &source.BuildPack, &source.InstallCommand, &source.BuildCommand, &source.StartCommand, &source.PublishDirectory, &encryptedArgs, &source.AutoDeploy)
	if err != nil {
		return source, err
	}
	source.ProviderID, source.InstallationID = providerID.String, installationID.Int64
	source.BuildArgs = []string{}
	if encryptedArgs != nil {
		plaintext, err := secrets.Decrypt(encryptedArgs)
		if err != nil {
			return source, err
		}
		if err := json.Unmarshal(plaintext, &source.BuildArgs); err != nil {
			return source, err
		}
	}
	return source, nil
}

func (h *Handler) authorizeGitSourceService(w http.ResponseWriter, r *http.Request) (databaseService, bool) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
		return databaseService{}, false
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return databaseService{}, false
	}
	serviceID, ok := pathID(w, r, "serviceID")
	if !ok {
		return databaseService{}, false
	}
	service, err := h.readService(r, workspaceID, projectID, serviceID, false)
	if err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return databaseService{}, false
	}
	return service, true
}

func (h *Handler) getGitSource(w http.ResponseWriter, r *http.Request) {
	service, ok := h.authorizeGitSourceService(w, r)
	if !ok {
		return
	}
	source, err := h.readGitSource(r.Context(), service.ID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read Git source")
		return
	}
	if err := h.addWebhookDetails(r.Context(), service.ID, &source); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read webhook secret")
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (h *Handler) addWebhookDetails(ctx context.Context, serviceID string, source *gitSource) error {
	var kind sql.NullString
	var encrypted []byte
	if err := h.db.QueryRowContext(ctx, "SELECT p.kind, s.encrypted_webhook_secret FROM service_git_sources s LEFT JOIN git_providers p ON p.id = s.provider_id WHERE s.service_id = ?", serviceID).Scan(&kind, &encrypted); err != nil {
		return err
	}
	if kind.String == "github_app" || encrypted == nil {
		return nil
	}
	secret, err := secrets.Decrypt(encrypted)
	if err != nil {
		return err
	}
	source.WebhookSecret = string(secret)
	source.WebhookURL = h.webhookURL(ctx, serviceID)
	return nil
}

func (h *Handler) updateGitSource(w http.ResponseWriter, r *http.Request) {
	if !gitBuildsEnabled() {
		writeError(w, http.StatusNotFound, "Git builds are disabled")
		return
	}
	service, ok := h.authorizeGitSourceService(w, r)
	if !ok {
		return
	}
	if service.Category != "application" || service.Engine != "docker" || service.Compose {
		writeError(w, http.StatusConflict, "Git builds require a Docker application")
		return
	}
	if service.Image != "docker.io/kymber/build-artifacts" && service.Status != "draft" {
		writeError(w, http.StatusConflict, "only draft apps can be converted to Git builds")
		return
	}
	var source gitSource
	if !decodeJSON(w, r, &source) {
		return
	}
	if !validGitRef(source.Ref) {
		writeError(w, http.StatusBadRequest, "a valid branch or ref is required")
		return
	}
	repository, err := h.openGitRepository(r.Context(), service.WorkspaceID, source)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	source.InstallationID = repository.installationID
	if err := validateGitBuildConfiguration(&source); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if source.BuildArgs == nil {
		source.BuildArgs = []string{}
	}
	source.WebhookURL, source.WebhookSecret = "", ""
	webhookSecret := make([]byte, 24)
	if _, err := rand.Read(webhookSecret); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create webhook secret")
		return
	}
	encryptedWebhookSecret, err := secrets.Encrypt([]byte(hex.EncodeToString(webhookSecret)))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "service encryption is not configured")
		return
	}
	plainArgs, _ := json.Marshal(source.BuildArgs)
	encryptedArgs, err := secrets.Encrypt(plainArgs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "service encryption is not configured")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `INSERT INTO service_git_sources (service_id, provider_id, installation_id, repository_url, git_ref, base_directory, dockerfile_path, build_pack, install_command, build_command, start_command, publish_directory, encrypted_build_args, auto_deploy, encrypted_webhook_secret)
		VALUES (?, NULLIF(?, ''), NULLIF(?, 0), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE encrypted_webhook_secret = COALESCE(encrypted_webhook_secret, VALUES(encrypted_webhook_secret)), provider_id = VALUES(provider_id), installation_id = VALUES(installation_id), repository_url = VALUES(repository_url), git_ref = VALUES(git_ref), base_directory = VALUES(base_directory),
		dockerfile_path = VALUES(dockerfile_path), build_pack = VALUES(build_pack), install_command = VALUES(install_command), build_command = VALUES(build_command), start_command = VALUES(start_command),
		publish_directory = VALUES(publish_directory), encrypted_build_args = VALUES(encrypted_build_args), auto_deploy = VALUES(auto_deploy)`,
		service.ID, source.ProviderID, source.InstallationID, source.RepositoryURL, source.Ref, source.BaseDirectory, source.DockerfilePath, source.BuildPack, source.InstallCommand, source.BuildCommand, source.StartCommand, source.PublishDirectory, encryptedArgs, source.AutoDeploy, encryptedWebhookSecret); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save Git source")
		return
	}
	if err := h.addWebhookDetails(r.Context(), service.ID, &source); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read webhook secret")
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (h *Handler) deployGitBuild(jobID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	var serviceID, workspaceID, projectID string
	if err := h.db.QueryRowContext(ctx, `SELECT s.id, s.workspace_id, s.project_id FROM git_build_jobs b JOIN database_services s ON s.id = b.service_id WHERE b.id = ?`, jobID).Scan(&serviceID, &workspaceID, &projectID); err != nil {
		return
	}
	body, _ := json.Marshal(map[string]string{"image_version": jobID})
	request := httptest.NewRequestWithContext(ctx, http.MethodPost, "/", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	h.runServiceAction(recorder, request, workspaceID, projectID, serviceID, "deploy")
	if recorder.Code >= http.StatusBadRequest {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(recorder.Body.Bytes(), &failure)
		message := "Automatic deploy after the Git build failed: " + strings.TrimSpace(failure.Error)
		_, _ = h.db.ExecContext(ctx, "UPDATE git_build_jobs SET result_message = ? WHERE id = ?", message, jobID)
		_, _ = h.db.ExecContext(ctx, "UPDATE database_services SET last_error = ? WHERE id = ?", message, serviceID)
		h.reportCommitStatus(jobID, "failure", "Deploy failed")
		return
	}
	h.reportCommitStatus(jobID, "success", "Deployed")
}
