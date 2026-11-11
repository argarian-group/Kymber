package services

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kymber/backend/internal/artifacts"
	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
	"kymber/backend/internal/secrets"
)

type gitBuildRequest struct {
	RepositoryURL    string `json:"repository_url"`
	Ref              string `json:"ref"`
	BuildPack        string `json:"build_pack"`
	InstallCommand   string `json:"install_command"`
	BuildCommand     string `json:"build_command"`
	StartCommand     string `json:"start_command"`
	PublishDirectory string `json:"publish_directory"`
}

func (h *Handler) createGitBuildDraft(w http.ResponseWriter, r *http.Request) {
	if !gitBuildsEnabled() {
		writeError(w, http.StatusNotFound, "Git builds are disabled")
		return
	}
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return
	}
	var input struct {
		Name          string `json:"name"`
		EnvironmentID string `json:"environment_id"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name = strings.ToLower(strings.TrimSpace(input.Name))
	if !serviceNamePattern.MatchString(input.Name) {
		writeError(w, http.StatusBadRequest, "valid app name is required")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create app")
		return
	}
	image := "docker.io/kymber/build-artifacts"
	var environmentName string
	if input.EnvironmentID != "" {
		if err := h.db.QueryRowContext(r.Context(), "SELECT name FROM project_environments WHERE id = ? AND project_id = ?", input.EnvironmentID, projectID).Scan(&environmentName); err != nil {
			writeError(w, http.StatusBadRequest, "environment not found")
			return
		}
	}
	config, err := secrets.Encrypt([]byte("{}"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "service encryption is not configured")
		return
	}
	_, err = h.db.ExecContext(r.Context(), `INSERT INTO database_services (id, workspace_id, project_id, engine, name, image_version, image, port, data_path, category, environment_id, database_name, database_user, encrypted_config, created_by) VALUES (?, ?, ?, 'docker', ?, 'draft', ?, 0, '', 'application', ?, '', '', ?, ?)`, id, workspaceID, projectID, input.Name, image, input.EnvironmentID, config, user.ID)
	if err != nil {
		writeError(w, http.StatusConflict, "could not create draft app; its name may already be used")
		return
	}
	writeJSON(w, http.StatusCreated, databaseService{ID: id, WorkspaceID: workspaceID, ProjectID: projectID, Engine: "docker", Name: input.Name, Image: image, ImageVersion: "draft", Status: "draft", Category: "application", EnvironmentID: input.EnvironmentID, EnvironmentName: environmentName})
}

type gitBuildJob struct {
	ID                string `json:"id"`
	ServiceID         string `json:"service_id"`
	RepositoryURL     string `json:"repository_url"`
	Ref               string `json:"ref"`
	CommitSHA         string `json:"commit_sha,omitempty"`
	Status            string `json:"status"`
	ArtifactDigest    string `json:"artifact_digest,omitempty"`
	ResultMessage     string `json:"result_message,omitempty"`
	CreatedAt         string `json:"created_at"`
	StartedAt         string `json:"started_at,omitempty"`
	FinishedAt        string `json:"finished_at,omitempty"`
	ArtifactExpiresAt string `json:"artifact_expires_at,omitempty"`
}

type gitHubRef struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

func (h *Handler) listGitHubBranches(w http.ResponseWriter, r *http.Request) {
	if !gitBuildsEnabled() {
		writeError(w, http.StatusNotFound, "Git builds are disabled")
		return
	}
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return
	}
	repositoryURL := r.URL.Query().Get("repository_url")
	if !validPublicGitHubURL(repositoryURL) {
		writeError(w, http.StatusBadRequest, "a public GitHub HTTPS repository is required")
		return
	}
	repositoryPath := strings.TrimSuffix(strings.TrimPrefix(repositoryURL, "https://github.com/"), ".git")
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://api.github.com/repos/"+repositoryPath+"/branches?per_page=100", nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not request GitHub branches")
		return
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "Kymber-build-controller")
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not reach GitHub")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusNotFound {
			writeError(w, http.StatusBadRequest, "GitHub repository was not found or is private")
			return
		}
		writeError(w, http.StatusBadGateway, fmt.Sprintf("GitHub returned HTTP %d while listing branches", response.StatusCode))
		return
	}
	var branches []gitHubRef
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&branches); err != nil {
		writeError(w, http.StatusBadGateway, "GitHub returned an invalid branches response")
		return
	}
	if len(branches) > 100 {
		branches = branches[:100]
	}
	writeJSON(w, http.StatusOK, map[string][]gitHubRef{"branches": branches})
}

type gitBuildClaim struct {
	ID            string `json:"id"`
	RepositoryURL string `json:"repository_url"`
	Ref           string `json:"ref"`
	ServiceID     string `json:"service_id"`
	ServerID      string `json:"server_id"`
	Attempt       int    `json:"attempt"`
}

var gitRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$`)
var gitHubOwnerPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}$`)
var gitHubRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
var gitCommitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

const maxBuildArtifactStorage int64 = 10 << 30

func gitBuildsEnabled() bool { return os.Getenv("KYMBER_GIT_BUILDS_ENABLED") == "true" }

func validPublicGitHubURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.Port() != "" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	parts := strings.Split(strings.TrimSuffix(u.Path, ".git"), "/")
	return len(parts) == 3 && parts[0] == "" && gitHubOwnerPattern.MatchString(parts[1]) && gitHubRepositoryPattern.MatchString(parts[2])
}

func validateGitBuildConfiguration(config gitBuildRequest) error {
	if config.BuildPack == "" {
		config.BuildPack = "dockerfile"
	}
	if config.BuildPack != "dockerfile" || config.InstallCommand != "" || config.BuildCommand != "" || config.StartCommand != "" || config.PublishDirectory != "" {
		return errors.New("only Dockerfile builds from the repository root are supported")
	}
	return nil
}

func validGitRef(ref string) bool {
	if !gitRefPattern.MatchString(ref) || strings.Contains(ref, "..") || strings.Contains(ref, "//") || strings.Contains(ref, "@{") {
		return false
	}
	for _, part := range strings.Split(ref, "/") {
		if strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".") || strings.HasSuffix(part, ".lock") {
			return false
		}
	}
	return true
}

func (h *Handler) claimGitBuild(w http.ResponseWriter, r *http.Request) {
	if !gitBuildsEnabled() {
		writeError(w, http.StatusNotFound, "Git builds are disabled")
		return
	}
	if !h.authenticateBuildWorker(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not claim build job")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(r.Context(), `UPDATE git_build_jobs
		SET status = 'failed', result_message = 'Build worker lease expired too many times.', finished_at = UTC_TIMESTAMP(6), lease_expires_at = NULL
		WHERE status = 'running' AND lease_expires_at < UTC_TIMESTAMP(6) AND attempts >= 5`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not recover expired build jobs")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE database_services s JOIN git_build_jobs b ON b.service_id = s.id
		SET s.status = 'failed', s.last_error = b.result_message WHERE b.status = 'failed' AND b.attempts >= 5
		AND s.status = 'queued' AND s.image_version = b.id`); err != nil {
		writeError(w, http.StatusInternalServerError, "could not mark expired builds failed")
		return
	}
	var job gitBuildClaim
	job.ServerID = r.Header.Get("X-Build-Server-ID")
	if !entityIDPattern.MatchString(job.ServerID) {
		writeError(w, http.StatusBadRequest, "valid build server ID is required")
		return
	}
	var connectionType string
	if err := tx.QueryRowContext(r.Context(), "SELECT connection_type FROM servers WHERE id = ?", job.ServerID).Scan(&connectionType); err != nil || connectionType != "local" {
		_ = tx.Rollback()
		writeError(w, http.StatusBadRequest, "build controller must run on the selected local deployment host")
		return
	}
	err = tx.QueryRowContext(r.Context(), `SELECT b.id, b.repository_url, b.git_ref, b.service_id, s.server_id, b.attempts
		FROM git_build_jobs b JOIN database_services s ON s.id = b.service_id
		WHERE (b.status = 'queued' OR (b.status = 'running' AND b.lease_expires_at < UTC_TIMESTAMP(6) AND b.attempts < 5))
		AND s.status IN ('queued', 'failed', 'draft') AND s.server_id = ?
		ORDER BY b.created_at LIMIT 1 FOR UPDATE SKIP LOCKED`, job.ServerID).Scan(&job.ID, &job.RepositoryURL, &job.Ref, &job.ServiceID, &job.ServerID, &job.Attempt)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not recover expired build jobs")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read build queue")
		return
	}
	job.Attempt++
	if _, err := tx.ExecContext(r.Context(), `UPDATE git_build_jobs SET status = 'running', attempts = ?, lease_expires_at = UTC_TIMESTAMP(6) + INTERVAL 15 MINUTE, started_at = COALESCE(started_at, UTC_TIMESTAMP(6)), result_message = NULL WHERE id = ?`, job.Attempt, job.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not claim build job")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not claim build job")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(job)
}

