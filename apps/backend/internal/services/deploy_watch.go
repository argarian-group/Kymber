package services

import (
	"context"
	"strings"
	"time"

	"kymber/backend/internal/logger"
)

const (
	deployWatchInterval = 3 * time.Second
	deployWatchLimit    = 2 * time.Hour
)

type deployProgress struct {
	done, succeeded, logExists bool
	log                        string
}

func deployProgressCommand(serviceID string) string {
	logFile, statusFile := deploymentLogFiles(serviceID)
	return "if test -f " + shellQuote(statusFile) + "; then printf '__KYMBER_EXIT__'; cat " + shellQuote(statusFile) + "; printf '\\n'; tail -c 65536 " + shellQuote(logFile) + " 2>/dev/null; elif test -f " + shellQuote(logFile) + "; then printf '__KYMBER_RUNNING__'; else printf '__KYMBER_NO_LOG__'; fi"
}

func parseDeployProgress(raw string) deployProgress {
	if strings.HasPrefix(raw, "__KYMBER_NO_LOG__") {
		return deployProgress{}
	}
	rest, finished := strings.CutPrefix(raw, "__KYMBER_EXIT__")
	if !finished {
		return deployProgress{logExists: true}
	}
	code, log, _ := strings.Cut(rest, "\n")
	return deployProgress{done: true, succeeded: strings.TrimSpace(code) == "0", logExists: true, log: strings.TrimSpace(log)}
}

// finishDeploy records a deploy's result once. Only the deploy that just finished applied the storages and ports, so later reads must not mark new ones.
func (h *Handler) finishDeploy(ctx context.Context, serviceID string, succeeded bool, output string, deploymentIDs ...string) {
	deploymentID := ""
	if len(deploymentIDs) > 0 {
		deploymentID = deploymentIDs[0]
	}
	if len(output) > 65536 {
		output = output[len(output)-65536:]
	}
	status := "failed"
	var lastError any = output
	if succeeded {
		status = "succeeded"
		lastError = nil
	}
	if !succeeded {
		_, _ = h.db.ExecContext(ctx, "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ? AND status = 'deploying'", output, serviceID)
	} else {
		result, err := h.db.ExecContext(ctx, "UPDATE database_services SET status = 'running', last_error = NULL WHERE id = ? AND status = 'deploying'", serviceID)
		if err != nil {
			return
		}
		if changed, _ := result.RowsAffected(); changed > 0 {
			_, _ = h.db.ExecContext(ctx, "UPDATE service_storages SET applied = TRUE WHERE service_id = ?", serviceID)
			_, _ = h.db.ExecContext(ctx, "UPDATE service_ports SET applied = TRUE WHERE service_id = ?", serviceID)
		}
	}
	if deploymentID != "" {
		_, _ = h.db.ExecContext(ctx, "UPDATE service_deployments SET status = ?, output = ?, last_error = ?, finished_at = UTC_TIMESTAMP(6) WHERE id = ? AND status = 'deploying'", status, output, lastError, deploymentID)
	} else {
		_, _ = h.db.ExecContext(ctx, "UPDATE service_deployments SET status = ?, output = ?, last_error = ?, finished_at = UTC_TIMESTAMP(6) WHERE service_id = ? AND status = 'deploying' ORDER BY started_at DESC LIMIT 1", status, output, lastError, serviceID)
	}
}

func (h *Handler) failDeployment(ctx context.Context, serviceID, deploymentID, message string) {
	_, _ = h.db.ExecContext(ctx, "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ? AND status = 'deploying'", message, serviceID)
	_, _ = h.db.ExecContext(ctx, "UPDATE service_deployments SET status = 'failed', last_error = ?, output = ?, finished_at = UTC_TIMESTAMP(6) WHERE id = ? AND status = 'deploying'", message, message, deploymentID)
}

// watchDeploy finishes a deploy whether or not anyone has the app open. A resumed watcher treats a missing log as an action the restart interrupted, because deploys create the log before they start.
func (h *Handler) watchDeploy(serverID, serviceID, deploymentID string, resumed bool) {
	ctx := context.Background()
	deadline := time.Now().Add(deployWatchLimit)
	for time.Now().Before(deadline) {
		var status string
		if err := h.db.QueryRowContext(ctx, "SELECT status FROM database_services WHERE id = ?", serviceID).Scan(&status); err != nil || status != "deploying" {
			return
		}
		// A dropped connection says nothing about the deploy, so it is simply tried again.
		if output, err := h.executeSSH(ctx, serverID, deployProgressCommand(serviceID)); err == nil {
			progress := parseDeployProgress(string(output))
			if progress.done {
				h.finishDeploy(ctx, serviceID, progress.succeeded, progress.log, deploymentID)
				return
			}
			if resumed && !progress.logExists {
				h.finishDeploy(ctx, serviceID, false, "The action was interrupted when Kymber restarted. Run it again.", deploymentID)
				return
			}
		}
		time.Sleep(deployWatchInterval)
	}
	h.finishDeploy(ctx, serviceID, false, "The deployment did not finish within 2 hours.", deploymentID)
	logger.Warn("deploy watch timed out: service=%s", serviceID)
}

func (h *Handler) resumeDeployWatches() {
	rows, err := h.db.Query(`SELECT s.id, s.server_id, COALESCE((SELECT d.id FROM service_deployments d WHERE d.service_id = s.id AND d.status = 'deploying' ORDER BY d.started_at DESC LIMIT 1), '') FROM database_services s WHERE s.status = 'deploying' AND s.server_id IS NOT NULL`)
	if err != nil {
		logger.Warn("could not resume deploy watches: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var serviceID, serverID, deploymentID string
		if rows.Scan(&serviceID, &serverID, &deploymentID) == nil {
			go h.watchDeploy(serverID, serviceID, deploymentID, true)
		}
	}
	if err := rows.Err(); err != nil {
		logger.Warn("could not read deploy watches: %v", err)
	}
}
