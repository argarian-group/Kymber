package services

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"kymber/backend/internal/database"
	"kymber/backend/internal/logger"
	"kymber/backend/internal/secrets"
)

const platformBackupDir = `"$HOME/.local/share/kymber/platform-backups/"`

func platformBackupFile(backupID string) string {
	return platformBackupDir + backupID + ".sql.gz"
}

func platformBackupCommand(project, backupID string) string {
	prepare := "c=$(docker ps -q --filter label=com.docker.compose.project=" + shellQuote(project) + " --filter label=com.docker.compose.service=mariadb | head -n 1); [ -n \"$c\" ] || { echo 'could not find the Kymber database container' >&2; exit 1; }; "
	source := `docker exec "$c" sh -c 'mariadb-dump -uroot -p"$MARIADB_ROOT_PASSWORD" --databases "$MARIADB_DATABASE" --single-transaction --routines --events'`
	return archiveCommand(prepare, source, platformBackupDir, platformBackupFile(backupID))
}

func composeProjectName() string {
	if project := os.Getenv("KYMBER_COMPOSE_PROJECT"); project != "" {
		return project
	}
	return "kymber"
}

func (h *Handler) localServerID(ctx context.Context) (string, error) {
	var id string
	err := h.db.QueryRowContext(ctx, "SELECT id FROM servers WHERE connection_type = 'local' ORDER BY created_at LIMIT 1").Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("the local server is not set up")
	}
	return id, err
}

func (h *Handler) listPlatformBackups(w http.ResponseWriter, r *http.Request) {
	if !requireRoot(w, r) {
		return
	}
	var settings struct {
		Schedule  string     `json:"schedule"`
		Keep      int        `json:"keep"`
		NextRunAt *time.Time `json:"next_run_at"`
	}
	if err := h.db.QueryRowContext(r.Context(), "SELECT backup_schedule, backup_keep, backup_next_run_at FROM platform_settings WHERE id = 1").Scan(&settings.Schedule, &settings.Keep, &settings.NextRunAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read backup settings")
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, status, scheduled, size_bytes, COALESCE(error, ''), created_at, finished_at, remote_key IS NOT NULL FROM platform_backups ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list backups")
		return
	}
	defer rows.Close()
	backups := make([]serviceBackup, 0)
	for rows.Next() {
		var backup serviceBackup
		if err := rows.Scan(&backup.ID, &backup.Status, &backup.Scheduled, &backup.SizeBytes, &backup.Error, &backup.CreatedAt, &backup.FinishedAt, &backup.Remote); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read backups")
			return
		}
		backups = append(backups, backup)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read backups")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings, "backups": backups})
}

