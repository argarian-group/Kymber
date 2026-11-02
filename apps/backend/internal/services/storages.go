package services

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/secrets"
	"kymber/backend/internal/templates"
)

// The script travels as one shell argument, which Linux caps at 128 KiB once file content is base64 encoded.
const maxStorageContent = 64 << 10
const maxStoragesPerService = 20

type serviceStorage struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Source     string `json:"source,omitempty"`
	MountPath  string `json:"mount_path"`
	VolumeName string `json:"volume_name,omitempty"`
	Applied    bool   `json:"applied"`
	content    []byte
}

func validateStorage(kind, source, mountPath, dataPath string, isRoot bool) error {
	if mountPath == "" || mountPath == "/" || !templates.ValidDataPath(mountPath) {
		return errors.New("mount path must be a clean absolute path")
	}
	if mountPath == dataPath {
		return errors.New("the app's main volume already uses this mount path")
	}
	switch kind {
	case "volume", "file":
		return nil
	case "bind":
		// Deploys run as the SSH user, so a bind mount reaches anything on the host that user can.
		if !isRoot {
			return errors.New("only the root account can mount server directories")
		}
		if source == "" || source == "/" || !templates.ValidDataPath(source) {
			return errors.New("source must be a clean absolute path on the server")
		}
		return nil
	}
	return errors.New("unsupported storage kind")
}

func storageVolumeName(serviceID, storageID string) string {
	return "kymber-service-" + serviceID + "-" + storageID
}

// Double quotes let the server expand $HOME; both IDs are generated hex.
func storageFilesDir(serviceID string) string {
	return `"$HOME/.kymber/service-files/` + serviceID + `"`
}

func storageFilePath(serviceID, storageID string) string {
	return `"$HOME/.kymber/service-files/` + serviceID + "/" + storageID + `/content"`
}

func storageDeployCommand(serviceID string, storages []serviceStorage) (prepare, args string) {
	for _, s := range storages {
		switch s.Kind {
		case "volume":
			args += " --volume " + shellQuote(storageVolumeName(serviceID, s.ID)+":"+s.MountPath)
		case "bind":
			args += " --volume " + shellQuote(s.Source+":"+s.MountPath)
		case "file":
			file := storageFilePath(serviceID, s.ID)
			// The directory stays private to the host user while the container's user can still read the file.
			prepare += "mkdir -p -m 700 " + strings.TrimSuffix(file, `/content"`) + `"; printf '%s' '` + base64.StdEncoding.EncodeToString(s.content) + "' | base64 -d > " + file + "; chmod 644 " + file + "; "
			args += " --volume " + file + shellQuote(":"+s.MountPath)
		}
	}
	return prepare, args
}

// Matching by name also catches volumes whose mount was removed while keeping their data; the main "-data" volume has no 6-hex suffix.
func storageDeleteCommand(serviceID string) string {
	return "rm -rf " + storageFilesDir(serviceID) + "; docker volume ls -q --filter " + shellQuote("name=^kymber-service-"+serviceID+"-[0-9a-f]{6}$") + " | xargs -r docker volume rm >/dev/null 2>&1 || true; "
}

func (h *Handler) readStorages(ctx context.Context, serviceID string, withContent bool) ([]serviceStorage, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT id, kind, COALESCE(source, ''), mount_path, encrypted_content, applied FROM service_storages WHERE service_id = ? ORDER BY created_at, id", serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	storages := make([]serviceStorage, 0)
	for rows.Next() {
		var s serviceStorage
		var encrypted []byte
		if err := rows.Scan(&s.ID, &s.Kind, &s.Source, &s.MountPath, &encrypted, &s.Applied); err != nil {
			return nil, err
		}
		if s.Kind == "volume" {
			s.VolumeName = storageVolumeName(serviceID, s.ID)
		}
		if withContent && s.Kind == "file" {
			if s.content, err = secrets.Decrypt(encrypted); err != nil {
				return nil, err
			}
		}
		storages = append(storages, s)
	}
	return storages, rows.Err()
}

// storageRequest authorizes the caller and returns the app the URL names.
func (h *Handler) storageRequest(w http.ResponseWriter, r *http.Request, write bool) (databaseService, auth.User, bool) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, write) {
		return databaseService{}, user, false
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return databaseService{}, user, false
	}
	serviceID, ok := pathID(w, r, "serviceID")
	if !ok {
		return databaseService{}, user, false
	}
	service, err := h.readService(r, workspaceID, projectID, serviceID, false)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "service not found")
		return service, user, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read service")
		return service, user, false
	}
	if write && (service.Status == "queued" || service.Status == "deploying") {
		writeError(w, http.StatusConflict, "wait for the current action to finish")
		return service, user, false
	}
	return service, user, true
}

func (h *Handler) checkStorageContent(w http.ResponseWriter, r *http.Request, serviceID, exceptID string, size int) bool {
	var used int
	if err := h.db.QueryRowContext(r.Context(), "SELECT COALESCE(SUM(LENGTH(encrypted_content)), 0) FROM service_storages WHERE service_id = ? AND id <> ?", serviceID, exceptID).Scan(&used); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check file mounts")
		return false
	}
	if used+size > maxStorageContent {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("file mounts of one app are limited to %d KB in total", maxStorageContent>>10))
		return false
	}
	return true
}

