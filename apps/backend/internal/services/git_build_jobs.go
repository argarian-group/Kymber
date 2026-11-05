package services

import (
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

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

var gitRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$`)
var gitHubOwnerPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}$`)
var gitHubRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)

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

func (h *Handler) createGitBuild(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("KYMBER_GIT_BUILDS_ENABLED") != "true" {
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
