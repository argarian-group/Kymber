package services

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/ssh"
	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/logger"
	"kymber/backend/internal/permissions"
)

type serviceJob struct {
	ID           string     `json:"id"`
	Schedule     string     `json:"schedule"`
	Command      string     `json:"command"`
	Enabled      bool       `json:"enabled"`
	NextRunAt    time.Time  `json:"next_run_at"`
	Running      bool       `json:"running"`
	LastRunAt    *time.Time `json:"last_run_at"`
	LastExitCode *int       `json:"last_exit_code"`
	LastOutput   string     `json:"last_output"`
}

type jobInput struct {
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
	Enabled  *bool  `json:"enabled"`
}

const jobTimeout = time.Hour

func (input *jobInput) validate() (cronSchedule, error) {
	input.Schedule, input.Command = strings.TrimSpace(input.Schedule), strings.TrimSpace(input.Command)
	schedule, err := parseCron(input.Schedule)
	if err != nil {
		return schedule, err
	}
	if input.Command == "" || utf8.RuneCountInString(input.Command) > 2000 {
		return schedule, errors.New("command must be 1 to 2000 characters")
	}
	return schedule, nil
}

// jobService authorizes the request and returns the app's ID, or "" after writing an error.
func (h *Handler) jobService(w http.ResponseWriter, r *http.Request) string {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
		return ""
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return ""
	}
	serviceID, ok := pathID(w, r, "serviceID")
	if !ok {
		return ""
	}
	var exists bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM database_services WHERE id = ? AND workspace_id = ? AND project_id = ? AND compose_file IS NULL)", serviceID, workspaceID, projectID).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read service")
		return ""
	}
	if !exists {
		writeError(w, http.StatusNotFound, "service not found")
		return ""
	}
	return serviceID
}

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	serviceID := h.jobService(w, r)
	if serviceID == "" {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, schedule, command, enabled, next_run_at, running_since IS NOT NULL, last_run_at, last_exit_code, COALESCE(last_output, '') FROM service_jobs WHERE service_id = ? ORDER BY created_at", serviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list jobs")
		return
	}
	defer rows.Close()
	jobs := make([]serviceJob, 0)
	for rows.Next() {
		var job serviceJob
		if err := rows.Scan(&job.ID, &job.Schedule, &job.Command, &job.Enabled, &job.NextRunAt, &job.Running, &job.LastRunAt, &job.LastExitCode, &job.LastOutput); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read jobs")
			return
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read jobs")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]serviceJob{"jobs": jobs})
}

