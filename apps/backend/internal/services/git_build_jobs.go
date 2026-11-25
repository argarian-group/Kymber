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

	"kymber/backend/internal/artifacts"
	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
	"kymber/backend/internal/secrets"
)

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
	Trigger           string `json:"trigger"`
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
	query := r.URL.Query()
	installationID, _ := strconv.ParseInt(query.Get("installation_id"), 10, 64)
	repository, err := h.openGitRepository(r.Context(), workspaceID, gitSource{ProviderID: query.Get("provider_id"), InstallationID: installationID, RepositoryURL: query.Get("repository_url")})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	branches, err := repository.branches(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string][]gitBranch{"branches": branches})
}

type gitBuildClaim struct {
	ID             string   `json:"id"`
	RepositoryURL  string   `json:"repository_url"`
	Ref            string   `json:"ref"`
	ServiceID      string   `json:"service_id"`
	ServerID       string   `json:"server_id"`
	Attempt        int      `json:"attempt"`
	BaseDirectory  string   `json:"base_directory"`
	DockerfilePath string   `json:"dockerfile_path"`
	BuildPack      string   `json:"build_pack"`
	BuildArgs      []string `json:"build_args"`
	Commands       struct {
		Install string `json:"install"`
		Build   string `json:"build"`
		Start   string `json:"start"`
	} `json:"commands"`
	PublishDirectory string       `json:"publish_directory"`
	Commit           string       `json:"commit,omitempty"`
	Auth             gitCloneAuth `json:"auth"`
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
	var encryptedConfig []byte
	var workspaceID string
	var commit sql.NullString
	err = tx.QueryRowContext(r.Context(), `SELECT b.id, b.repository_url, b.git_ref, b.service_id, b.attempts, b.encrypted_build_config, s.workspace_id, b.commit_sha
		FROM git_build_jobs b JOIN database_services s ON s.id = b.service_id
		WHERE (b.status = 'queued' OR (b.status = 'running' AND b.lease_expires_at < UTC_TIMESTAMP(6) AND b.attempts < 5))
		AND (s.status IN ('queued', 'failed', 'draft') OR (b.deploy_on_success AND s.status <> 'deleted')) AND (s.server_id = ? OR s.server_id IS NULL)
		AND NOT EXISTS (SELECT 1 FROM git_build_jobs o WHERE o.service_id = b.service_id AND o.id <> b.id AND o.status = 'running' AND o.lease_expires_at > UTC_TIMESTAMP(6))
		ORDER BY b.created_at LIMIT 1 FOR UPDATE SKIP LOCKED`, job.ServerID).Scan(&job.ID, &job.RepositoryURL, &job.Ref, &job.ServiceID, &job.Attempt, &encryptedConfig, &workspaceID, &commit)
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
	source := gitSource{DockerfilePath: "Dockerfile", BuildPack: "dockerfile"}
	if encryptedConfig != nil {
		plaintext, err := secrets.Decrypt(encryptedConfig)
		if err == nil {
			err = json.Unmarshal(plaintext, &source)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not read build configuration")
			return
		}
	}
	job.BaseDirectory, job.DockerfilePath, job.BuildPack, job.BuildArgs = source.BaseDirectory, source.DockerfilePath, source.BuildPack, source.BuildArgs
	job.Commands.Install, job.Commands.Build, job.Commands.Start, job.PublishDirectory = source.InstallCommand, source.BuildCommand, source.StartCommand, source.PublishDirectory
	if job.BuildArgs == nil {
		job.BuildArgs = []string{}
	}
	job.Commit = commit.String
	repository, err := h.openGitRepository(r.Context(), workspaceID, source)
	if err == nil {
		job.Auth, err = repository.cloneAuth(r.Context())
	}
	job.Attempt++
	if err != nil {
		message := "Could not prepare repository access: " + err.Error()
		if _, err := tx.ExecContext(r.Context(), `UPDATE git_build_jobs SET status = 'failed', attempts = ?, result_message = ?, finished_at = UTC_TIMESTAMP(6) WHERE id = ?`, job.Attempt, message, job.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not fail build job")
			return
		}
		if _, err := tx.ExecContext(r.Context(), `UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ? AND status = 'queued' AND image_version = ?`, message, job.ServiceID, job.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not fail build job")
			return
		}
		if err := tx.Commit(); err != nil {
			writeError(w, http.StatusInternalServerError, "could not fail build job")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if _, err := tx.ExecContext(r.Context(), `UPDATE git_build_jobs SET status = 'running', attempts = ?, lease_expires_at = UTC_TIMESTAMP(6) + INTERVAL 15 MINUTE, started_at = COALESCE(started_at, UTC_TIMESTAMP(6)), result_message = NULL WHERE id = ?`, job.Attempt, job.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not claim build job")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not claim build job")
		return
	}
	h.reportCommitStatus(job.ID, "running", "Building")
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
	h.reportCommitStatus(jobID, "failure", "Build failed")
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
	var pinnedCommit sql.NullString
	var deployOnSuccess bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT commit_sha, deploy_on_success FROM git_build_jobs WHERE id = ? AND status = 'running' AND attempts = ? AND lease_expires_at > UTC_TIMESTAMP(6)", jobID, attempt).Scan(&pinnedCommit, &deployOnSuccess); err != nil {
		writeError(w, http.StatusConflict, "build lease is no longer owned")
		return
	}
	if pinnedCommit.Valid && pinnedCommit.String != commitSHA {
		writeError(w, http.StatusConflict, "reported commit does not match the queued commit")
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
	if !deployOnSuccess {
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
	}
	if err := capacityTx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not commit build completion")
		return
	}
	keep = true
	if deployOnSuccess {
		h.reportCommitStatus(jobID, "running", "Built, deploying")
		go h.deployGitBuild(jobID)
	} else {
		h.reportCommitStatus(jobID, "success", "Built, ready to deploy")
	}
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
	id, status, err := h.queueGitBuild(r.Context(), service, "manual", "")
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "status": "queued"})
}

