package services

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kymber/backend/internal/database"
	"kymber/backend/internal/logger"
)

const maxBackupUpload int64 = 50 << 30

// receivedBackup is an uploaded file stream, gzip-compressed on the way to the server if it was not already.
type receivedBackup struct {
	body   io.Reader
	kind   string
	header []byte
}

// receiveBackup reads the start of an upload to tell what it holds: a PostgreSQL custom dump, a mongodump archive, a tar of files or a text dump.
func receiveBackup(w http.ResponseWriter, r *http.Request) (receivedBackup, bool) {
	if !allowArtifactTransfer(w) {
		return receivedBackup{}, false
	}
	reader := bufio.NewReaderSize(http.MaxBytesReader(w, r.Body, maxBackupUpload), 1<<16)
	peeked, _ := reader.Peek(1 << 16)
	if len(peeked) == 0 {
		writeError(w, http.StatusBadRequest, "the file is empty")
		return receivedBackup{}, false
	}
	compressed := len(peeked) >= 2 && peeked[0] == 0x1f && peeked[1] == 0x8b
	header := peeked
	if compressed {
		gz, err := gzip.NewReader(bytes.NewReader(peeked))
		if err != nil {
			writeError(w, http.StatusBadRequest, "the file is not a valid gzip archive")
			return receivedBackup{}, false
		}
		header, _ = io.ReadAll(io.LimitReader(gz, 1024))
	}
	if len(header) > 1024 {
		header = header[:1024]
	}
	received := receivedBackup{body: reader, kind: "text", header: header}
	switch {
	case bytes.HasPrefix(header, []byte("PGDMP")):
		received.kind = "pgdmp"
	case bytes.HasPrefix(header, []byte{0x6d, 0xe2, 0x99, 0x81}):
		received.kind = "mongo"
	case len(header) >= 262 && string(header[257:262]) == "ustar":
		received.kind = "tar"
	}
	if !compressed {
		pipeReader, pipeWriter := io.Pipe()
		go func() {
			gz := gzip.NewWriter(pipeWriter)
			_, err := io.Copy(gz, reader)
			if closeErr := gz.Close(); err == nil {
				err = closeErr
			}
			pipeWriter.CloseWithError(err)
		}()
		received.body = pipeReader
	}
	return received, true
}

// firstTarEntry returns the first path in a tar header, without a leading "./".
func firstTarEntry(header []byte) string {
	name, _, _ := bytes.Cut(header[:min(100, len(header))], []byte{0})
	return strings.TrimPrefix(string(name), "./")
}

func underPath(entry, mount string) bool {
	mount = strings.Trim(mount, "/")
	return entry == mount || strings.HasPrefix(entry, mount+"/")
}

// appBackupFormat decides how an uploaded file is restored into this app, or explains why it does not fit.
func appBackupFormat(t backupTarget, received receivedBackup, chosen string) (format, restorePath string, err error) {
	switch t.service.Engine {
	case "postgres", "timescale":
		if received.kind == "pgdmp" {
			return "pgdmp", "", nil
		}
		if received.kind != "text" {
			return "", "", errors.New("upload a PostgreSQL dump: SQL text or pg_dump's custom format")
		}
		return "tar", "", nil
	case "mysql", "mariadb", "percona":
		if received.kind != "text" {
			return "", "", errors.New("upload a SQL dump made with mysqldump or mariadb-dump")
		}
		return "tar", "", nil
	case "mongodb":
		if received.kind != "mongo" {
			return "", "", errors.New("upload an archive made with mongodump --archive")
		}
		return "tar", "", nil
	}
	if received.kind != "tar" {
		return "", "", errors.New("upload a .tar or .tar.gz archive of the files")
	}
	if t.service.Compose {
		return "tar", "", nil
	}
	if chosen != "" {
		for _, p := range t.paths {
			if p == chosen {
				return "dir", p, nil
			}
		}
		return "", "", errors.New("choose one of the app's volumes")
	}
	entry := firstTarEntry(received.header)
	for _, p := range t.paths {
		if underPath(entry, p) {
			return "tar", "", nil
		}
	}
	if t.dataPath != "" && underPath(entry, path.Base(t.dataPath)) {
		return "cp", "", nil
	}
	if len(t.paths) == 1 {
		return "dir", t.paths[0], nil
	}
	return "", "", errors.New("choose which volume receives these files")
}

// storeUpload streams an upload to a backup file on the server and returns its size.
func (h *Handler) storeUpload(ctx context.Context, serverID, dir, file string, body io.Reader) (int64, error) {
	output, err := h.executeSSHStream(ctx, serverID, "mkdir -p "+dir+" && cat > "+file+".part && gunzip -t "+file+".part && mv "+file+".part "+file+" && wc -c < "+file+" || { rm -f "+file+".part; exit 1; }", body)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
}

