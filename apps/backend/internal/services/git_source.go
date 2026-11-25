package services

import (
	"context"
	"database/sql"
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
	if source.BuildPack != "dockerfile" || source.InstallCommand != "" || source.BuildCommand != "" || source.StartCommand != "" || source.PublishDirectory != "" {
		return errors.New("only Dockerfile builds are supported")
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
	err := h.db.QueryRowContext(ctx, `SELECT repository_url, git_ref, base_directory, dockerfile_path, build_pack, install_command, build_command, start_command, publish_directory, encrypted_build_args, auto_deploy
		FROM service_git_sources WHERE service_id = ?`, serviceID).Scan(&source.RepositoryURL, &source.Ref, &source.BaseDirectory, &source.DockerfilePath, &source.BuildPack, &source.InstallCommand, &source.BuildCommand, &source.StartCommand, &source.PublishDirectory, &encryptedArgs, &source.AutoDeploy)
	if err != nil {
		return source, err
	}
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
	writeJSON(w, http.StatusOK, source)
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
	if !validPublicGitHubURL(source.RepositoryURL) || !validGitRef(source.Ref) {
		writeError(w, http.StatusBadRequest, "a public GitHub HTTPS repository and valid ref are required")
		return
	}
	if err := validateGitBuildConfiguration(&source); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if source.BuildArgs == nil {
		source.BuildArgs = []string{}
	}
	plainArgs, _ := json.Marshal(source.BuildArgs)
	encryptedArgs, err := secrets.Encrypt(plainArgs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "service encryption is not configured")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), `INSERT INTO service_git_sources (service_id, repository_url, git_ref, base_directory, dockerfile_path, build_pack, install_command, build_command, start_command, publish_directory, encrypted_build_args, auto_deploy)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE repository_url = VALUES(repository_url), git_ref = VALUES(git_ref), base_directory = VALUES(base_directory),
		dockerfile_path = VALUES(dockerfile_path), build_pack = VALUES(build_pack), install_command = VALUES(install_command), build_command = VALUES(build_command), start_command = VALUES(start_command),
		publish_directory = VALUES(publish_directory), encrypted_build_args = VALUES(encrypted_build_args), auto_deploy = VALUES(auto_deploy)`,
		service.ID, source.RepositoryURL, source.Ref, source.BaseDirectory, source.DockerfilePath, source.BuildPack, source.InstallCommand, source.BuildCommand, source.StartCommand, source.PublishDirectory, encryptedArgs, source.AutoDeploy); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save Git source")
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
	}
}