func (h *Handler) rebuildGitCommit(w http.ResponseWriter, r *http.Request) {
	if !gitBuildsEnabled() {
		writeError(w, http.StatusNotFound, "Git builds are disabled")
		return
	}
	service, ok := h.authorizeGitSourceService(w, r)
	if !ok {
		return
	}
	jobID, ok := pathID(w, r, "jobID")
	if !ok {
		return
	}
	var commit sql.NullString
	if err := h.db.QueryRowContext(r.Context(), "SELECT commit_sha FROM git_build_jobs WHERE id = ? AND service_id = ?", jobID, service.ID).Scan(&commit); err != nil {
		writeError(w, http.StatusNotFound, "build not found")
		return
	}
	if !commit.Valid {
		writeError(w, http.StatusConflict, "this build has no recorded commit")
		return
	}
	id, status, err := h.queueGitBuild(r.Context(), service, "rollback", commit.String)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id, "status": "queued"})
}

func (h *Handler) queueGitBuild(ctx context.Context, service databaseService, trigger, commit string) (string, int, error) {
	source, err := h.readGitSource(ctx, service.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", http.StatusConflict, errors.New("configure the app's Git source first")
	}
	if err != nil {
		return "", http.StatusInternalServerError, errors.New("could not read Git source")
	}
	repository, err := h.openGitRepository(ctx, service.WorkspaceID, source)
	if err != nil {
		return "", http.StatusConflict, err
	}
	if commit == "" {
		if commit, err = repository.resolveCommit(ctx, source.Ref); err != nil {
			return "", http.StatusBadGateway, err
		}
	}
	config, err := json.Marshal(source)
	if err != nil {
		return "", http.StatusInternalServerError, errors.New("could not snapshot Git source")
	}
	encryptedConfig, err := secrets.Encrypt(config)
	if err != nil {
		return "", http.StatusInternalServerError, errors.New("service encryption is not configured")
	}
	id, err := database.NewShortID()
	if err != nil {
		return "", http.StatusInternalServerError, errors.New("could not create build job")
	}
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return "", http.StatusInternalServerError, errors.New("could not queue build job")
	}
	defer tx.Rollback()
	var status, image string
	var serverID sql.NullString
	if err := tx.QueryRowContext(ctx, "SELECT status, image, server_id FROM database_services WHERE id = ? FOR UPDATE", service.ID).Scan(&status, &image, &serverID); err != nil {
		return "", http.StatusNotFound, errors.New("service not found")
	}
	if status == "queued" || status == "deploying" || status == "deleted" {
		return "", http.StatusConflict, errors.New("wait for the current app action to finish")
	}
	if image != "docker.io/kymber/build-artifacts" && status != "draft" {
		return "", http.StatusConflict, errors.New("only draft apps can be converted to Git builds")
	}
	activeStatuses := "'queued', 'running'"
	if trigger == "push" {
		if _, err := tx.ExecContext(ctx, "UPDATE git_build_jobs SET status = 'cancelled', result_message = 'Superseded by a newer push.', finished_at = UTC_TIMESTAMP(6) WHERE service_id = ? AND status = 'queued' AND deploy_on_success", service.ID); err != nil {
			return "", http.StatusInternalServerError, errors.New("could not replace queued builds")
		}
		activeStatuses = "'queued'"
	}
	var active bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM git_build_jobs WHERE service_id = ? AND status IN ("+activeStatuses+"))", service.ID).Scan(&active); err != nil {
		return "", http.StatusInternalServerError, errors.New("could not check running builds")
	}
	if active {
		return "", http.StatusConflict, errors.New("a build is already queued or running for this app")
	}
	deployOnSuccess := serverID.Valid && status != "draft"
	if !deployOnSuccess {
		if _, err := tx.ExecContext(ctx, `UPDATE database_services SET image='docker.io/kymber/build-artifacts', image_version=?, status='queued', last_error=NULL WHERE id=?`, id, service.ID); err != nil {
			return "", http.StatusInternalServerError, errors.New("could not prepare application for build")
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO git_build_jobs (id, service_id, repository_url, git_ref, commit_sha, encrypted_build_config, build_trigger, deploy_on_success) VALUES (?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?)`,
		id, service.ID, source.RepositoryURL, source.Ref, commit, encryptedConfig, trigger, deployOnSuccess); err != nil {
		return "", http.StatusInternalServerError, errors.New("could not queue build job")
	}
	if err := tx.Commit(); err != nil {
		return "", http.StatusInternalServerError, errors.New("could not queue build job")
	}
	h.reportCommitStatus(id, "pending", "Queued in Kymber")
	return id, http.StatusAccepted, nil
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
	rows, err := h.db.QueryContext(r.Context(), `SELECT b.id, b.service_id, b.repository_url, b.git_ref, COALESCE(b.commit_sha, ''), b.status, COALESCE(b.artifact_digest, ''), COALESCE(b.result_message, ''), b.created_at, b.started_at, b.finished_at, b.artifact_expires_at, b.build_trigger
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
		if err := rows.Scan(&job.ID, &job.ServiceID, &job.RepositoryURL, &job.Ref, &job.CommitSHA, &job.Status, &job.ArtifactDigest, &job.ResultMessage, &job.CreatedAt, &startedAt, &finishedAt, &artifactExpiresAt, &job.Trigger); err != nil {
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
