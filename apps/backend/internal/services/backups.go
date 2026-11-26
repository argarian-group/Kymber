package services

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/backupstore"
	"kymber/backend/internal/database"
	"kymber/backend/internal/logger"
	"kymber/backend/internal/permissions"
)

type serviceBackup struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"`
	Scheduled  bool       `json:"scheduled"`
	SizeBytes  int64      `json:"size_bytes"`
	Error      string     `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Remote     bool       `json:"remote"`
	format     string
	remoteKey  string
}

// backupMethod says how an engine is dumped and restored. Engines without one fall back to copying the data directory.
type backupMethod struct{ dump, restore string }

var backupMethods = map[string]backupMethod{
	"mysql":     {`mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --all-databases --single-transaction --routines --events`, `mysql -uroot -p"$MYSQL_ROOT_PASSWORD"`},
	"percona":   {`mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --all-databases --single-transaction --routines --events`, `mysql -uroot -p"$MYSQL_ROOT_PASSWORD"`},
	"mariadb":   {`mariadb-dump -uroot -p"$MARIADB_ROOT_PASSWORD" --all-databases --single-transaction --routines --events`, `mariadb -uroot -p"$MARIADB_ROOT_PASSWORD"`},
	"postgres":  {`pg_dumpall -U "$POSTGRES_USER" --clean --if-exists`, `psql -q -U "$POSTGRES_USER" -d postgres`},
	"timescale": {`pg_dumpall -U "$POSTGRES_USER" --clean --if-exists`, `psql -q -U "$POSTGRES_USER" -d postgres`},
	"mongodb":   {`mongodump --archive -u "$MONGO_INITDB_ROOT_USERNAME" -p "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin`, `mongorestore --archive --drop -u "$MONGO_INITDB_ROOT_USERNAME" -p "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin`},
}

const backupDir = `"$HOME/.local/share/kymber/backups/"`
const backupTimeout = 6 * time.Hour

type backupTarget struct {
	service   databaseService
	container string
	dataPath  string
	paths     []string
}

const backupImage = "busybox:1.37.0"

func (h *Handler) loadBackupTarget(ctx context.Context, serviceID string) (backupTarget, error) {
	var target backupTarget
	s := &target.service
	var compose bool
	err := h.db.QueryRowContext(ctx, "SELECT id, engine, name, image, port, data_path, COALESCE(server_id, ''), status, compose_file IS NOT NULL, COALESCE(compose_project, name) FROM database_services WHERE id = ?", serviceID).Scan(&s.ID, &s.Engine, &s.Name, &s.Image, &s.Port, &s.DataPath, &s.ServerID, &s.Status, &compose, &s.ComposeProject)
	if err != nil {
		return target, err
	}
	s.Compose = compose
	template := serviceTemplate(*s)
	target.container, target.dataPath = template.ContainerName(s.Name, s.ID), template.DataPath
	if !compose {
		if target.dataPath != "" {
			target.paths = append(target.paths, target.dataPath)
		}
		storages, err := h.readStorages(ctx, serviceID, false)
		if err != nil {
			return target, err
		}
		for _, storage := range storages {
			if storage.Kind == "volume" && storage.MountPath != target.dataPath {
				target.paths = append(target.paths, storage.MountPath)
			}
		}
	}
	_, logical := backupMethods[s.Engine]
	switch {
	case !compose && !logical && len(target.paths) == 0:
		return target, errors.New("this app has no volumes to back up; add one in Storage")
	case s.ServerID == "":
		return target, errors.New("deploy the app before backing it up")
	}
	return target, nil
}

func backupFile(serviceID, backupID string) string {
	return backupDir + serviceID + "/" + backupID + ".gz"
}

// Compose volume names come from Docker and only use [a-zA-Z0-9_.-], so they are safe unquoted.
func composeVolumeMounts(project string) string {
	return "vols=$(docker volume ls -q --filter label=com.docker.compose.project=" + shellQuote(project) + "); [ -n \"$vols\" ] || { echo 'this stack has no named volumes to back up' >&2; exit 1; }; mounts=; for v in $vols; do mounts=\"$mounts -v $v:/v/$v\"; done; "
}

func volumePaths(paths []string) string {
	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = shellQuote(strings.TrimPrefix(p, "/"))
	}
	return strings.Join(quoted, " ")
}

func backupCommand(t backupTarget, backupID string) string {
	prepare := ""
	source := "docker run --rm --volumes-from " + shellQuote(t.container) + ":ro " + backupImage + " tar -cf - -C / " + volumePaths(t.paths)
	if method, ok := backupMethods[t.service.Engine]; ok {
		source = "docker exec " + shellQuote(t.container) + " sh -c " + shellQuote(method.dump)
	} else if t.service.Compose {
		prepare = composeVolumeMounts(t.service.ComposeProject)
		source = "docker run --rm $mounts " + backupImage + " tar -cf - -C /v ."
	}
	base := backupDir + t.service.ID + "/" + backupID
	part, code, stderr := base+".part", base+".code", base+".err"
	return "set -u; " + prepare + "mkdir -p " + backupDir + t.service.ID + " || exit 1; { " + source + " 2>" + stderr + "; echo $? >" + code + "; } | gzip >" + part +
		"; if [ \"$(cat " + code + " 2>/dev/null)\" != 0 ]; then cat " + stderr + " >&2; rm -f " + part + " " + code + " " + stderr + "; exit 1; fi; rm -f " + code + " " + stderr +
		"; mv " + part + " " + backupFile(t.service.ID, backupID) + " && wc -c <" + backupFile(t.service.ID, backupID)
}

func restoreCommand(t backupTarget, backupID, format string) string {
	file := backupFile(t.service.ID, backupID)
	if method, ok := backupMethods[t.service.Engine]; ok {
		return "set -eu; test -f " + file + "; gunzip -c " + file + " | docker exec -i " + shellQuote(t.container) + " sh -c " + shellQuote(method.restore)
	}
	if t.service.Compose {
		project := "label=com.docker.compose.project=" + shellQuote(t.service.ComposeProject)
		return "set -eu; test -f " + file + "; " + composeVolumeMounts(t.service.ComposeProject) + "running=$(docker ps -q --filter " + project + "); [ -z \"$running\" ] || docker stop $running >/dev/null; status=0; gunzip -c " + file + " | docker run --rm -i $mounts " + backupImage + " tar -xf - -C /v || status=$?; [ -z \"$running\" ] || docker start $running >/dev/null; exit $status"
	}
	container := shellQuote(t.container)
	extract := "docker run --rm -i --volumes-from " + container + " " + backupImage + " tar -xf - -C /"
	if format == "cp" {
		extract = "docker cp - " + shellQuote(t.container+":"+path.Dir(t.dataPath))
	}
	return "set -eu; test -f " + file + "; docker stop " + container + " >/dev/null; status=0; gunzip -c " + file + " | " + extract + " || status=$?; docker start " + container + " >/dev/null; exit $status"
}

func (h *Handler) backupService(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
		return "", false
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return "", false
	}
	serviceID, ok := pathID(w, r, "serviceID")
	if !ok {
		return "", false
	}
	var exists bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM database_services WHERE id = ? AND workspace_id = ? AND project_id = ?)", serviceID, workspaceID, projectID).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read service")
		return "", false
	}
	if !exists {
		writeError(w, http.StatusNotFound, "service not found")
		return "", false
	}
	return serviceID, true
}

func (h *Handler) listBackups(w http.ResponseWriter, r *http.Request) {
	serviceID, ok := h.backupService(w, r)
	if !ok {
		return
	}
	var settings struct {
		Schedule  string     `json:"schedule"`
		Keep      int        `json:"keep"`
		NextRunAt *time.Time `json:"next_run_at"`
		Supported bool       `json:"supported"`
		Logical   bool       `json:"logical"`
		Reason    string     `json:"reason,omitempty"`
	}
	if err := h.db.QueryRowContext(r.Context(), "SELECT backup_schedule, backup_keep, backup_next_run_at FROM database_services WHERE id = ?", serviceID).Scan(&settings.Schedule, &settings.Keep, &settings.NextRunAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read backup settings")
		return
	}
	target, err := h.loadBackupTarget(r.Context(), serviceID)
	settings.Supported = err == nil
	if err != nil {
		settings.Reason = err.Error()
	}
	_, settings.Logical = backupMethods[target.service.Engine]
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, status, scheduled, size_bytes, COALESCE(error, ''), created_at, finished_at, remote_key IS NOT NULL FROM service_backups WHERE service_id = ? ORDER BY created_at DESC LIMIT 100", serviceID)
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

func (h *Handler) updateBackupSettings(w http.ResponseWriter, r *http.Request) {
	serviceID, ok := h.backupService(w, r)
	if !ok {
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
	if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET backup_schedule = ?, backup_keep = ?, backup_next_run_at = ? WHERE id = ?", input.Schedule, input.Keep, nextRun, serviceID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save backup settings")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createBackup(w http.ResponseWriter, r *http.Request) {
	serviceID, ok := h.backupService(w, r)
	if !ok {
		return
	}
	id, err := h.startBackup(r.Context(), serviceID, false)
	if errors.Is(err, errServiceActionInProgress) {
		writeError(w, http.StatusConflict, "a backup or restore is already running")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id})
}

func (h *Handler) startBackup(ctx context.Context, serviceID string, scheduled bool) (string, error) {
	target, err := h.loadBackupTarget(ctx, serviceID)
	if err != nil {
		return "", err
	}
	id, err := database.NewShortID()
	if err != nil {
		return "", err
	}
	result, err := h.db.ExecContext(ctx, `INSERT INTO service_backups (id, service_id, server_id, scheduled) SELECT ?, ?, ?, ? FROM DUAL
		WHERE NOT EXISTS (SELECT 1 FROM service_backups WHERE service_id = ? AND status IN ('running', 'restoring') AND created_at > UTC_TIMESTAMP(6) - INTERVAL 7 HOUR)`, id, serviceID, target.service.ServerID, scheduled, serviceID)
	if err != nil {
		return "", err
	}
	if inserted, _ := result.RowsAffected(); inserted == 0 {
		return "", errServiceActionInProgress
	}
	go h.runBackup(target, id)
	return id, nil
}

func (h *Handler) runBackup(target backupTarget, backupID string) {
	ctx, cancel := context.WithTimeout(context.Background(), backupTimeout)
	defer cancel()
	output, err := h.executeSSH(ctx, target.service.ServerID, backupCommand(target, backupID))
	if err != nil {
		h.finishBackup(backupID, "failed", 0, err.Error())
		return
	}
	size, _ := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	message := ""
	key, err := h.uploadBackup(ctx, target.service.ServerID, backupFile(target.service.ID, backupID), size, "apps", target.service.ID, backupID+".gz")
	if err != nil {
		message = "Saved on the server, but the copy to S3 failed: " + err.Error()
	} else if key != "" {
		if _, err := h.db.ExecContext(ctx, "UPDATE service_backups SET remote_key = ? WHERE id = ?", key, backupID); err != nil {
			logger.Warn("could not save remote key of backup %s: %v", backupID, err)
		}
	}
	h.finishBackup(backupID, "succeeded", size, message)
	h.pruneBackups(ctx, target.service.ID, target.service.ServerID)
}

// uploadBackup copies a finished backup from its server to the S3 destination and returns the object key, or "" when none is set.
func (h *Handler) uploadBackup(ctx context.Context, serverID, file string, size int64, keyParts ...string) (string, error) {
	store, err := backupstore.Load(ctx, h.db)
	if store == nil || err != nil {
		return "", err
	}
	connection, err := h.connectSSH(ctx, serverID)
	if err != nil {
		return "", err
	}
	defer connection.Close()
	session, err := connection.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	stdout, err := session.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := session.Start("cat " + file); err != nil {
		return "", err
	}
	key := store.Key(keyParts...)
	if err := store.Put(ctx, key, stdout, size); err != nil {
		return "", err
	}
	return key, session.Wait()
}

// fetchBackup puts a backup back on the server from S3 when its local copy is gone, for example after moving to a new server.
func (h *Handler) fetchBackup(ctx context.Context, serverID, dir, file, remoteKey string) error {
	if _, err := h.executeSSH(ctx, serverID, "test -f "+file); err == nil || remoteKey == "" {
		return nil
	}
	store, err := backupstore.Load(ctx, h.db)
	if err != nil {
		return err
	}
	if store == nil {
		return errors.New("this backup is only in S3, which is no longer configured")
	}
	object, err := store.Get(ctx, remoteKey)
	if err != nil {
		return err
	}
	defer object.Close()
	_, err = h.executeSSHStream(ctx, serverID, "mkdir -p "+dir+" && cat > "+file+".part && mv "+file+".part "+file, object)
	return err
}

func (h *Handler) deleteRemoteBackup(ctx context.Context, remoteKey string) error {
	if remoteKey == "" {
		return nil
	}
	store, err := backupstore.Load(ctx, h.db)
	if store == nil || err != nil {
		return err
	}
	return store.Delete(ctx, remoteKey)
}

func (h *Handler) finishBackup(backupID, status string, size int64, message string) {
	var errorText any
	if message != "" {
		errorText = message
	}
	if _, err := h.db.ExecContext(context.Background(), "UPDATE service_backups SET status = ?, size_bytes = ?, error = ?, finished_at = UTC_TIMESTAMP(6) WHERE id = ?", status, size, errorText, backupID); err != nil {
		logger.Warn("could not save backup %s: %v", backupID, err)
	}
}

func (h *Handler) pruneBackups(ctx context.Context, serviceID, serverID string) {
	rows, err := h.db.QueryContext(ctx, "SELECT id FROM service_backups WHERE service_id = ? AND status = 'succeeded' ORDER BY created_at DESC LIMIT 1000 OFFSET ?", serviceID, h.backupKeep(ctx, serviceID))
	if err != nil {
		logger.Warn("could not list old backups: %v", err)
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
		if err := h.removeBackup(ctx, serviceID, serverID, id); err != nil {
			logger.Warn("could not prune backup %s: %v", id, err)
		}
	}
}

func (h *Handler) backupKeep(ctx context.Context, serviceID string) int {
	keep := 7
	_ = h.db.QueryRowContext(ctx, "SELECT backup_keep FROM database_services WHERE id = ?", serviceID).Scan(&keep)
	return keep
}

func (h *Handler) removeBackup(ctx context.Context, serviceID, serverID, backupID string) error {
	var remoteKey string
	if err := h.db.QueryRowContext(ctx, "SELECT COALESCE(remote_key, '') FROM service_backups WHERE id = ?", backupID).Scan(&remoteKey); err != nil {
		return err
	}
	if _, err := h.executeSSH(ctx, serverID, "rm -f "+backupFile(serviceID, backupID)); err != nil && remoteKey == "" {
		return err
	}
	if err := h.deleteRemoteBackup(ctx, remoteKey); err != nil {
		return err
	}
	_, err := h.db.ExecContext(ctx, "DELETE FROM service_backups WHERE id = ?", backupID)
	return err
}

func (h *Handler) readBackup(w http.ResponseWriter, r *http.Request, serviceID string) (serviceBackup, string, bool) {
	var backup serviceBackup
	var serverID string
	err := h.db.QueryRowContext(r.Context(), "SELECT id, status, server_id, format, COALESCE(remote_key, '') FROM service_backups WHERE id = ? AND service_id = ?", r.PathValue("backupID"), serviceID).Scan(&backup.ID, &backup.Status, &serverID, &backup.format, &backup.remoteKey)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "backup not found")
		return backup, "", false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read backup")
		return backup, "", false
	}
	return backup, serverID, true
}

func (h *Handler) deleteBackup(w http.ResponseWriter, r *http.Request) {
	serviceID, ok := h.backupService(w, r)
	if !ok {
		return
	}
	backup, serverID, ok := h.readBackup(w, r, serviceID)
	if !ok {
		return
	}
	if backup.Status == "running" || backup.Status == "restoring" {
		writeError(w, http.StatusConflict, "wait for the backup to finish")
		return
	}
	if err := h.removeBackup(r.Context(), serviceID, serverID, backup.ID); err != nil {
		writeError(w, http.StatusBadGateway, "could not delete backup: "+err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) restoreBackup(w http.ResponseWriter, r *http.Request) {
	serviceID, ok := h.backupService(w, r)
	if !ok {
		return
	}
	backup, serverID, ok := h.readBackup(w, r, serviceID)
	if !ok {
		return
	}
	target, err := h.loadBackupTarget(r.Context(), serviceID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if target.service.ServerID != serverID && backup.remoteKey == "" {
		writeError(w, http.StatusConflict, "this backup is stored on another server")
		return
	}
	if target.service.Status != "running" {
		writeError(w, http.StatusConflict, "start the app before restoring")
		return
	}
	result, err := h.db.ExecContext(r.Context(), `UPDATE service_backups SET status = 'restoring', error = NULL WHERE id = ? AND status = 'succeeded'
		AND NOT EXISTS (SELECT 1 FROM (SELECT id FROM service_backups WHERE service_id = ? AND status IN ('running', 'restoring') AND created_at > UTC_TIMESTAMP(6) - INTERVAL 7 HOUR) busy)`, backup.ID, serviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start restore")
		return
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		writeError(w, http.StatusConflict, "only finished backups can be restored, one at a time")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), backupTimeout)
		defer cancel()
		message := ""
		serverID := target.service.ServerID
		if err := h.fetchBackup(ctx, serverID, backupDir+serviceID, backupFile(serviceID, backup.ID), backup.remoteKey); err != nil {
			message = "Could not copy the backup from S3: " + err.Error()
		} else if _, err := h.executeSSH(ctx, serverID, restoreCommand(target, backup.ID, backup.format)); err != nil {
			message = "Restore failed: " + err.Error()
		}
		var errorText any
		if message != "" {
			errorText = message
		}
		if _, err := h.db.ExecContext(context.Background(), "UPDATE service_backups SET status = 'succeeded', error = ?, server_id = IF(? = '', server_id, ?) WHERE id = ?", errorText, message, serverID, backup.ID); err != nil {
			logger.Warn("could not save restore of backup %s: %v", backup.ID, err)
		}
	}()
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) downloadBackup(w http.ResponseWriter, r *http.Request) {
	serviceID, ok := h.backupService(w, r)
	if !ok {
		return
	}
	backup, serverID, ok := h.readBackup(w, r, serviceID)
	if !ok {
		return
	}
	if backup.Status != "succeeded" {
		writeError(w, http.StatusConflict, "backup is not finished")
		return
	}
	var name, engine string
	if err := h.db.QueryRowContext(r.Context(), "SELECT name, engine FROM database_services WHERE id = ?", serviceID).Scan(&name, &engine); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read service")
		return
	}
	extension := ".tar.gz"
	if _, ok := backupMethods[engine]; ok {
		extension = ".sql.gz"
		if engine == "mongodb" {
			extension = ".archive.gz"
		}
	}
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+"-"+backup.ID+extension+`"`)
	if backup.remoteKey != "" {
		if _, err := h.executeSSH(r.Context(), serverID, "test -f "+backupFile(serviceID, backup.ID)); err != nil {
			h.downloadRemoteBackup(w, r, backup.remoteKey)
			return
		}
	}
	connection, err := h.connectSSH(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not reach the server")
		return
	}
	defer connection.Close()
	session, err := connection.NewSession()
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not reach the server")
		return
	}
	defer session.Close()
	controller := http.NewResponseController(w)
	_ = controller.SetWriteDeadline(time.Now().Add(2 * time.Hour))
	session.Stdout = w
	if err := session.Run("cat " + backupFile(serviceID, backup.ID)); err != nil {
		logger.Warn("backup download %s failed: %v", backup.ID, err)
	}
}

