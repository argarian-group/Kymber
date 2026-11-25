package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kymber/backend/internal/secrets"
)

type gitPush struct {
	repository string
	branch     string
	commit     string
}

func validHubSignature(secret, header string, body []byte) bool {
	signature, found := strings.CutPrefix(header, "sha256=")
	if !found || secret == "" {
		return false
	}
	expected := hmac.New(sha256.New, []byte(secret))
	expected.Write(body)
	given, err := hex.DecodeString(signature)
	return err == nil && hmac.Equal(given, expected.Sum(nil))
}

func parsePush(header http.Header, body []byte) (gitPush, bool, error) {
	switch {
	case header.Get("X-Gitlab-Event") != "":
		if header.Get("X-Gitlab-Event") != "Push Hook" {
			return gitPush{}, false, nil
		}
		var payload struct {
			Ref         string `json:"ref"`
			After       string `json:"after"`
			CheckoutSHA string `json:"checkout_sha"`
			Project     struct {
				Path string `json:"path_with_namespace"`
			} `json:"project"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return gitPush{}, false, err
		}
		branch, isBranch := strings.CutPrefix(payload.Ref, "refs/heads/")
		commit := payload.CheckoutSHA
		if commit == "" {
			commit = payload.After
		}
		return gitPush{repository: payload.Project.Path, branch: branch, commit: commit}, isBranch && gitCommitPattern.MatchString(commit) && commit != strings.Repeat("0", 40), nil
	case header.Get("X-Event-Key") != "":
		if header.Get("X-Event-Key") != "repo:push" {
			return gitPush{}, false, nil
		}
		var payload struct {
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
			Push struct {
				Changes []struct {
					New *struct {
						Type   string `json:"type"`
						Name   string `json:"name"`
						Target struct {
							Hash string `json:"hash"`
						} `json:"target"`
					} `json:"new"`
				} `json:"changes"`
			} `json:"push"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return gitPush{}, false, err
		}
		for i := len(payload.Push.Changes) - 1; i >= 0; i-- {
			change := payload.Push.Changes[i].New
			if change != nil && change.Type == "branch" && gitCommitPattern.MatchString(change.Target.Hash) {
				return gitPush{repository: payload.Repository.FullName, branch: change.Name, commit: change.Target.Hash}, true, nil
			}
		}
		return gitPush{}, false, nil
	}
	event := header.Get("X-GitHub-Event")
	if event == "" {
		event = header.Get("X-Gitea-Event")
	}
	if event != "push" {
		return gitPush{}, false, nil
	}
	var payload struct {
		Ref        string `json:"ref"`
		After      string `json:"after"`
		Deleted    bool   `json:"deleted"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return gitPush{}, false, err
	}
	branch, isBranch := strings.CutPrefix(payload.Ref, "refs/heads/")
	return gitPush{repository: payload.Repository.FullName, branch: branch, commit: payload.After}, isBranch && !payload.Deleted && gitCommitPattern.MatchString(payload.After), nil
}

func readWebhookBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "webhook payload is too large")
		return nil, false
	}
	return body, true
}

func (h *Handler) sourceWebhook(w http.ResponseWriter, r *http.Request) {
	serviceID := r.PathValue("serviceID")
	if !entityIDPattern.MatchString(serviceID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	body, ok := readWebhookBody(w, r)
	if !ok {
		return
	}
	var encryptedSecret []byte
	if err := h.db.QueryRowContext(r.Context(), "SELECT encrypted_webhook_secret FROM service_git_sources WHERE service_id = ?", serviceID).Scan(&encryptedSecret); err != nil || encryptedSecret == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	secret, err := secrets.Decrypt(encryptedSecret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read webhook secret")
		return
	}
	token := r.Header.Get("X-Gitlab-Token")
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		signature = r.Header.Get("X-Hub-Signature")
	}
	if !(token != "" && subtle.ConstantTimeCompare([]byte(token), secret) == 1) && !validHubSignature(string(secret), signature, body) {
		writeError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}
	if request, ok, err := parsePullRequest(r.Header, body); err != nil || ok {
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid webhook payload")
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": h.handlePullRequest(r.Context(), serviceID, request)})
		return
	}
	push, build, err := parsePush(r.Header, body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook payload")
		return
	}
	if !build {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": h.buildPushedService(r.Context(), serviceID, push)})
}

func (h *Handler) gitHubAppWebhook(w http.ResponseWriter, r *http.Request) {
	body, ok := readWebhookBody(w, r)
	if !ok {
		return
	}
	var envelope struct {
		Installation struct {
			ID int64 `json:"id"`
		} `json:"installation"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook payload")
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, workspace_id FROM git_providers WHERE kind = 'github_app' AND encrypted_credentials IS NOT NULL")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read GitHub Apps")
		return
	}
	type candidate struct{ id, workspaceID string }
	var candidates []candidate
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.id, &item.workspaceID); err == nil {
			candidates = append(candidates, item)
		}
	}
	rows.Close()
	var provider *gitProvider
	for _, item := range candidates {
		loaded, err := h.loadGitProvider(r.Context(), item.workspaceID, item.id)
		if err == nil && validHubSignature(loaded.credentials.WebhookSecret, r.Header.Get("X-Hub-Signature-256"), body) {
			provider = loaded
			break
		}
	}
	if provider == nil {
		writeError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}
	switch r.Header.Get("X-GitHub-Event") {
	case "installation", "installation_repositories":
		if err := h.syncInstallations(r.Context(), provider); err != nil {
			log.Printf("github app %s: %v", provider.ID, err)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "synced"})
		return
	case "ping":
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	request, isPullRequest, err := parsePullRequest(r.Header, body)
	var push gitPush
	build := false
	if err == nil && !isPullRequest {
		push, build, err = parsePush(r.Header, body)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid webhook payload")
		return
	}
	if !build && !isPullRequest {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}
	query, branch := "SELECT service_id FROM service_git_sources WHERE provider_id = ? AND installation_id = ? AND git_ref = ? AND auto_deploy", push.branch
	if isPullRequest {
		query, branch = "SELECT service_id FROM service_git_sources WHERE provider_id = ? AND installation_id = ? AND git_ref = ? AND previews_enabled", request.base
	}
	serviceRows, err := h.db.QueryContext(r.Context(), query, provider.ID, envelope.Installation.ID, branch)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not find apps for this push")
		return
	}
	var serviceIDs []string
	for serviceRows.Next() {
		var id string
		if err := serviceRows.Scan(&id); err == nil {
			serviceIDs = append(serviceIDs, id)
		}
	}
	serviceRows.Close()
	results := map[string]string{}
	for _, id := range serviceIDs {
		if isPullRequest {
			results[id] = h.handlePullRequest(r.Context(), id, request)
		} else {
			results[id] = h.buildPushedService(r.Context(), id, push)
		}
	}
	writeJSON(w, http.StatusAccepted, results)
}