func (h *Handler) authenticateBuildWorker(r *http.Request) bool {
	token := os.Getenv("KYMBER_BUILD_WORKER_TOKEN")
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	authorization := strings.TrimPrefix(header, "Bearer ")
	return token != "" && len(token) == len(authorization) && subtle.ConstantTimeCompare([]byte(token), []byte(authorization)) == 1
}

func (h *Handler) buildWorkerServer(w http.ResponseWriter, r *http.Request) {
	if !h.authenticateBuildWorker(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var id string
	if err := h.db.QueryRowContext(r.Context(), "SELECT id FROM servers s WHERE s.connection_type = 'local' AND "+serverStatusSQL+" = 'online' ORDER BY s.created_at LIMIT 1").Scan(&id); err != nil {
		writeError(w, http.StatusNotFound, "no online local server is available")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"server_id": id})
}

func (h *Handler) heartbeatGitBuild(w http.ResponseWriter, r *http.Request) {
	if !gitBuildsEnabled() {
		writeError(w, http.StatusNotFound, "Git builds are disabled")
		return
	}
	if !h.authenticateBuildWorker(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	jobID := r.PathValue("jobID")
	attempt, ok := buildAttempt(w, r)
	if !ok || !entityIDPattern.MatchString(jobID) {
		if ok {
			writeError(w, http.StatusBadRequest, "invalid job id")
		}
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not renew build lease")
		return
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(r.Context(), `UPDATE git_build_jobs SET lease_expires_at = UTC_TIMESTAMP(6) + INTERVAL 15 MINUTE
		WHERE id = ? AND status = 'running' AND attempts = ? AND lease_expires_at > UTC_TIMESTAMP(6)`, jobID, attempt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not renew build lease")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		writeError(w, http.StatusConflict, "build lease is no longer owned")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not renew build lease")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) failGitBuild(w http.ResponseWriter, r *http.Request) {
	if !gitBuildsEnabled() {
		writeError(w, http.StatusNotFound, "Git builds are disabled")
		return
	}
	if !h.authenticateBuildWorker(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	jobID := r.PathValue("jobID")
	attempt, ok := buildAttempt(w, r)
	if !ok || !entityIDPattern.MatchString(jobID) {
		if ok {
			writeError(w, http.StatusBadRequest, "invalid job id")
		}
		return
	}
	var input struct {
		Message string `json:"message"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Message == "" || len(input.Message) > 2048 {
		writeError(w, http.StatusBadRequest, "failure message must be between 1 and 2048 bytes")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not record build failure")
		return
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(r.Context(), `UPDATE git_build_jobs SET status = 'failed', result_message = ?, finished_at = UTC_TIMESTAMP(6), lease_expires_at = NULL
		WHERE id = ? AND status = 'running' AND attempts = ? AND lease_expires_at > UTC_TIMESTAMP(6)`, input.Message, jobID, attempt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not record build failure")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		writeError(w, http.StatusConflict, "build lease is no longer owned")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE database_services SET status = 'failed', last_error = ?
		WHERE id = (SELECT service_id FROM git_build_jobs WHERE id = ?) AND status = 'queued' AND image_version = ?`, input.Message, jobID, jobID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update failed application")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record build failure")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) storeGitBuildArtifact(w http.ResponseWriter, r *http.Request) {
	if !gitBuildsEnabled() {
		writeError(w, http.StatusNotFound, "Git builds are disabled")
		return
	}
	if !h.authenticateBuildWorker(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	jobID := r.PathValue("jobID")
	attempt, ok := buildAttempt(w, r)
	if !ok || !entityIDPattern.MatchString(jobID) {
		if ok {
			writeError(w, http.StatusBadRequest, "invalid job id")
		}
		return
	}
	commitSHA := r.Header.Get("X-Git-Commit")
	if !gitCommitPattern.MatchString(commitSHA) {
		writeError(w, http.StatusBadRequest, "invalid Git commit SHA")
		return
	}
	var owned bool
	if err := h.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM git_build_jobs
		WHERE id = ? AND status = 'running' AND attempts = ? AND lease_expires_at > UTC_TIMESTAMP(6))`, jobID, attempt).Scan(&owned); err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify build lease")
		return
	}
	if !owned {
		writeError(w, http.StatusConflict, "build lease is no longer owned")
		return
	}
	var repositoryURL, gitRef string
	if err := h.db.QueryRowContext(r.Context(), "SELECT repository_url, git_ref FROM git_build_jobs WHERE id = ? AND status = 'running' AND attempts = ? AND lease_expires_at > UTC_TIMESTAMP(6)", jobID, attempt).Scan(&repositoryURL, &gitRef); err != nil {
		writeError(w, http.StatusConflict, "build lease is no longer owned")
		return
	}
	if !allowArtifactTransfer(w) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, artifacts.MaxOCIArchiveSize)
	archive, digest, err := artifacts.SpoolOCI(r.Body)
	if err != nil {
		status := http.StatusBadRequest
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) || strings.Contains(err.Error(), "exceeds 1 GiB") {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, err.Error())
		return
	}
	defer os.Remove(archive.Name())
	defer archive.Close()
	if !validPublicGitHubURL(repositoryURL) || !validGitRef(gitRef) {
		writeError(w, http.StatusConflict, "build job source is invalid")
		return
	}
	repositoryPath := strings.TrimSuffix(strings.TrimPrefix(repositoryURL, "https://github.com/"), ".git")
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, "https://api.github.com/repos/"+repositoryPath+"/commits/"+url.PathEscape(gitRef), nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not verify source commit")
		return
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "Kymber-build-controller")
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not verify source commit")
		return
	}
	var githubCommit struct {
		SHA string `json:"sha"`
	}
	decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&githubCommit)
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("GitHub returned HTTP %d while verifying the requested ref", response.StatusCode))
		return
	}
	if decodeErr != nil {
		writeError(w, http.StatusBadGateway, "GitHub returned an invalid ref response")
		return
	}
	if githubCommit.SHA != commitSHA {
		writeError(w, http.StatusConflict, "reported commit does not match the requested Git ref")
		return
	}
	converted, _, err := artifacts.ConvertOCIToDockerArchive(archive, "docker.io/kymber/build-artifacts:"+jobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "OCI artifact must contain exactly one valid image: "+err.Error())
		return
	}
	converted.Close()
	os.Remove(converted.Name())
	directory := os.Getenv("KYMBER_BUILD_ARTIFACT_DIR")
	if directory == "" {
		writeError(w, http.StatusInternalServerError, "build artifact storage is not configured")
		return
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create artifact storage")
		return
	}
	stored, err := os.CreateTemp(directory, fmt.Sprintf("%s-%d-*.oci", jobID, attempt))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not store build artifact")
		return
	}
	storedPath := stored.Name()
	keep := false
	defer func() {
		stored.Close()
		if !keep {
			os.Remove(storedPath)
		}
	}()
	if _, err := archive.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read validated artifact")
		return
	}
	if _, err := io.Copy(stored, archive); err != nil {
		writeError(w, http.StatusInternalServerError, "could not persist artifact")
		return
	}
	if err := stored.Sync(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not persist artifact")
		return
	}
	if err := stored.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not persist artifact")
		return
	}
	info, err := os.Stat(storedPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check artifact storage capacity")
		return
	}
	if info.Size() > artifacts.MaxOCIArchiveSize {
		writeError(w, http.StatusRequestEntityTooLarge, "OCI artifact exceeds 1 GiB")
		return
	}
	if err := h.expireGitBuildArtifacts(r.Context(), directory); err != nil {
		writeError(w, http.StatusInternalServerError, "could not expire old build artifacts")
		return
	}
	capacityTx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check artifact storage capacity")
		return
	}
	defer capacityTx.Rollback()
	if _, err := capacityTx.ExecContext(r.Context(), "UPDATE git_build_jobs SET status = status WHERE id = ?", jobID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not reserve artifact storage")
		return
	}
	var lockedJobID string
	if err := capacityTx.QueryRowContext(r.Context(), "SELECT id FROM git_build_jobs WHERE id = ? AND status = 'running' AND attempts = ? AND lease_expires_at > UTC_TIMESTAMP(6) FOR UPDATE", jobID, attempt).Scan(&lockedJobID); err != nil {
		writeError(w, http.StatusConflict, "build lease is no longer owned")
		return
	}
	var artifactBytes int64
	rows, err := capacityTx.QueryContext(r.Context(), "SELECT artifact_size FROM git_build_jobs WHERE artifact_path IS NOT NULL AND artifact_expires_at > UTC_TIMESTAMP(6) AND id <> ? FOR UPDATE", jobID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check artifact storage capacity")
		return
	}
	for rows.Next() {
		var size int64
		if err := rows.Scan(&size); err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "could not check artifact storage capacity")
			return
		}
		if artifactBytes > math.MaxInt64-size {
			rows.Close()
			writeError(w, http.StatusInsufficientStorage, "build artifact storage accounting overflow")
			return
		}
		artifactBytes += size
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		writeError(w, http.StatusInternalServerError, "could not check build artifact storage capacity")
		return
	}
	if err := rows.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check artifact storage capacity")
		return
	}
	if info.Size() > maxBuildArtifactStorage || artifactBytes > maxBuildArtifactStorage-info.Size() {
		writeError(w, http.StatusInsufficientStorage, "build artifact storage limit reached")
		return
	}
	finalPath := storedPath + ".tar"
	if err := os.Rename(storedPath, finalPath); err != nil {
		writeError(w, http.StatusInternalServerError, "could not publish artifact")
		return
	}
	result, err := capacityTx.ExecContext(r.Context(), `UPDATE git_build_jobs SET status = 'succeeded', commit_sha = ?, artifact_digest = ?, artifact_path = ?, artifact_size = ?, artifact_expires_at = UTC_TIMESTAMP(6) + INTERVAL 30 DAY, result_message = NULL,
		finished_at = UTC_TIMESTAMP(6), lease_expires_at = NULL WHERE id = ? AND status = 'running' AND attempts = ? AND lease_expires_at > UTC_TIMESTAMP(6)`, commitSHA, digest, filepath.Base(finalPath), info.Size(), jobID, attempt)
	if err != nil {
		os.Remove(finalPath)
		writeError(w, http.StatusInternalServerError, "could not complete build job")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		os.Remove(finalPath)
		writeError(w, http.StatusInternalServerError, "could not verify build completion")
		return
	}
	if changed != 1 {
		os.Remove(finalPath)
		writeError(w, http.StatusConflict, "build lease is no longer owned")
		return
	}
	serviceResult, err := capacityTx.ExecContext(r.Context(), `UPDATE database_services SET image = 'docker.io/kymber/build-artifacts', image_version = ?, status = 'draft', last_error = NULL
		WHERE id = (SELECT service_id FROM git_build_jobs WHERE id = ?) AND image_version = ? AND status = 'queued'`, jobID, jobID, jobID)
	if err != nil {
		os.Remove(finalPath)
		writeError(w, http.StatusInternalServerError, "could not update built application")
		return
	}
	serviceChanged, err := serviceResult.RowsAffected()
	if err != nil || serviceChanged != 1 {
		os.Remove(finalPath)
		writeError(w, http.StatusConflict, "application changed while the build was running")
		return
	}
	if err := capacityTx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not commit build completion")
		return
	}
	keep = true
	w.WriteHeader(http.StatusNoContent)
}

func buildAttempt(w http.ResponseWriter, r *http.Request) (int, bool) {
	attempt, err := strconv.Atoi(r.Header.Get("X-Build-Attempt"))
	if err != nil || attempt < 1 || attempt > 5 {
		writeError(w, http.StatusBadRequest, "invalid build attempt")
		return 0, false
	}
	return attempt, true
}

func (h *Handler) expireGitBuildArtifacts(ctx context.Context, directory string) error {
	rows, err := h.db.QueryContext(ctx, `SELECT id, artifact_path FROM git_build_jobs WHERE artifact_path IS NOT NULL AND artifact_expires_at <= UTC_TIMESTAMP(6) ORDER BY artifact_expires_at LIMIT 100`)
	if err != nil {
		return err
	}
	type artifact struct{ id, name string }
	var expired []artifact
	for rows.Next() {
		var item artifact
		if err := rows.Scan(&item.id, &item.name); err != nil {
			rows.Close()
			return err
		}
		expired = append(expired, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, item := range expired {
		if filepath.Base(item.name) != item.name || item.name == "." {
			return fmt.Errorf("invalid stored build artifact path")
		}
	}
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range expired {
		if _, err := tx.ExecContext(ctx, `UPDATE git_build_jobs SET artifact_path = NULL, artifact_size = NULL, artifact_expires_at = NULL WHERE id = ? AND artifact_path = ? AND artifact_expires_at <= UTC_TIMESTAMP(6)`, item.id, item.name); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, item := range expired {
		if err := os.Remove(filepath.Join(directory, item.name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (h *Handler) createGitBuild(w http.ResponseWriter, r *http.Request) {
	if !gitBuildsEnabled() {
		writeError(w, http.StatusNotFound, "Git builds are disabled")
		return
	}
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return
	}
	serviceID, ok := pathID(w, r, "serviceID")
	if !ok {
		return
	}
	service, err := h.readService(r, workspaceID, projectID, serviceID, false)
	if err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if service.Category != "application" || service.Engine != "docker" || service.Compose {
		writeError(w, http.StatusConflict, "Git builds require a Docker application")
		return
	}
	if service.Status != "draft" && service.Status != "failed" {
		writeError(w, http.StatusConflict, "Git builds can only be queued for draft or failed apps")
		return
	}
	var input gitBuildRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.BuildPack == "" {
		input.BuildPack = "dockerfile"
	}
	if !validPublicGitHubURL(input.RepositoryURL) || !validGitRef(input.Ref) {
		writeError(w, http.StatusBadRequest, "a public GitHub HTTPS repository and valid ref are required")
		return
	}
	if service.Image != "docker.io/kymber/build-artifacts" && service.Status != "draft" {
		writeError(w, http.StatusConflict, "only draft apps can be converted to GitHub builds")
		return
	}
	if err := validateGitBuildConfiguration(input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create build job")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not queue build job")
		return
	}
	defer tx.Rollback()
	serviceResult, err := tx.ExecContext(r.Context(), `UPDATE database_services SET image='docker.io/kymber/build-artifacts', image_version=?, status='queued', last_error=NULL
		WHERE id=? AND status NOT IN ('queued', 'deploying') AND (image='docker.io/kymber/build-artifacts' OR status='draft')`, id, serviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not prepare application for build")
		return
	}
	if changed, err := serviceResult.RowsAffected(); err != nil || changed != 1 {
		writeError(w, http.StatusConflict, "wait for the current app action to finish")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO git_build_jobs (id, service_id, repository_url, git_ref) VALUES (?, ?, ?, ?)", id, serviceID, input.RepositoryURL, input.Ref); err != nil {
		writeError(w, http.StatusInternalServerError, "could not queue build job")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not queue build job")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "status": "queued"})
}

func (h *Handler) listGitBuilds(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, "") {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return
	}
	serviceID, ok := pathID(w, r, "serviceID")
	if !ok {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `SELECT b.id, b.service_id, b.repository_url, b.git_ref, COALESCE(b.commit_sha, ''), b.status, COALESCE(b.artifact_digest, ''), COALESCE(b.result_message, ''), b.created_at, b.started_at, b.finished_at, b.artifact_expires_at
		FROM git_build_jobs b JOIN database_services s ON s.id = b.service_id WHERE b.service_id = ? AND s.workspace_id = ? AND s.project_id = ? ORDER BY b.created_at DESC LIMIT 100`, serviceID, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list build jobs")
		return
	}
	defer rows.Close()
	jobs := make([]gitBuildJob, 0)
	for rows.Next() {
		var job gitBuildJob
		var startedAt, finishedAt, artifactExpiresAt *string
		if err := rows.Scan(&job.ID, &job.ServiceID, &job.RepositoryURL, &job.Ref, &job.CommitSHA, &job.Status, &job.ArtifactDigest, &job.ResultMessage, &job.CreatedAt, &startedAt, &finishedAt, &artifactExpiresAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read build jobs")
			return
		}
		if startedAt != nil {
			job.StartedAt = *startedAt
		}
		if finishedAt != nil {
			job.FinishedAt = *finishedAt
		}
		if artifactExpiresAt != nil {
			job.ArtifactExpiresAt = *artifactExpiresAt
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read build jobs")
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}