func (h *Handler) createJob(w http.ResponseWriter, r *http.Request) {
	serviceID := h.jobService(w, r)
	if serviceID == "" {
		return
	}
	var input jobInput
	if !decodeJSON(w, r, &input) {
		return
	}
	schedule, err := input.validate()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate job id")
		return
	}
	job := serviceJob{ID: id, Schedule: input.Schedule, Command: input.Command, Enabled: true, NextRunAt: schedule.next(time.Now())}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO service_jobs (id, service_id, schedule, command, next_run_at) VALUES (?, ?, ?, ?, ?)", job.ID, serviceID, job.Schedule, job.Command, job.NextRunAt); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create job")
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (h *Handler) updateJob(w http.ResponseWriter, r *http.Request) {
	serviceID := h.jobService(w, r)
	if serviceID == "" {
		return
	}
	var input jobInput
	if !decodeJSON(w, r, &input) {
		return
	}
	schedule, err := input.validate()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	enabled := input.Enabled == nil || *input.Enabled
	result, err := h.db.ExecContext(r.Context(), "UPDATE service_jobs SET schedule = ?, command = ?, enabled = ?, next_run_at = ? WHERE id = ? AND service_id = ?", input.Schedule, input.Command, enabled, schedule.next(time.Now()), r.PathValue("jobID"), serviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update job")
		return
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		var exists bool
		if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM service_jobs WHERE id = ? AND service_id = ?)", r.PathValue("jobID"), serviceID).Scan(&exists); err != nil || !exists {
			writeError(w, http.StatusNotFound, "job not found")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteJob(w http.ResponseWriter, r *http.Request) {
	serviceID := h.jobService(w, r)
	if serviceID == "" {
		return
	}
	result, err := h.db.ExecContext(r.Context(), "DELETE FROM service_jobs WHERE id = ? AND service_id = ?", r.PathValue("jobID"), serviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete job")
		return
	}
	if deleted, _ := result.RowsAffected(); deleted == 0 {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) runJobNow(w http.ResponseWriter, r *http.Request) {
	serviceID := h.jobService(w, r)
	if serviceID == "" {
		return
	}
	jobID := r.PathValue("jobID")
	var command string
	if err := h.db.QueryRowContext(r.Context(), "SELECT command FROM service_jobs WHERE id = ? AND service_id = ?", jobID, serviceID).Scan(&command); err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	result, err := h.db.ExecContext(r.Context(), "UPDATE service_jobs SET running_since = UTC_TIMESTAMP(6) WHERE id = ? AND (running_since IS NULL OR running_since < UTC_TIMESTAMP(6) - INTERVAL 70 MINUTE)", jobID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start job")
		return
	}
	if claimed, _ := result.RowsAffected(); claimed == 0 {
		writeError(w, http.StatusConflict, "job is already running")
		return
	}
	go h.runJob(jobID, serviceID, command)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) runJobScheduler() {
	if _, err := h.db.Exec("UPDATE service_backups SET status = CASE status WHEN 'running' THEN 'failed' ELSE 'succeeded' END, error = 'Interrupted when Kymber restarted.', finished_at = COALESCE(finished_at, UTC_TIMESTAMP(6)) WHERE status IN ('running', 'restoring')"); err != nil {
		logger.Warn("could not close interrupted backups: %v", err)
	}
	if _, err := h.db.Exec("UPDATE platform_backups SET status = 'failed', error = 'Interrupted when Kymber restarted.', finished_at = UTC_TIMESTAMP(6) WHERE status = 'running'"); err != nil {
		logger.Warn("could not close interrupted platform backups: %v", err)
	}
	for range time.Tick(20 * time.Second) {
		h.startDueJobs()
		h.startDueBackups()
		h.startDuePlatformBackup()
	}
}

func (h *Handler) startDueJobs() {
	ctx := context.Background()
	rows, err := h.db.QueryContext(ctx, `SELECT id, service_id, schedule, command, next_run_at FROM service_jobs
		WHERE enabled AND next_run_at <= UTC_TIMESTAMP(6) AND (running_since IS NULL OR running_since < UTC_TIMESTAMP(6) - INTERVAL 70 MINUTE)
		ORDER BY next_run_at LIMIT 50`)
	if err != nil {
		logger.Warn("could not read due jobs: %v", err)
		return
	}
	type dueJob struct {
		id, serviceID, schedule, command string
		nextRunAt                        time.Time
	}
	var due []dueJob
	for rows.Next() {
		var job dueJob
		if err := rows.Scan(&job.id, &job.serviceID, &job.schedule, &job.command, &job.nextRunAt); err != nil {
			logger.Warn("could not read due job: %v", err)
			continue
		}
		due = append(due, job)
	}
	rows.Close()
	for _, job := range due {
		schedule, err := parseCron(job.schedule)
		if err != nil {
			continue
		}
		result, err := h.db.ExecContext(ctx, "UPDATE service_jobs SET running_since = UTC_TIMESTAMP(6), next_run_at = ? WHERE id = ? AND next_run_at = ?", schedule.next(time.Now()), job.id, job.nextRunAt)
		if err != nil {
			logger.Warn("could not claim job %s: %v", job.id, err)
			continue
		}
		if claimed, _ := result.RowsAffected(); claimed == 1 {
			go h.runJob(job.id, job.serviceID, job.command)
		}
	}
}

func (h *Handler) runJob(jobID, serviceID, command string) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout+time.Minute)
	defer cancel()
	output, exitCode := h.execInApp(ctx, serviceID, command)
	if len(output) > 16<<10 {
		output = output[len(output)-16<<10:]
	}
	if _, err := h.db.ExecContext(context.Background(), "UPDATE service_jobs SET running_since = NULL, last_run_at = UTC_TIMESTAMP(6), last_exit_code = ?, last_output = ? WHERE id = ?", exitCode, strings.ToValidUTF8(output, "�"), jobID); err != nil {
		logger.Warn("could not save result of job %s: %v", jobID, err)
	}
}

// execInApp runs a shell command inside the app's container and returns its combined output and exit code.
func (h *Handler) execInApp(ctx context.Context, serviceID, command string) (string, int) {
	var service databaseService
	err := h.db.QueryRowContext(ctx, "SELECT id, engine, name, image, port, data_path, COALESCE(server_id, ''), status FROM database_services WHERE id = ?", serviceID).Scan(&service.ID, &service.Engine, &service.Name, &service.Image, &service.Port, &service.DataPath, &service.ServerID, &service.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return "The app no longer exists.", -1
	}
	if err != nil {
		return "Could not read the app: " + err.Error(), -1
	}
	if service.ServerID == "" || service.Status != "running" {
		return "The app is not running.", -1
	}
	container := serviceTemplate(service).ContainerName(service.Name, service.ID)
	remote := "timeout " + strconv.Itoa(int(jobTimeout.Seconds())) + " docker exec " + shellQuote(container) + " sh -c " + shellQuote(command)
	output, err := h.executeSSH(ctx, service.ServerID, remote)
	var exit *ssh.ExitError
	switch {
	case err == nil:
		return string(output), 0
	case errors.As(err, &exit):
		return string(output), exit.ExitStatus()
	default:
		return err.Error(), -1
	}
}
