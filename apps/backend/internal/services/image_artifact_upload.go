package services

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"kymber/backend/internal/artifacts"
	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
)

func (h *Handler) deployImageArtifact(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permissions.AppsManage) {
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
	if service.Category != "application" || service.Engine != "docker" || service.Compose || service.ServerID == "" {
		writeError(w, http.StatusConflict, "OCI artifacts require a deployed Docker application")
		return
	}
	if service.Status == "deploying" || service.Status == "queued" {
		writeError(w, http.StatusConflict, "wait for the current action to finish")
		return
	}
	var connectionType string
	var sshConfigured bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT connection_type, encrypted_ssh_key IS NOT NULL FROM servers WHERE id = ?", service.ServerID).Scan(&connectionType, &sshConfigured); err != nil || connectionType != "ssh" && connectionType != "local" || !sshConfigured {
		writeError(w, http.StatusConflict, "OCI artifacts require an SSH-connected server")
		return
	}
	if !h.requireServerOnline(w, r, service.ServerID) {
		return
	}
	if r.ContentLength > artifacts.MaxOCIArchiveSize {
		writeError(w, http.StatusRequestEntityTooLarge, "OCI archive exceeds 1 GiB")
		return
	}
	if !allowArtifactTransfer(w) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, artifacts.MaxOCIArchiveSize)
	archive, _, err := artifacts.SpoolOCI(r.Body)
	if err != nil {
		status := http.StatusBadRequest
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) || strings.Contains(err.Error(), "exceeds 1 GiB") {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, err.Error())
		return
	}
	defer os.Remove(archive.Name())
	defer archive.Close()
	deploymentID, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create deployment")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not start deployment")
		return
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(r.Context(), `UPDATE database_services
		SET status = 'deploying', image = 'docker.io/kymber/build-artifacts', image_version = ?, last_error = NULL
		WHERE id = ? AND workspace_id = ? AND project_id = ? AND server_id = ? AND status NOT IN ('deploying', 'queued')`, deploymentID, serviceID, workspaceID, projectID, service.ServerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not reserve deployment")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not check deployment status")
		return
	}
	if changed == 0 {
		writeError(w, http.StatusConflict, "wait for the current action to finish")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO service_deployments (id, service_id, action, image_version, status, output) VALUES (?, ?, 'deploy', ?, 'deploying', '')", deploymentID, serviceID, deploymentID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not record deployment")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not commit deployment")
		return
	}
	if _, err := h.importOCIImage(r.Context(), service.ServerID, deploymentID, archive); err != nil {
		h.failDeployment(context.WithoutCancel(r.Context()), serviceID, deploymentID, err.Error())
		writeError(w, http.StatusBadGateway, "could not import OCI image: "+err.Error())
		return
	}
	if err := h.startServiceDeploySSH(r, service.ServerID, serviceID, deploymentID); err != nil {
		h.failDeployment(context.WithoutCancel(r.Context()), serviceID, deploymentID, err.Error())
		writeError(w, http.StatusBadGateway, "could not start imported image: "+err.Error())
		return
	}
	go h.watchDeploy(service.ServerID, serviceID, deploymentID, false)
	writeJSON(w, http.StatusAccepted, map[string]string{"deployment_id": deploymentID, "status": "deploying"})
}