func (h *Handler) buildPushedService(ctx context.Context, serviceID string, push gitPush) string {
	var workspaceID, status string
	var serverID sql.NullString
	if err := h.db.QueryRowContext(ctx, "SELECT workspace_id, status, server_id FROM database_services WHERE id = ?", serviceID).Scan(&workspaceID, &status, &serverID); err != nil {
		return "app not found"
	}
	source, err := h.readGitSource(ctx, serviceID)
	if err != nil {
		return "app has no Git source"
	}
	if !source.AutoDeploy {
		return "automatic deploys are off"
	}
	if source.Ref != push.branch {
		return "branch does not match"
	}
	if !serverID.Valid || status == "draft" || status == "deleted" {
		return "app has not been deployed yet"
	}
	repository, err := h.openGitRepository(ctx, workspaceID, source)
	if err != nil {
		return err.Error()
	}
	if push.repository != "" && !strings.EqualFold(push.repository, repository.path) {
		return "repository does not match"
	}
	var duplicate bool
	if err := h.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM git_build_jobs WHERE service_id = ? AND commit_sha = ? AND status IN ('queued', 'running'))", serviceID, push.commit).Scan(&duplicate); err != nil || duplicate {
		return "commit is already building"
	}
	id, _, err := h.queueGitBuild(ctx, databaseService{ID: serviceID, WorkspaceID: workspaceID}, "push", push.commit)
	if err != nil {
		return err.Error()
	}
	return "queued " + id
}

func (h *Handler) reportCommitStatus(jobID, state, description string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.sendCommitStatus(ctx, jobID, state, description); err != nil {
			log.Printf("commit status for build %s: %v", jobID, err)
		}
	}()
}

func (h *Handler) sendCommitStatus(ctx context.Context, jobID, state, description string) error {
	var serviceID, workspaceID, projectID string
	var commit sql.NullString
	var encryptedConfig []byte
	if err := h.db.QueryRowContext(ctx, `SELECT s.id, s.workspace_id, s.project_id, b.commit_sha, b.encrypted_build_config FROM git_build_jobs b JOIN database_services s ON s.id = b.service_id WHERE b.id = ?`, jobID).
		Scan(&serviceID, &workspaceID, &projectID, &commit, &encryptedConfig); err != nil {
		return err
	}
	if !commit.Valid || encryptedConfig == nil {
		return nil
	}
	plaintext, err := secrets.Decrypt(encryptedConfig)
	if err != nil {
		return err
	}
	var source gitSource
	if err := json.Unmarshal(plaintext, &source); err != nil {
		return err
	}
	if source.ProviderID == "" {
		return nil
	}
	repository, err := h.openGitRepository(ctx, workspaceID, source)
	if err != nil {
		return err
	}
	targetURL := ""
	if panel, err := h.panelURL(ctx); err == nil {
		targetURL = panel + "/projects/" + projectID + "/apps/" + serviceID + "/deployments"
	}
	switch repository.provider.kind() {
	case "github_app", "github_token":
		githubState := map[string]string{"pending": "pending", "running": "pending", "success": "success", "failure": "failure"}[state]
		return repository.api(ctx, http.MethodPost, repository.apiPath()+"/statuses/"+commit.String, map[string]string{"state": githubState, "description": description, "context": "kymber/build", "target_url": targetURL}, nil)
	case "gitlab":
		gitlabState := map[string]string{"pending": "pending", "running": "running", "success": "success", "failure": "failed"}[state]
		return repository.api(ctx, http.MethodPost, repository.apiPath()+"/statuses/"+commit.String, map[string]string{"state": gitlabState, "description": description, "name": "kymber/build", "target_url": targetURL}, nil)
	case "bitbucket":
		if targetURL == "" {
			return errors.New("Bitbucket build statuses need the panel domain")
		}
		bitbucketState := map[string]string{"pending": "INPROGRESS", "running": "INPROGRESS", "success": "SUCCESSFUL", "failure": "FAILED"}[state]
		return repository.api(ctx, http.MethodPost, repository.apiPath()+"/commit/"+commit.String+"/statuses/build", map[string]string{"state": bitbucketState, "key": "kymber-build", "name": "Kymber build", "description": description, "url": targetURL}, nil)
	}
	return nil
}

func (h *Handler) webhookURL(ctx context.Context, serviceID string) string {
	panel, err := h.panelURL(ctx)
	if err != nil {
		return ""
	}
	return panel + "/api/webhooks/git/" + url.PathEscape(serviceID)
}