func (h *Handler) updatePlatformBackupSettings(w http.ResponseWriter, r *http.Request) {
	if !requireRoot(w, r) {
		return
	}
	var input struct {
		Schedule string `json:"schedule"`
		Keep     int    `json:"keep"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Schedule = strings.TrimSpace(input.Schedule)
	if input.Keep < 1 || input.Keep > 365 {
		writeError(w, http.StatusBadRequest, "keep between 1 and 365 backups")
		return
	}
	var nextRun any
	if input.Schedule != "" {
		schedule, err := parseCron(input.Schedule)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		nextRun = schedule.next(time.Now())
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE platform_settings SET backup_schedule = ?, backup_keep = ?, backup_next_run_at = ? WHERE id = 1", input.Schedule, input.Keep, nextRun); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save backup settings")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createPlatformBackup(w http.ResponseWriter, r *http.Request) {
	if !requireRoot(w, r) {
		return
	}
	id, err := h.startPlatformBackup(r.Context(), false)
	if errors.Is(err, errServiceActionInProgress) {
		writeError(w, http.StatusConflict, "a backup is already running")
		return
	}
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id})
}

func (h *Handler) startPlatformBackup(ctx context.Context, scheduled bool) (string, error) {
	serverID, err := h.localServerID(ctx)
	if err != nil {
		return "", err
	}
	id, err := database.NewShortID()
	if err != nil {
		return "", err
	}
	result, err := h.db.ExecContext(ctx, `INSERT INTO platform_backups (id, server_id, scheduled) SELECT ?, ?, ? FROM DUAL
		WHERE NOT EXISTS (SELECT 1 FROM platform_backups WHERE status = 'running' AND created_at > UTC_TIMESTAMP(6) - INTERVAL 7 HOUR)`, id, serverID, scheduled)
	if err != nil {
		return "", err
	}
	if inserted, _ := result.RowsAffected(); inserted == 0 {
		return "", errServiceActionInProgress
	}
	go h.runPlatformBackup(serverID, id)
	return id, nil
}

func (h *Handler) runPlatformBackup(serverID, backupID string) {
	ctx, cancel := context.WithTimeout(context.Background(), backupTimeout)
	defer cancel()
	finish := func(status string, size int64, remoteKey, message string) {
		var key, errorText any
		if remoteKey != "" {
			key = remoteKey
		}
		if message != "" {
			errorText = message
		}
		if _, err := h.db.ExecContext(context.Background(), "UPDATE platform_backups SET status = ?, size_bytes = ?, remote_key = ?, error = ?, finished_at = UTC_TIMESTAMP(6) WHERE id = ?", status, size, key, errorText, backupID); err != nil {
			logger.Warn("could not save platform backup %s: %v", backupID, err)
		}
	}
	output, err := h.executeSSH(ctx, serverID, platformBackupCommand(composeProjectName(), backupID))
	if err != nil {
		finish("failed", 0, "", err.Error())
		return
	}
	size, _ := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	key, err := h.uploadBackup(ctx, serverID, platformBackupFile(backupID), size, "platform", backupID+".sql.gz")
	message := ""
	if err != nil {
		message = "Saved on the server, but the copy to S3 failed: " + err.Error()
	}
	finish("succeeded", size, key, message)
	h.prunePlatformBackups(ctx)
}

func (h *Handler) prunePlatformBackups(ctx context.Context) {
	keep := 7
	_ = h.db.QueryRowContext(ctx, "SELECT backup_keep FROM platform_settings WHERE id = 1").Scan(&keep)
	rows, err := h.db.QueryContext(ctx, "SELECT id FROM platform_backups WHERE status = 'succeeded' ORDER BY created_at DESC LIMIT 1000 OFFSET ?", keep)
	if err != nil {
		logger.Warn("could not list old platform backups: %v", err)
		return
	}
	var old []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			old = append(old, id)
		}
	}
	rows.Close()
	for _, id := range old {
		if err := h.removePlatformBackup(ctx, id); err != nil {
			logger.Warn("could not prune platform backup %s: %v", id, err)
		}
	}
}

func (h *Handler) removePlatformBackup(ctx context.Context, backupID string) error {
	var serverID, remoteKey string
	if err := h.db.QueryRowContext(ctx, "SELECT server_id, COALESCE(remote_key, '') FROM platform_backups WHERE id = ?", backupID).Scan(&serverID, &remoteKey); err != nil {
		return err
	}
	if _, err := h.executeSSH(ctx, serverID, "rm -f "+platformBackupFile(backupID)); err != nil && remoteKey == "" {
		return err
	}
	if err := h.deleteRemoteBackup(ctx, remoteKey); err != nil {
		return err
	}
	_, err := h.db.ExecContext(ctx, "DELETE FROM platform_backups WHERE id = ?", backupID)
	return err
}

func (h *Handler) readPlatformBackup(w http.ResponseWriter, r *http.Request) (string, string, string, bool) {
	var status, serverID, remoteKey string
	err := h.db.QueryRowContext(r.Context(), "SELECT status, server_id, COALESCE(remote_key, '') FROM platform_backups WHERE id = ?", r.PathValue("backupID")).Scan(&status, &serverID, &remoteKey)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "backup not found")
		return "", "", "", false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read backup")
		return "", "", "", false
	}
	return status, serverID, remoteKey, true
}

func (h *Handler) deletePlatformBackup(w http.ResponseWriter, r *http.Request) {
	if !requireRoot(w, r) {
		return
	}
	status, _, _, ok := h.readPlatformBackup(w, r)
	if !ok {
		return
	}
	if status == "running" {
		writeError(w, http.StatusConflict, "wait for the backup to finish")
		return
	}
	if err := h.removePlatformBackup(r.Context(), r.PathValue("backupID")); err != nil {
		writeError(w, http.StatusBadGateway, "could not delete the backup")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) downloadPlatformBackup(w http.ResponseWriter, r *http.Request) {
	if !requireRoot(w, r) {
		return
	}
	status, serverID, remoteKey, ok := h.readPlatformBackup(w, r)
	if !ok {
		return
	}
	if status != "succeeded" {
		writeError(w, http.StatusConflict, "backup is not finished")
		return
	}
	backupID := r.PathValue("backupID")
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="kymber-`+backupID+`.sql.gz"`)
	h.streamBackupFile(w, r, serverID, platformBackupFile(backupID), remoteKey)
}

// exportEncryptionKey is a POST so the audit log records every export.
func (h *Handler) exportEncryptionKey(w http.ResponseWriter, r *http.Request) {
	if !requireRoot(w, r) {
		return
	}
	key, err := secrets.Key()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "service encryption is not configured")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"key": key})
}

func (h *Handler) startDuePlatformBackup() {
	ctx := context.Background()
	var schedule string
	var nextRunAt time.Time
	err := h.db.QueryRowContext(ctx, "SELECT backup_schedule, backup_next_run_at FROM platform_settings WHERE id = 1 AND backup_schedule <> '' AND backup_next_run_at <= UTC_TIMESTAMP(6)").Scan(&schedule, &nextRunAt)
	if err != nil {
		return
	}
	cron, err := parseCron(schedule)
	if err != nil {
		return
	}
	result, err := h.db.ExecContext(ctx, "UPDATE platform_settings SET backup_next_run_at = ? WHERE id = 1 AND backup_next_run_at = ?", cron.next(time.Now()), nextRunAt)
	if err != nil {
		return
	}
	if claimed, _ := result.RowsAffected(); claimed == 1 {
		if _, err := h.startPlatformBackup(ctx, true); err != nil {
			logger.Warn("scheduled platform backup did not start: %v", err)
		}
	}
}
