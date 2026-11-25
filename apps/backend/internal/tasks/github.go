package tasks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type githubReference struct {
	number int
	closes bool
}

type githubMention struct {
	source, title, url string
	canClose           bool
	refs               []githubReference
}

type githubPayload struct {
	Ref        string `json:"ref"`
	Action     string `json:"action"`
	Repository struct {
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
	Commits []struct {
		Message string `json:"message"`
		URL     string `json:"url"`
	} `json:"commits"`
	PullRequest *struct {
		Title   string `json:"title"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		Merged  bool   `json:"merged"`
	} `json:"pull_request"`
}

func taskReferences(prefix, text string) []githubReference {
	pattern := regexp.MustCompile(`(?i)(?:\b(fix(?:es|ed)?|close[sd]?|resolve[sd]?):?\s+)?\b` + regexp.QuoteMeta(prefix) + `-(\d+)\b`)
	var refs []githubReference
	seen := map[int]int{}
	for _, match := range pattern.FindAllStringSubmatch(text, 50) {
		number, err := strconv.Atoi(match[2])
		if err != nil {
			continue
		}
		if index, ok := seen[number]; ok {
			refs[index].closes = refs[index].closes || match[1] != ""
			continue
		}
		seen[number] = len(refs)
		refs = append(refs, githubReference{number, match[1] != ""})
	}
	return refs
}

func githubMentions(event, prefix string, payload githubPayload) []githubMention {
	var mentions []githubMention
	switch event {
	case "push":
		onDefault := payload.Repository.DefaultBranch != "" && payload.Ref == "refs/heads/"+payload.Repository.DefaultBranch
		for _, commit := range payload.Commits {
			if refs := taskReferences(prefix, commit.Message); len(refs) > 0 {
				title, _, _ := strings.Cut(commit.Message, "\n")
				mentions = append(mentions, githubMention{"commit", title, commit.URL, onDefault, refs})
			}
		}
	case "pull_request":
		pr := payload.PullRequest
		merged := payload.Action == "closed" && pr != nil && pr.Merged
		if pr != nil && (payload.Action == "opened" || merged) {
			if refs := taskReferences(prefix, pr.Title+"\n"+pr.Body); len(refs) > 0 {
				mentions = append(mentions, githubMention{"pull_request", pr.Title, pr.HTMLURL, merged, refs})
			}
		}
	}
	return mentions
}

// githubWebhook links commits and pull requests that mention PREFIX-N to tasks, and closes the task on "fixes PREFIX-N" once merged.
func (h *Handler) githubWebhook(w http.ResponseWriter, r *http.Request) {
	projectID, ok := pathID(w, r, "projectID")
	if !ok {
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 5<<20))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "payload too large")
		return
	}
	var secret sql.NullString
	var prefix string
	err = h.db.QueryRowContext(r.Context(), "SELECT github_webhook_secret, task_prefix FROM projects WHERE id = ?", projectID).Scan(&secret, &prefix)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "could not read project")
		return
	}
	if !secret.Valid || !validSignature(secret.String, body, r.Header.Get("X-Hub-Signature-256")) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}
	var payload githubPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	for _, mention := range githubMentions(r.Header.Get("X-GitHub-Event"), prefix, payload) {
		for _, ref := range mention.refs {
			if err := h.applyGitHubReference(r, projectID, mention, ref); err != nil {
				writeError(w, http.StatusInternalServerError, "could not update tasks")
				return
			}
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func validSignature(secret string, body []byte, header string) bool {
	signature, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil || !strings.HasPrefix(header, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(signature, mac.Sum(nil))
}

func (h *Handler) applyGitHubReference(r *http.Request, projectID string, mention githubMention, ref githubReference) error {
	ctx := r.Context()
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var taskID, listName string
	var done bool
	err = tx.QueryRowContext(ctx, "SELECT t.id, l.name, l.is_done FROM project_tasks t JOIN task_lists l ON l.id = t.list_id WHERE t.project_id = ? AND t.number = ? FOR UPDATE", projectID, ref.number).Scan(&taskID, &listName, &done)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	closes := ref.closes && mention.canClose
	event := activityEvent{"referenced", map[string]any{"via": "github", "source": mention.source, "title": mention.title, "url": mention.url, "closes": closes}}
	if err := recordActivity(ctx, tx, taskID, nil, event); err != nil {
		return err
	}
	if closes && !done {
		var doneListID, doneName string
		err := tx.QueryRowContext(ctx, "SELECT id, name FROM task_lists WHERE project_id = ? AND is_done ORDER BY position, id LIMIT 1", projectID).Scan(&doneListID, &doneName)
		if err == nil {
			if _, err := tx.ExecContext(ctx, "UPDATE project_tasks SET list_id = ?, position = COALESCE((SELECT MAX(position) FROM project_tasks WHERE list_id = ?), 0) + 1 WHERE id = ?", doneListID, doneListID, taskID); err != nil {
				return err
			}
			if err := recordActivity(ctx, tx, taskID, nil, activityEvent{"moved", map[string]any{"via": "github", "from": listName, "to": doneName}}); err != nil {
				return err
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	return tx.Commit()
}

// deployedAt is when the first build started after GitHub closed the task finished deploying any of the project's apps.
// shortcut: matches by time, not by commit ancestry, so a build of another branch started after the merge counts too.
func (h *Handler) deployedAt(ctx context.Context, taskID string) (*time.Time, error) {
	var deployed sql.NullTime
	err := h.db.QueryRowContext(ctx, `SELECT MIN(d.finished_at) FROM task_activity a
		JOIN project_tasks t ON t.id = a.task_id
		JOIN database_services s ON s.project_id = t.project_id
		JOIN git_build_jobs b ON b.service_id = s.id AND b.created_at >= a.created_at
		JOIN service_deployments d ON d.service_id = s.id AND d.image_version = b.id AND d.status = 'succeeded'
		WHERE a.task_id = ? AND a.kind = 'referenced' AND JSON_EXTRACT(a.payload, '$.closes') = 'true'`, taskID).Scan(&deployed)
	if err != nil || !deployed.Valid {
		return nil, err
	}
	return &deployed.Time, nil
}
