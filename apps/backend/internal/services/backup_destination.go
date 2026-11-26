package services

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/backupstore"
	"kymber/backend/internal/secrets"
)

var bucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)

func requireRoot(w http.ResponseWriter, r *http.Request) bool {
	if user, _ := auth.UserFromContext(r); !user.IsRoot {
		writeError(w, http.StatusForbidden, "root account required")
		return false
	}
	return true
}

func (h *Handler) getBackupDestination(w http.ResponseWriter, r *http.Request) {
	if !requireRoot(w, r) {
		return
	}
	config, encrypted, err := backupstore.ReadConfig(r.Context(), h.db)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read the backup destination")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"destination": config, "has_secret": len(encrypted) > 0})
}

func (h *Handler) setBackupDestination(w http.ResponseWriter, r *http.Request) {
	if !requireRoot(w, r) {
		return
	}
	var input struct {
		backupstore.Config
		SecretKey string `json:"secret_key"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.Bucket == "" {
		if _, err := h.db.ExecContext(r.Context(), "UPDATE platform_settings SET s3_endpoint = '', s3_region = '', s3_bucket = '', s3_prefix = '', s3_access_key = '', encrypted_s3_secret = NULL WHERE id = 1"); err != nil {
			writeError(w, http.StatusInternalServerError, "could not save the backup destination")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !bucketPattern.MatchString(input.Bucket) || input.Endpoint == "" || input.AccessKey == "" || len(input.Endpoint) > 255 || len(input.Region) > 64 || len(input.Prefix) > 255 || len(input.AccessKey) > 255 || len(input.SecretKey) > 512 {
		writeError(w, http.StatusBadRequest, "fill in the endpoint, a valid bucket name and the access key")
		return
	}
	secret := input.SecretKey
	if secret == "" {
		_, encrypted, err := backupstore.ReadConfig(r.Context(), h.db)
		if err != nil || len(encrypted) == 0 {
			writeError(w, http.StatusBadRequest, "enter the secret key")
			return
		}
		plain, err := secrets.Decrypt(encrypted)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not read the saved secret key")
			return
		}
		secret = string(plain)
	}
	store, err := backupstore.New(input.Config, secret)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := store.Check(ctx); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	encrypted, err := secrets.Encrypt([]byte(secret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "service encryption is not configured")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE platform_settings SET s3_endpoint = ?, s3_region = ?, s3_bucket = ?, s3_prefix = ?, s3_access_key = ?, encrypted_s3_secret = ? WHERE id = 1", input.Endpoint, input.Region, input.Bucket, input.Prefix, input.AccessKey, encrypted); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save the backup destination")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
