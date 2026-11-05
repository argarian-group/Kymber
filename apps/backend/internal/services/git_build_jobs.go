package services

import (
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
)

type gitBuildRequest struct {
	RepositoryURL string `json:"repository_url"`
	Ref           string `json:"ref"`
}

type gitBuildJob struct {
	ID             string `json:"id"`
	ServiceID      string `json:"service_id"`
	RepositoryURL  string `json:"repository_url"`
	Ref            string `json:"ref"`
	CommitSHA      string `json:"commit_sha,omitempty"`
	Status         string `json:"status"`
	ArtifactDigest string `json:"artifact_digest,omitempty"`
	ResultMessage  string `json:"result_message,omitempty"`
	CreatedAt      string `json:"created_at"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
}

type gitBuildClaim struct {
	ID            string `json:"id"`
	RepositoryURL string `json:"repository_url"`
	Ref           string `json:"ref"`
	ServiceID     string `json:"service_id"`
	Attempt       int    `json:"attempt"`
}

var gitRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$`)
var gitHubOwnerPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}$`)
var gitHubRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
var gitCommitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

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
	var job gitBuildClaim
	err = tx.QueryRowContext(r.Context(), `SELECT id, repository_url, git_ref, service_id, attempts
		FROM git_build_jobs
		WHERE status = 'queued' OR (status = 'running' AND lease_expires_at < UTC_TIMESTAMP(6) AND attempts < 5)
		ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&job.ID, &job.RepositoryURL, &job.Ref, &job.ServiceID, &job.Attempt)
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
	result, err := h.db.ExecContext(r.Context(), `UPDATE git_build_jobs SET lease_expires_at = UTC_TIMESTAMP(6) + INTERVAL 15 MINUTE
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
	result, err := h.db.ExecContext(r.Context(), `UPDATE git_build_jobs SET status = 'failed', result_message = ?, finished_at = UTC_TIMESTAMP(6), lease_expires_at = NULL
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
	finalPath := storedPath + ".tar"
	if err := os.Rename(storedPath, finalPath); err != nil {
		writeError(w, http.StatusInternalServerError, "could not publish artifact")
		return
	}
	result, err := h.db.ExecContext(r.Context(), `UPDATE git_build_jobs SET status = 'succeeded', commit_sha = ?, artifact_digest = ?, artifact_path = ?, result_message = NULL,
		finished_at = UTC_TIMESTAMP(6), lease_expires_at = NULL WHERE id = ? AND status = 'running' AND attempts = ? AND lease_expires_at > UTC_TIMESTAMP(6)`, commitSHA, digest, filepath.Base(finalPath), jobID, attempt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not complete build job")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not verify build completion")
		return
	}
	if changed != 1 {
		os.Remove(finalPath)
		writeError(w, http.StatusConflict, "build lease is no longer owned")
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
	var input gitBuildRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	if !validPublicGitHubURL(input.RepositoryURL) || !validGitRef(input.Ref) {
		writeError(w, http.StatusBadRequest, "a public GitHub HTTPS repository and valid ref are required")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create build job")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO git_build_jobs (id, service_id, repository_url, git_ref) VALUES (?, ?, ?, ?)", id, serviceID, input.RepositoryURL, input.Ref); err != nil {
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
	rows, err := h.db.QueryContext(r.Context(), `SELECT b.id, b.service_id, b.repository_url, b.git_ref, COALESCE(b.commit_sha, ''), b.status, COALESCE(b.artifact_digest, ''), COALESCE(b.result_message, ''), b.created_at, b.started_at, b.finished_at
		FROM git_build_jobs b JOIN database_services s ON s.id = b.service_id WHERE b.service_id = ? AND s.workspace_id = ? AND s.project_id = ? ORDER BY b.created_at DESC LIMIT 100`, serviceID, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list build jobs")
		return
	}
	defer rows.Close()
	jobs := make([]gitBuildJob, 0)
	for rows.Next() {
		var job gitBuildJob
		var startedAt, finishedAt *string
		if err := rows.Scan(&job.ID, &job.ServiceID, &job.RepositoryURL, &job.Ref, &job.CommitSHA, &job.Status, &job.ArtifactDigest, &job.ResultMessage, &job.CreatedAt, &startedAt, &finishedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read build jobs")
			return
		}
		if startedAt != nil {
			job.StartedAt = *startedAt
		}
		if finishedAt != nil {
			job.FinishedAt = *finishedAt
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read build jobs")
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}
