package services

import (
	"net/http"

	"kymber/backend/internal/auth"
)

func (h *Handler) cancelServiceDeploy(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, true) {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return
	}
	serviceID, ok := pathID(w, r, "serviceID")
	if !ok {
		return
	}
	service, err := h.readService(r, workspaceID, projectID, serviceID, false)
	if err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if service.Status != "deploying" {
		writeError(w, http.StatusConflict, "service is not deploying")
		return
	}
	var configured bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT encrypted_ssh_key IS NOT NULL FROM servers WHERE id = ?", service.ServerID).Scan(&configured); err != nil || !configured {
		writeError(w, http.StatusConflict, "force stop is only available for SSH deployments")
		return
	}
	logFile, statusFile := deploymentLogFiles(serviceID)
	pidFile := deploymentProcessFile(serviceID)
	command := "pid=; if test -s " + shellQuote(pidFile) + "; then pid=$(cat " + shellQuote(pidFile) + "); else self_group=$(ps -o pgid= -p $$ | tr -d ' '); for file in /proc/[0-9]*/cmdline; do cmd=$(tr '\\000' ' ' < \"$file\" 2>/dev/null) || continue; case \"$cmd\" in *" + logFile + "*) candidate=${file#/proc/}; candidate=${candidate%/cmdline}; group=$(ps -o pgid= -p \"$candidate\" 2>/dev/null | tr -d ' '); if test \"$group\" -gt 1 2>/dev/null && test \"$group\" != \"$self_group\"; then pid=$group; break; fi;; esac; done; fi; case \"$pid\" in ''|*[!0-9]*) exit 1;; esac; test \"$pid\" -gt 1; kill -KILL \"-$pid\" 2>/dev/null || true; rm -f " + shellQuote(pidFile) + "; printf '\\nDeployment force-stopped by user.\\n' >> " + shellQuote(logFile) + "; printf '130' > " + shellQuote(statusFile+".tmp") + "; mv " + shellQuote(statusFile+".tmp") + " " + shellQuote(statusFile)
	if _, err := h.executeSSH(r.Context(), service.ServerID, command); err != nil {
		writeError(w, http.StatusBadGateway, "could not force-stop deployment: "+err.Error())
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ? AND status = 'deploying'", "Deployment force-stopped by user.", serviceID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update deployment status")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
