package tasks

import (
	"context"
	"net/http"
	"strings"
	"unicode/utf8"

	"kymber/backend/internal/database"
)

type TaskChecklist struct {
	ID    string              `json:"id"`
	Title string              `json:"title"`
	Items []TaskChecklistItem `json:"items"`
}

type TaskChecklistItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type checklistPayload struct {
	Title string `json:"title"`
}

type checklistItemPayload struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func (h *Handler) loadChecklists(ctx context.Context, taskID string) ([]TaskChecklist, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT id, title FROM task_checklists WHERE task_id = ? ORDER BY position, id", taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	checklists := make([]TaskChecklist, 0)
	index := make(map[string]int)
	for rows.Next() {
		checklist := TaskChecklist{Items: []TaskChecklistItem{}}
		if err := rows.Scan(&checklist.ID, &checklist.Title); err != nil {
			return nil, err
		}
		index[checklist.ID] = len(checklists)
		checklists = append(checklists, checklist)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items, err := h.db.QueryContext(ctx, "SELECT i.checklist_id, i.id, i.title, i.done FROM task_checklist_items i JOIN task_checklists c ON c.id = i.checklist_id WHERE c.task_id = ? ORDER BY i.position, i.id", taskID)
	if err != nil {
		return nil, err
	}
	defer items.Close()
	for items.Next() {
		var checklistID string
		var item TaskChecklistItem
		if err := items.Scan(&checklistID, &item.ID, &item.Title, &item.Done); err != nil {
			return nil, err
		}
		checklists[index[checklistID]].Items = append(checklists[index[checklistID]].Items, item)
	}
	return checklists, items.Err()
}

func checklistTitle(w http.ResponseWriter, title string, limit int) (string, bool) {
	title = strings.TrimSpace(title)
	if length := utf8.RuneCountInString(title); length < 1 || length > limit {
		writeError(w, http.StatusBadRequest, "title is empty or too long")
		return "", false
	}
	return title, true
}

func (h *Handler) authorizeChecklist(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return "", "", false
	}
	checklistID, ok := pathID(w, r, "checklistID")
	if !ok || !h.scopeExists(w, r, "SELECT EXISTS(SELECT 1 FROM task_checklists c JOIN project_tasks t ON t.id = c.task_id WHERE c.id = ? AND t.project_id = ?)", "checklist not found", checklistID, projectID) {
		return "", "", false
	}
	return projectID, checklistID, true
}

func (h *Handler) authorizeChecklistItem(w http.ResponseWriter, r *http.Request) (string, bool) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return "", false
	}
	itemID, ok := pathID(w, r, "itemID")
	if !ok || !h.scopeExists(w, r, "SELECT EXISTS(SELECT 1 FROM task_checklist_items i JOIN task_checklists c ON c.id = i.checklist_id JOIN project_tasks t ON t.id = c.task_id WHERE i.id = ? AND t.project_id = ?)", "checklist item not found", itemID, projectID) {
		return "", false
	}
	return itemID, true
}

func (h *Handler) createChecklist(w http.ResponseWriter, r *http.Request) {
	_, _, taskID, ok := h.authorizeTask(w, r, "")
	if !ok {
		return
	}
	var input checklistPayload
	if !decodeJSON(w, r, &input) {
		return
	}
	title, ok := checklistTitle(w, input.Title, 120)
	if !ok {
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate checklist id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO task_checklists (id, task_id, title, position) SELECT ?, ?, ?, COALESCE(MAX(position), 0) + 1 FROM task_checklists WHERE task_id = ?", id, taskID, title, taskID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create checklist")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) updateChecklist(w http.ResponseWriter, r *http.Request) {
	_, checklistID, ok := h.authorizeChecklist(w, r)
	if !ok {
		return
	}
	var input checklistPayload
	if !decodeJSON(w, r, &input) {
		return
	}
	title, ok := checklistTitle(w, input.Title, 120)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE task_checklists SET title = ? WHERE id = ?", title, checklistID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update checklist")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteChecklist(w http.ResponseWriter, r *http.Request) {
	_, checklistID, ok := h.authorizeChecklist(w, r)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "DELETE FROM task_checklists WHERE id = ?", checklistID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete checklist")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createChecklistItem(w http.ResponseWriter, r *http.Request) {
	_, checklistID, ok := h.authorizeChecklist(w, r)
	if !ok {
		return
	}
	var input checklistItemPayload
	if !decodeJSON(w, r, &input) {
		return
	}
	title, ok := checklistTitle(w, input.Title, 300)
	if !ok {
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate checklist item id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO task_checklist_items (id, checklist_id, title, done, position) SELECT ?, ?, ?, ?, COALESCE(MAX(position), 0) + 1 FROM task_checklist_items WHERE checklist_id = ?", id, checklistID, title, input.Done, checklistID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create checklist item")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) updateChecklistItem(w http.ResponseWriter, r *http.Request) {
	itemID, ok := h.authorizeChecklistItem(w, r)
	if !ok {
		return
	}
	var input checklistItemPayload
	if !decodeJSON(w, r, &input) {
		return
	}
	title, ok := checklistTitle(w, input.Title, 300)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE task_checklist_items SET title = ?, done = ? WHERE id = ?", title, input.Done, itemID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update checklist item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteChecklistItem(w http.ResponseWriter, r *http.Request) {
	itemID, ok := h.authorizeChecklistItem(w, r)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "DELETE FROM task_checklist_items WHERE id = ?", itemID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete checklist item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