func (h *Handler) uploadAppBackup(w http.ResponseWriter, r *http.Request) {
	serviceID, ok := h.backupService(w, r)
	if !ok {
		return
	}
	target, err := h.loadBackupTarget(r.Context(), serviceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	received, ok := receiveBackup(w, r)
	if !ok {
		return
	}
	format, restorePath, err := appBackupFormat(target, received, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create backup")
		return
	}
	var storedPath any
	if restorePath != "" {
		storedPath = restorePath
	}
	result, err := h.db.ExecContext(r.Context(), `INSERT INTO service_backups (id, service_id, server_id, uploaded, format, restore_path) SELECT ?, ?, ?, TRUE, ?, ? FROM DUAL
		WHERE NOT EXISTS (SELECT 1 FROM service_backups WHERE service_id = ? AND status IN ('running', 'restoring') AND created_at > UTC_TIMESTAMP(6) - INTERVAL 7 HOUR)`, id, serviceID, target.service.ServerID, format, storedPath, serviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create backup")
		return
	}
	if inserted, _ := result.RowsAffected(); inserted == 0 {
		writeError(w, http.StatusConflict, "wait for the running backup or restore to finish")
		return
	}
	size, err := h.storeUpload(r.Context(), target.service.ServerID, backupDir+serviceID, backupFile(serviceID, id), received.body)
	if err != nil {
		h.finishBackup(id, "failed", 0, "Upload failed: "+err.Error())
		writeError(w, http.StatusBadGateway, "could not save the file on the server")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), backupTimeout)
		defer cancel()
		message := ""
		key, err := h.uploadBackup(ctx, target.service.ServerID, backupFile(serviceID, id), size, "apps", serviceID, id+".gz")
		if err != nil {
			message = "Saved on the server, but the copy to S3 failed: " + err.Error()
		} else if key != "" {
			_, _ = h.db.ExecContext(ctx, "UPDATE service_backups SET remote_key = ? WHERE id = ?", key, id)
		}
		h.finishBackup(id, "succeeded", size, message)
	}()
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) uploadPlatformBackup(w http.ResponseWriter, r *http.Request) {
	if !requireRoot(w, r) {
		return
	}
	serverID, err := h.localServerID(r.Context())
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	received, ok := receiveBackup(w, r)
	if !ok {
		return
	}
	if received.kind != "text" || !bytes.Contains(received.header, []byte(" dump")) {
		writeError(w, http.StatusBadRequest, "upload a Kymber platform backup (.sql.gz)")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create backup")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO platform_backups (id, server_id, uploaded) VALUES (?, ?, TRUE)", id, serverID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create backup")
		return
	}
	size, err := h.storeUpload(r.Context(), serverID, platformBackupDir, platformBackupFile(id), received.body)
	finish := func(status string, size int64, key, message string) {
		var remoteKey, errorText any
		if key != "" {
			remoteKey = key
		}
		if message != "" {
			errorText = message
		}
		if _, err := h.db.ExecContext(context.Background(), "UPDATE platform_backups SET status = ?, size_bytes = ?, remote_key = ?, error = ?, finished_at = UTC_TIMESTAMP(6) WHERE id = ?", status, size, remoteKey, errorText, id); err != nil {
			logger.Warn("could not save platform backup %s: %v", id, err)
		}
	}
	if err != nil {
		finish("failed", 0, "", "Upload failed: "+err.Error())
		writeError(w, http.StatusBadGateway, "could not save the file on the server")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), backupTimeout)
		defer cancel()
		key, err := h.uploadBackup(ctx, serverID, platformBackupFile(id), size, "platform", id+".sql.gz")
		message := ""
		if err != nil {
			message = "Saved on the server, but the copy to S3 failed: " + err.Error()
		}
		finish("succeeded", size, key, message)
	}()
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// restorePlatformBackup replaces the panel database with a backup. A key from another installation replaces this one's, so the backup's secrets stay readable.
func (h *Handler) restorePlatformBackup(w http.ResponseWriter, r *http.Request) {
	if !requireRoot(w, r) {
		return
	}
	var input struct {
		Key string `json:"key"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Key = strings.TrimSpace(input.Key)
	keyFile := os.Getenv("SERVICE_ENCRYPTION_KEY_FILE")
	if input.Key != "" {
		if decoded, err := hex.DecodeString(input.Key); err != nil || len(decoded) != 32 {
			writeError(w, http.StatusBadRequest, "the recovery key must be the 64-character key downloaded from Kymber")
			return
		}
		if keyFile == "" {
			writeError(w, http.StatusConflict, "this installation reads its key from the environment; change SERVICE_ENCRYPTION_KEY instead")
			return
		}
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
	var row struct {
		scheduled, uploaded   bool
		size                  int64
		createdAt, finishedAt any
	}
	if err := h.db.QueryRowContext(r.Context(), "SELECT scheduled, uploaded, size_bytes, created_at, finished_at FROM platform_backups WHERE id = ?", backupID).Scan(&row.scheduled, &row.uploaded, &row.size, &row.createdAt, &row.finishedAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read backup")
		return
	}
	var storedKey any
	if remoteKey != "" {
		storedKey = remoteKey
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), backupTimeout)
	defer cancel()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(backupTimeout))
	if err := h.fetchBackup(ctx, serverID, platformBackupDir, platformBackupFile(backupID), remoteKey); err != nil {
		writeError(w, http.StatusBadGateway, "could not copy the backup from S3: "+err.Error())
		return
	}
	file := platformBackupFile(backupID)
	command := "set -eu; test -f " + file + "; c=$(docker ps -q --filter label=com.docker.compose.project=" + shellQuote(composeProjectName()) + " --filter label=com.docker.compose.service=mariadb | head -n 1); [ -n \"$c\" ] || { echo 'could not find the Kymber database container' >&2; exit 1; }; gunzip -c " + file + ` | docker exec -i "$c" sh -c 'exec mariadb -uroot -p"$MARIADB_ROOT_PASSWORD"'`
	if _, err := h.executeSSH(ctx, serverID, command); err != nil {
		writeError(w, http.StatusBadGateway, "restore failed: "+err.Error())
		return
	}
	if input.Key != "" {
		if err := replaceKeyFile(keyFile, input.Key); err != nil {
			writeError(w, http.StatusInternalServerError, "the database was restored, but the recovery key could not be saved: "+err.Error())
			return
		}
	}
	if err := database.Migrate(h.db); err != nil {
		writeError(w, http.StatusInternalServerError, "the database was restored, but updating it failed: "+err.Error())
		return
	}
	// The dump holds its own row as still running, and an uploaded backup is not in it at all.
	if _, err := h.db.ExecContext(ctx, `INSERT INTO platform_backups (id, server_id, status, scheduled, uploaded, size_bytes, remote_key, created_at, finished_at) VALUES (?, ?, 'succeeded', ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE server_id = VALUES(server_id), status = 'succeeded', uploaded = VALUES(uploaded), size_bytes = VALUES(size_bytes), remote_key = VALUES(remote_key), error = NULL, finished_at = VALUES(finished_at)`,
		backupID, serverID, row.scheduled, row.uploaded, row.size, storedKey, row.createdAt, row.finishedAt); err != nil {
		logger.Warn("could not keep restored platform backup %s: %v", backupID, err)
	}
	h.settleRestoredPlatformBackups(ctx)
	w.WriteHeader(http.StatusNoContent)
}

// settleRestoredPlatformBackups closes rows the dump captured mid-run: the backup that made the dump finished, so its file is there.
func (h *Handler) settleRestoredPlatformBackups(ctx context.Context) {
	rows, err := h.db.QueryContext(ctx, "SELECT id, server_id FROM platform_backups WHERE status = 'running'")
	if err != nil {
		logger.Warn("could not list restored platform backups: %v", err)
		return
	}
	var running [][2]string
	for rows.Next() {
		var id, serverID string
		if rows.Scan(&id, &serverID) == nil {
			running = append(running, [2]string{id, serverID})
		}
	}
	rows.Close()
	for _, backup := range running {
		output, err := h.executeSSH(ctx, backup[1], "wc -c < "+platformBackupFile(backup[0]))
		size, parseErr := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
		if err != nil || parseErr != nil {
			_, err = h.db.ExecContext(ctx, "UPDATE platform_backups SET status = 'failed', error = 'Interrupted by a restore.', finished_at = UTC_TIMESTAMP(6) WHERE id = ?", backup[0])
		} else {
			_, err = h.db.ExecContext(ctx, "UPDATE platform_backups SET status = 'succeeded', size_bytes = ?, finished_at = UTC_TIMESTAMP(6) WHERE id = ?", size, backup[0])
		}
		if err != nil {
			logger.Warn("could not settle restored platform backup %s: %v", backup[0], err)
		}
	}
}

func replaceKeyFile(keyFile, key string) error {
	temp, err := os.CreateTemp(filepath.Dir(keyFile), ".service-encryption-key-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.WriteString(strings.ToLower(key)); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(0o400); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), keyFile)
}