func (h *Handler) downloadRemoteBackup(w http.ResponseWriter, r *http.Request, remoteKey string) {
	store, err := backupstore.Load(r.Context(), h.db)
	if err == nil && store == nil {
		err = errors.New("S3 is no longer configured")
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not read the backup from S3")
		return
	}
	object, err := store.Get(r.Context(), remoteKey)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not read the backup from S3")
		return
	}
	defer object.Close()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(2 * time.Hour))
	if _, err := io.Copy(w, object); err != nil {
		logger.Warn("backup download from S3 failed: %v", err)
	}
}

func (h *Handler) startDueBackups() {
	ctx := context.Background()
	rows, err := h.db.QueryContext(ctx, "SELECT id, backup_schedule, backup_next_run_at FROM database_services WHERE backup_schedule <> '' AND backup_next_run_at <= UTC_TIMESTAMP(6) LIMIT 50")
	if err != nil {
		logger.Warn("could not read due backups: %v", err)
		return
	}
	type due struct {
		id, schedule string
		nextRunAt    time.Time
	}
	var services []due
	for rows.Next() {
		var service due
		if rows.Scan(&service.id, &service.schedule, &service.nextRunAt) == nil {
			services = append(services, service)
		}
	}
	rows.Close()
	for _, service := range services {
		schedule, err := parseCron(service.schedule)
		if err != nil {
			continue
		}
		result, err := h.db.ExecContext(ctx, "UPDATE database_services SET backup_next_run_at = ? WHERE id = ? AND backup_next_run_at = ?", schedule.next(time.Now()), service.id, service.nextRunAt)
		if err != nil {
			continue
		}
		if claimed, _ := result.RowsAffected(); claimed == 1 {
			if _, err := h.startBackup(ctx, service.id, true); err != nil {
				logger.Warn("scheduled backup of %s did not start: %v", service.id, err)
			}
		}
	}
}