func (h *Handler) listStorages(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.storageRequest(w, r, false)
	if !ok {
		return
	}
	storages, err := h.readStorages(r.Context(), service.ID, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read storages")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]serviceStorage{"storages": storages})
}

func (h *Handler) createStorage(w http.ResponseWriter, r *http.Request) {
	service, user, ok := h.storageRequest(w, r, true)
	if !ok {
		return
	}
	var input struct {
		Kind      string `json:"kind"`
		Source    string `json:"source"`
		MountPath string `json:"mount_path"`
		Content   string `json:"content"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Source, input.MountPath = strings.TrimSpace(input.Source), strings.TrimSpace(input.MountPath)
	if err := validateStorage(input.Kind, input.Source, input.MountPath, serviceTemplate(service).DataPath, user.IsRoot); err != nil {
		status := http.StatusBadRequest
		if input.Kind == "bind" && !user.IsRoot {
			status = http.StatusForbidden
		}
		writeError(w, status, err.Error())
		return
	}
	var count int
	if err := h.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM service_storages WHERE service_id = ?", service.ID).Scan(&count); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read storages")
		return
	}
	if count >= maxStoragesPerService {
		writeError(w, http.StatusConflict, fmt.Sprintf("an app can have up to %d storages", maxStoragesPerService))
		return
	}
	var source sql.NullString
	var encrypted []byte
	if input.Kind == "bind" {
		source = sql.NullString{String: input.Source, Valid: true}
	}
	if input.Kind == "file" {
		var err error
		if encrypted, err = secrets.Encrypt([]byte(input.Content)); err != nil {
			writeError(w, http.StatusInternalServerError, "service encryption is not configured")
			return
		}
		if !h.checkStorageContent(w, r, service.ID, "", len(encrypted)) {
			return
		}
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate storage id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO service_storages (id, service_id, kind, source, mount_path, encrypted_content) VALUES (?, ?, ?, ?, ?, ?)", id, service.ID, input.Kind, source, input.MountPath, encrypted); err != nil {
		writeError(w, http.StatusConflict, "this mount path is already in use")
		return
	}
	storage := serviceStorage{ID: id, Kind: input.Kind, Source: input.Source, MountPath: input.MountPath}
	if input.Kind == "volume" {
		storage.VolumeName = storageVolumeName(service.ID, id)
	}
	writeJSON(w, http.StatusCreated, map[string]serviceStorage{"storage": storage})
}

func (h *Handler) readStorageRow(w http.ResponseWriter, r *http.Request, serviceID string) (string, []byte, bool) {
	id, ok := pathID(w, r, "storageID")
	if !ok {
		return "", nil, false
	}
	var kind string
	var encrypted []byte
	if err := h.db.QueryRowContext(r.Context(), "SELECT kind, encrypted_content FROM service_storages WHERE id = ? AND service_id = ?", id, serviceID).Scan(&kind, &encrypted); errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "storage not found")
		return "", nil, false
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read storage")
		return "", nil, false
	}
	return kind, encrypted, true
}

// File content can hold secrets, so unlike the list it needs write access.
func (h *Handler) getStorageContent(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.storageRequest(w, r, true)
	if !ok {
		return
	}
	kind, encrypted, ok := h.readStorageRow(w, r, service.ID)
	if !ok {
		return
	}
	if kind != "file" {
		writeError(w, http.StatusConflict, "only file mounts have content")
		return
	}
	content, err := secrets.Decrypt(encrypted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read file content")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": string(content)})
}

func (h *Handler) updateStorageContent(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.storageRequest(w, r, true)
	if !ok {
		return
	}
	kind, _, ok := h.readStorageRow(w, r, service.ID)
	if !ok {
		return
	}
	if kind != "file" {
		writeError(w, http.StatusConflict, "only file mounts have content")
		return
	}
	var input struct {
		Content string `json:"content"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	encrypted, err := secrets.Encrypt([]byte(input.Content))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "service encryption is not configured")
		return
	}
	id := r.PathValue("storageID")
	if !h.checkStorageContent(w, r, service.ID, id, len(encrypted)) {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE service_storages SET encrypted_content = ?, applied = FALSE WHERE id = ?", encrypted, id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save file content")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteStorage(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.storageRequest(w, r, true)
	if !ok {
		return
	}
	kind, _, ok := h.readStorageRow(w, r, service.ID)
	if !ok {
		return
	}
	id := r.PathValue("storageID")
	if service.ServerID != "" {
		switch {
		case kind == "volume" && r.URL.Query().Get("delete_data") == "true":
			// Removing the data must succeed first, or the volume would be left with nothing pointing at it.
			if _, err := h.executeSSH(r.Context(), service.ServerID, "output=$(docker volume rm "+shellQuote(storageVolumeName(service.ID, id))+" 2>&1) || case \"$output\" in *'no such volume'*) ;; *) printf '%s' \"$output\" >&2; exit 1;; esac"); err != nil {
				writeError(w, http.StatusConflict, "could not delete the volume's data. Remove the container first, then try again: "+err.Error())
				return
			}
		case kind == "file":
			// A leftover file holds no state the app needs and the record is what decides future mounts.
			_, _ = h.executeSSH(r.Context(), service.ServerID, "rm -rf "+strings.TrimSuffix(storageFilePath(service.ID, id), `/content"`)+`"`)
		}
	}
	if _, err := h.db.ExecContext(r.Context(), "DELETE FROM service_storages WHERE id = ? AND service_id = ?", id, service.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete storage")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
