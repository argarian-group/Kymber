package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"kymber/backend/internal/database"
)

type gitPullRequest struct {
	repository string
	number     int
	base       string
	ref        string
	commit     string
	closed     bool
}

func parsePullRequest(header http.Header, body []byte) (gitPullRequest, bool, error) {
	switch {
	case header.Get("X-Gitlab-Event") != "":
		if header.Get("X-Gitlab-Event") != "Merge Request Hook" {
			return gitPullRequest{}, false, nil
		}
		var payload struct {
			Project struct {
				Path string `json:"path_with_namespace"`
			} `json:"project"`
			Attributes struct {
				IID             int    `json:"iid"`
				Action          string `json:"action"`
				TargetBranch    string `json:"target_branch"`
				SourceProjectID int    `json:"source_project_id"`
				TargetProjectID int    `json:"target_project_id"`
				LastCommit      struct {
					ID string `json:"id"`
				} `json:"last_commit"`
			} `json:"object_attributes"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return gitPullRequest{}, false, err
		}
		a := payload.Attributes
		request := gitPullRequest{repository: payload.Project.Path, number: a.IID, base: a.TargetBranch, ref: fmt.Sprintf("refs/merge-requests/%d/head", a.IID), commit: a.LastCommit.ID, closed: a.Action == "close" || a.Action == "merge"}
		active := request.closed || a.Action == "open" || a.Action == "reopen" || a.Action == "update"
		return request, active && a.IID > 0 && a.SourceProjectID == a.TargetProjectID && (request.closed || gitCommitPattern.MatchString(request.commit)), nil
	case header.Get("X-Event-Key") != "":
		event := header.Get("X-Event-Key")
		if !strings.HasPrefix(event, "pullrequest:") {
			return gitPullRequest{}, false, nil
		}
		var payload struct {
			PullRequest struct {
				ID     int `json:"id"`
				Source struct {
					Branch struct {
						Name string `json:"name"`
					} `json:"branch"`
					Repository struct {
						FullName string `json:"full_name"`
					} `json:"repository"`
				} `json:"source"`
				Destination struct {
					Branch struct {
						Name string `json:"name"`
					} `json:"branch"`
					Repository struct {
						FullName string `json:"full_name"`
					} `json:"repository"`
				} `json:"destination"`
			} `json:"pullrequest"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return gitPullRequest{}, false, err
		}
		p := payload.PullRequest
		closed := event == "pullrequest:fulfilled" || event == "pullrequest:rejected"
		active := closed || event == "pullrequest:created" || event == "pullrequest:updated"
		request := gitPullRequest{repository: p.Destination.Repository.FullName, number: p.ID, base: p.Destination.Branch.Name, ref: p.Source.Branch.Name, closed: closed}
		return request, active && p.ID > 0 && strings.EqualFold(p.Source.Repository.FullName, p.Destination.Repository.FullName), nil
	}
	event := header.Get("X-GitHub-Event")
	if event == "" {
		event = header.Get("X-Gitea-Event")
	}
	if event != "pull_request" {
		return gitPullRequest{}, false, nil
	}
	var payload struct {
		Action      string `json:"action"`
		Number      int    `json:"number"`
		PullRequest struct {
			Head struct {
				SHA  string `json:"sha"`
				Repo struct {
					FullName string `json:"full_name"`
				} `json:"repo"`
			} `json:"head"`
			Base struct {
				Ref  string `json:"ref"`
				Repo struct {
					FullName string `json:"full_name"`
				} `json:"repo"`
			} `json:"base"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return gitPullRequest{}, false, err
	}
	head, base := payload.PullRequest.Head, payload.PullRequest.Base
	request := gitPullRequest{repository: base.Repo.FullName, number: payload.Number, base: base.Ref, ref: fmt.Sprintf("refs/pull/%d/head", payload.Number), commit: head.SHA, closed: payload.Action == "closed"}
	active := request.closed || payload.Action == "opened" || payload.Action == "reopened" || payload.Action == "synchronize"
	return request, active && payload.Number > 0 && strings.EqualFold(head.Repo.FullName, base.Repo.FullName) && (request.closed || gitCommitPattern.MatchString(head.SHA)), nil
}

func previewNames(parentName, previewDomain string, number int) (string, string) {
	suffix := fmt.Sprintf("-pr-%d", number)
	name := parentName
	if len(name)+len(suffix) > 63 {
		name = strings.TrimRight(name[:63-len(suffix)], "-")
	}
	label := fmt.Sprintf("pr-%d-%s", number, parentName)
	if len(label) > 63 {
		label = strings.TrimRight(label[:63], "-")
	}
	return name + suffix, label + "." + previewDomain
}

func (h *Handler) handlePullRequest(ctx context.Context, parentID string, request gitPullRequest) string {
	var workspaceID, projectID, name, status string
	var serverID sql.NullString
	var previewOf sql.NullString
	if err := h.db.QueryRowContext(ctx, "SELECT workspace_id, project_id, name, status, server_id, preview_of FROM database_services WHERE id = ?", parentID).Scan(&workspaceID, &projectID, &name, &status, &serverID, &previewOf); err != nil {
		return "app not found"
	}
	source, err := h.readGitSource(ctx, parentID)
	if err != nil {
		return "app has no Git source"
	}
	if previewOf.Valid || !source.PreviewsEnabled || source.PreviewDomain == "" {
		return "previews are off"
	}
	if source.Ref != request.base {
		return "target branch does not match"
	}
	repository, err := h.openGitRepository(ctx, workspaceID, source)
	if err != nil {
		return err.Error()
	}
	if request.repository != "" && !strings.EqualFold(request.repository, repository.path) {
		return "repository does not match"
	}
	var previewID string
	err = h.db.QueryRowContext(ctx, "SELECT id FROM database_services WHERE preview_of = ? AND pr_number = ?", parentID, request.number).Scan(&previewID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "could not read preview"
	}
	if request.closed {
		if previewID == "" {
			return "no preview to remove"
		}
		recorder := httptest.NewRecorder()
		h.runDeleteService(recorder, httptest.NewRequestWithContext(ctx, http.MethodDelete, "/", nil), workspaceID, projectID, previewID)
		if recorder.Code >= http.StatusBadRequest {
			return "could not remove preview: " + strings.TrimSpace(recorder.Body.String())
		}
		return "preview removed"
	}
	if !serverID.Valid || status == "draft" || status == "deleted" {
		return "app has not been deployed yet"
	}
	created := false
	if previewID == "" {
		if previewID, err = h.createPreview(ctx, parentID, name, source.PreviewDomain, request); err != nil {
			return "could not create preview: " + err.Error()
		}
		created = true
	} else if _, err := h.db.ExecContext(ctx, "UPDATE service_git_sources SET git_ref = ? WHERE service_id = ?", request.ref, previewID); err != nil {
		return "could not update preview"
	}
	id, _, err := h.queueGitBuild(ctx, databaseService{ID: previewID, WorkspaceID: workspaceID}, "pull_request", request.commit, false)
	if err != nil {
		return err.Error()
	}
	if created {
		_, domain := previewNames(name, source.PreviewDomain, request.number)
		go h.commentPreview(repository, request.number, domain)
	}
	return "queued " + id
}

func (h *Handler) createPreview(ctx context.Context, parentID, parentName, previewDomain string, request gitPullRequest) (string, error) {
	id, err := database.NewShortID()
	if err != nil {
		return "", err
	}
	name, domain := previewNames(parentName, previewDomain, request.number)
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO database_services (id, workspace_id, project_id, server_id, engine, name, image_version, image, port, domain, data_path, logo_url, category, environment_id, encrypted_config, status, created_by, preview_of, pr_number)
		SELECT ?, workspace_id, project_id, server_id, engine, ?, 'draft', 'docker.io/kymber/build-artifacts', port, ?, data_path, logo_url, category, environment_id, encrypted_config, 'stopped', created_by, id, ? FROM database_services WHERE id = ?`,
		id, name, domain, request.number, parentID); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO service_git_sources (service_id, provider_id, installation_id, repository_url, git_ref, base_directory, dockerfile_path, build_pack, install_command, build_command, start_command, publish_directory, encrypted_build_args, auto_deploy)
		SELECT ?, provider_id, installation_id, repository_url, ?, base_directory, dockerfile_path, build_pack, install_command, build_command, start_command, publish_directory, encrypted_build_args, FALSE FROM service_git_sources WHERE service_id = ?`,
		id, request.ref, parentID); err != nil {
		return "", err
	}
	return id, tx.Commit()
}

func (h *Handler) commentPreview(repository gitRepository, number int, domain string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	body := fmt.Sprintf("Kymber is deploying a preview of this pull request at https://%s", domain)
	var err error
	switch repository.provider.kind() {
	case "github_app", "github_token":
		err = repository.api(ctx, http.MethodPost, fmt.Sprintf("%s/issues/%d/comments", repository.apiPath(), number), map[string]string{"body": body}, nil)
	case "gitlab":
		err = repository.api(ctx, http.MethodPost, fmt.Sprintf("%s/merge_requests/%d/notes", repository.apiPath(), number), map[string]string{"body": body}, nil)
	case "bitbucket":
		err = repository.api(ctx, http.MethodPost, fmt.Sprintf("%s/pullrequests/%d/comments", repository.apiPath(), number), map[string]any{"content": map[string]string{"raw": body}}, nil)
	}
	if err != nil {
		log.Printf("preview comment on #%d: %v", number, err)
	}
}
