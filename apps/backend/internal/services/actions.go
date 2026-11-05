package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
	"kymber/backend/internal/secrets"
	"kymber/backend/internal/templates"
)

func (h *Handler) serviceAction(w http.ResponseWriter, r *http.Request) {
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
	action := r.PathValue("action")
	if action != "deploy" && action != "start" && action != "stop" && action != "restart" && action != "remove_container" && action != "remove_container_image" {
		writeError(w, http.StatusBadRequest, "unsupported service action")
		return
	}
	if action != "deploy" {
		var status string
		if err := h.db.QueryRowContext(r.Context(), "SELECT status FROM database_services WHERE id = ? AND workspace_id = ? AND project_id = ?", serviceID, workspaceID, projectID).Scan(&status); err != nil {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		if status == "deploying" || status == "queued" {
			writeError(w, http.StatusConflict, "wait for the current action to finish")
			return
		}
	}
	imageVersion, requestedServer := "", ""
	if action == "deploy" && r.ContentLength != 0 {
		var input struct {
			ImageVersion string `json:"image_version"`
			ServerID     string `json:"server_id"`
		}
		if !decodeJSON(w, r, &input) {
			return
		}
		imageVersion, requestedServer = input.ImageVersion, input.ServerID
	}
	if action == "deploy" {
		service, err := h.readService(r, workspaceID, projectID, serviceID, false)
		if err != nil {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		if service.Compose {
			imageVersion = service.ImageVersion
		} else {
			template := serviceTemplate(service)
			if imageVersion == "" {
				imageVersion = service.ImageVersion
			}
			if _, err := template.ImageRef(imageVersion); err != nil {
				writeError(w, http.StatusBadRequest, "invalid image version")
				return
			}
		}
	}
	var nullableServerID sql.NullString
	var isCompose bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT server_id, compose_file IS NOT NULL FROM database_services WHERE id = ? AND workspace_id = ? AND project_id = ?", serviceID, workspaceID, projectID).Scan(&nullableServerID, &isCompose); err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if requestedServer != "" && nullableServerID.Valid && requestedServer != nullableServerID.String {
		writeError(w, http.StatusConflict, "app already runs on another server")
		return
	}
	if action == "deploy" && !nullableServerID.Valid {
		if isCompose {
			var localServer sql.NullString
			if err := h.db.QueryRowContext(r.Context(), "SELECT s.id FROM servers s WHERE s.connection_type = 'local' AND "+serverStatusSQL+" = 'online' ORDER BY s.created_at LIMIT 1").Scan(&localServer); err != nil || !localServer.Valid {
				writeError(w, http.StatusConflict, "Compose stacks require an online local server")
				return
			}
			nullableServerID = localServer
		}
		if requestedServer != "" && !isCompose {
			if !h.requireServerOnline(w, r, requestedServer) {
				return
			}
			nullableServerID = sql.NullString{String: requestedServer, Valid: true}
		} else if !nullableServerID.Valid {
			err := h.db.QueryRowContext(r.Context(), "SELECT s.id FROM servers s WHERE s.connection_type = 'local' AND "+serverStatusSQL+" = 'online' ORDER BY s.created_at LIMIT 1").Scan(&nullableServerID)
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusConflict, "no online server is available")
				return
			}
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not find an online server")
				return
			}
		}
		var domain string
		if err := h.db.QueryRowContext(r.Context(), "SELECT COALESCE(domain, '') FROM database_services WHERE id = ?", serviceID).Scan(&domain); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read service domain")
			return
		}
		if domain != "" {
			var connectionType string
			if err := h.db.QueryRowContext(r.Context(), "SELECT connection_type FROM servers WHERE id = ?", nullableServerID.String).Scan(&connectionType); err != nil || connectionType != "local" {
				writeError(w, http.StatusConflict, "public domains currently require the local server with Traefik")
				return
			}
		}
		if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET server_id = ? WHERE id = ?", nullableServerID.String, serviceID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not assign server")
			return
		}
	}
	if !nullableServerID.Valid {
		writeError(w, http.StatusConflict, "service has not been deployed")
		return
	}
	var connectionType string
	if err := h.db.QueryRowContext(r.Context(), "SELECT connection_type FROM servers WHERE id = ?", nullableServerID.String).Scan(&connectionType); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read server connection")
		return
	}
	if action == "deploy" {
		var domain string
		if err := h.db.QueryRowContext(r.Context(), "SELECT COALESCE(domain, '') FROM database_services WHERE id = ?", serviceID).Scan(&domain); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read service domain")
			return
		}
		if isCompose {
			if err := h.db.QueryRowContext(r.Context(), "SELECT connection_type FROM servers WHERE id = ?", nullableServerID.String).Scan(&connectionType); err != nil || connectionType != "local" {
				writeError(w, http.StatusConflict, "Compose stacks require the local server")
				return
			}
			service, err := h.readServiceByID(r, serviceID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not read stack")
				return
			}
			if err := h.validateComposeOnServer(r, nullableServerID.String, service, service.ComposeFile); err != nil {
				writeError(w, http.StatusBadRequest, "invalid Compose file: "+err.Error())
				return
			}
		}
		if domain != "" && connectionType != "local" {
			writeError(w, http.StatusConflict, "public domains currently require the local server with Traefik")
			return
		}
	}
	if connectionType == "local" && !h.requireServerOnline(w, r, nullableServerID.String) {
		return
	}
	serverID := nullableServerID.String
	if action == "remove_container" || action == "remove_container_image" {
		var status string
		if err := h.db.QueryRowContext(r.Context(), "SELECT status FROM database_services WHERE id = ?", serviceID).Scan(&status); err != nil || status != "stopped" {
			writeError(w, http.StatusConflict, "stop the application before removing its container")
			return
		}
		if isCompose {
			writeError(w, http.StatusBadRequest, "container removal is not available for stacks")
			return
		}
	}
	var sshConfigured bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT encrypted_ssh_key IS NOT NULL FROM servers WHERE id = ?", serverID).Scan(&sshConfigured); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read server credentials")
		return
	}
	if action == "deploy" {
		ports, err := h.readPorts(r.Context(), serviceID)
		if isCompose {
			err = nil
		} else if err == nil {
			err = h.checkPortConflicts(r.Context(), serverID, serviceID, ports)
		}
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}
	if connectionType == "ssh" || sshConfigured {
		if action != "deploy" {
			logFile, statusFile := deploymentLogFiles(serviceID)
			if _, err := h.executeSSH(r.Context(), serverID, "rm -f "+shellQuote(logFile)+" "+shellQuote(statusFile)+" "+shellQuote(statusFile+".tmp")); err != nil {
				writeError(w, http.StatusBadGateway, "could not clear previous deployment logs")
				return
			}
		}
		deploymentID := ""
		if action == "deploy" {
			var err error
			deploymentID, err = database.NewShortID()
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not create deployment")
				return
			}
			tx, err := h.db.BeginTx(r.Context(), nil)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not update service status")
				return
			}
			defer tx.Rollback()
			result, err := tx.ExecContext(r.Context(), "UPDATE database_services SET status = 'deploying', image_version = ?, last_error = NULL WHERE id = ? AND status NOT IN ('deploying', 'queued')", imageVersion, serviceID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not update service status")
				return
			}
			changed, err := result.RowsAffected()
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not check service status")
				return
			}
			if changed == 0 {
				writeError(w, http.StatusConflict, "wait for the current action to finish")
				return
			}
			if _, err := tx.ExecContext(r.Context(), "INSERT INTO service_deployments (id, service_id, action, image_version, status, output) VALUES (?, ?, ?, ?, 'deploying', '')", deploymentID, serviceID, action, imageVersion); err != nil {
				writeError(w, http.StatusInternalServerError, "could not record deployment")
				return
			}
			if err := tx.Commit(); err != nil {
				writeError(w, http.StatusInternalServerError, "could not start deployment")
				return
			}
		} else {
			result, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'deploying', last_error = NULL WHERE id = ? AND status NOT IN ('deploying', 'queued')", serviceID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not update service status")
				return
			}
			if changed, _ := result.RowsAffected(); changed == 0 {
				writeError(w, http.StatusConflict, "wait for the current action to finish")
				return
			}
		}
		if action == "deploy" {
			service, err := h.readServiceByID(r, serviceID)
			if err == nil && service.Compose {
				if err := h.startComposeDeploySSH(r, serverID, serviceID, action); err != nil {
					h.failDeployment(r.Context(), serviceID, deploymentID, err.Error())
					writeError(w, http.StatusBadGateway, "Compose deployment failed: "+err.Error())
					return
				}
				go h.watchDeploy(serverID, serviceID, deploymentID, false)
				w.WriteHeader(http.StatusAccepted)
				return
			}
			if err := h.startServiceDeploySSH(r, serverID, serviceID, imageVersion); err != nil {
				h.failDeployment(r.Context(), serviceID, deploymentID, err.Error())
				writeError(w, http.StatusBadGateway, "SSH service deployment failed: "+err.Error())
				return
			}
			go h.watchDeploy(serverID, serviceID, deploymentID, false)
			w.WriteHeader(http.StatusAccepted)
			return
		}
		service, err := h.readServiceByID(r, serviceID)
		if err == nil && service.Compose {
			if action == "remove_container" || action == "remove_container_image" {
				writeError(w, http.StatusBadRequest, "container removal is not available for stacks")
				return
			}
			if action == "start" || action == "restart" {
				if err := h.startComposeDeploySSH(r, serverID, serviceID, action); err != nil {
					_, _ = h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ?", err.Error(), serviceID)
					writeError(w, http.StatusBadGateway, "Compose action failed: "+err.Error())
					return
				}
				go h.watchDeploy(serverID, serviceID, "", false)
				w.WriteHeader(http.StatusAccepted)
				return
			}
			if err := h.executeComposeSSH(r, serverID, service, action); err != nil {
				_, _ = h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ? AND status = 'deploying'", err.Error(), serviceID)
				writeError(w, http.StatusBadGateway, "Compose action failed: "+err.Error())
				return
			}
			status := "running"
			if action == "stop" {
				status = "stopped"
			}
			if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET status = ? WHERE id = ?", status, serviceID); err != nil {
				writeError(w, http.StatusInternalServerError, "could not update stack status")
				return
			}
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if err := h.executeServiceSSH(r, serverID, serviceID, action, imageVersion); err != nil {
			_, _ = h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ?", err.Error(), serviceID)
			writeError(w, http.StatusBadGateway, "SSH service action failed: "+err.Error())
			return
		}
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if err := h.queueServiceAction(r, serverID, serviceID, action, imageVersion); err != nil {
		if errors.Is(err, errServiceActionInProgress) {
			writeError(w, http.StatusConflict, "wait for the current action to finish")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not queue service action")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) serviceLogs(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, "") {
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
	if service.ServerID == "" {
		writeError(w, http.StatusConflict, "service has not been deployed")
		return
	}
	var configured bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT encrypted_ssh_key IS NOT NULL FROM servers WHERE id = ?", service.ServerID).Scan(&configured); err != nil || !configured {
		writeError(w, http.StatusConflict, "server does not support SSH logs")
		return
	}
	if r.URL.Query().Get("source") == "runtime" {
		tail, ok := parseLogTail(r.URL.Query().Get("tail"))
		if !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("tail must be between 1 and %d", maxLogTail))
			return
		}
		command := runtimeLogsCommand(service.Name+"-service-"+service.ID, tail)
		if service.Compose {
			composeLogs := "docker compose -p " + shellQuote(service.Name) + " -f " + shellQuote("/tmp/kymber-compose-"+service.ID+".yaml") + " logs --tail " + strconv.Itoa(tail)
			if container := r.URL.Query().Get("container"); container != "" {
				if !composeContainerExists(service.ComposeFile, container) {
					writeError(w, http.StatusBadRequest, "unknown Compose service")
					return
				}
				composeLogs += " " + shellQuote(container)
			}
			command = composeLogs + " 2>&1 | tail -c " + strconv.Itoa(maxLogBytes)
		}
		logs, err := h.executeSSH(r.Context(), service.ServerID, command)
		if err != nil {
			writeError(w, http.StatusBadGateway, "could not read container logs: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": service.Status, "logs": string(logs)})
		return
	}
	logFile, statusFile := deploymentLogFiles(service.ID)
	name := service.Name + "-service-" + service.ID
	command := "if test -f " + shellQuote(logFile) + "; then printf '__KYMBER_LOG__'; tail -c 65536 " + shellQuote(logFile) + "; else printf '__KYMBER_NO_LOG__'; fi; if test -f " + shellQuote(statusFile) + "; then code=$(cat " + shellQuote(statusFile) + "); printf '\\n__KYMBER_EXIT__%s' \"$code\"; if test \"$code\" = 0; then printf '\\n__KYMBER_CONTAINER__'; if docker inspect " + shellQuote(name) + " >/dev/null 2>&1; then docker logs --tail 300 " + shellQuote(name) + " 2>&1; else printf 'Container output is not available yet.'; fi; fi; fi"
	if service.Compose {
		command = "if test -f " + shellQuote(logFile) + "; then printf '__KYMBER_LOG__'; tail -c 65536 " + shellQuote(logFile) + "; else printf '__KYMBER_NO_LOG__'; fi; if test -f " + shellQuote(statusFile) + "; then printf '\\n__KYMBER_EXIT__'; cat " + shellQuote(statusFile) + "; fi"
	}
	logs, err := h.executeSSH(r.Context(), service.ServerID, command)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not read deployment logs: "+err.Error())
		return
	}
	status := service.Status
	if (service.Status == "deploying" || service.Status == "running") && strings.Contains(string(logs), "__KYMBER_LOG__") {
		_, output, _ := strings.Cut(string(logs), "__KYMBER_LOG__")
		if before, after, found := strings.Cut(output, "__KYMBER_EXIT__"); found {
			output = before
			code, after, _ := strings.Cut(after, "\n")
			code = strings.TrimSpace(code)
			if code == "0" {
				status = "running"
				h.finishDeploy(r.Context(), serviceID, true, "")
				if !service.Compose {
					if _, container, found := strings.Cut(after, "__KYMBER_CONTAINER__"); found {
						output = strings.TrimSpace(output) + "\n\n--- Container output ---\n" + strings.TrimSpace(container)
					}
				}
			} else {
				status = "failed"
				h.finishDeploy(r.Context(), serviceID, false, strings.TrimSpace(output))
			}
		}
		logs = []byte(strings.TrimSpace(output))
	} else if service.Status == "deploying" || service.Status == "draft" || service.Status == "queued" {
		logs = []byte("Preparing Docker deployment…")
	} else {
		command := "if docker inspect " + shellQuote(name) + " >/dev/null 2>&1; then docker logs --tail 300 " + shellQuote(name) + " 2>&1; else printf 'Container is not running.'; fi"
		if service.Compose {
			composeLogs := "docker compose -p " + shellQuote(service.Name) + " -f " + shellQuote("/tmp/kymber-compose-"+service.ID+".yaml") + " logs --tail 300"
			if container := r.URL.Query().Get("container"); container != "" {
				if !composeContainerExists(service.ComposeFile, container) {
					writeError(w, http.StatusBadRequest, "unknown Compose service")
					return
				}
				composeLogs += " " + shellQuote(container)
			}
			command = composeLogs + " 2>&1 | tail -c " + strconv.Itoa(maxLogBytes)
		}
		logs, err = h.executeSSH(r.Context(), service.ServerID, command)
		if err != nil {
			writeError(w, http.StatusBadGateway, "could not read container logs: "+err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status, "logs": string(logs)})
}

const (
	maxLogTail  = 50000
	maxLogBytes = 10 << 20
)

func parseLogTail(raw string) (int, bool) {
	if raw == "" {
		return 300, true
	}
	tail, err := strconv.Atoi(raw)
	return tail, err == nil && tail >= 1 && tail <= maxLogTail
}

// The byte cap keeps a few enormous lines from loading the whole log into the API's memory.
func runtimeLogsCommand(container string, tail int) string {
	return fmt.Sprintf("if docker inspect %[1]s >/dev/null 2>&1; then docker logs --tail %[2]d %[1]s 2>&1 | tail -c %[3]d; else printf 'Container is not running.\\n'; fi", shellQuote(container), tail, maxLogBytes)
}

func (h *Handler) executeServiceSSH(r *http.Request, serverID, serviceID, action, imageVersion string) error {
	service, err := h.readServiceByID(r, serviceID)
	if err != nil {
		return err
	}
	if action != "deploy" {
		imageVersion = service.ImageVersion
	}
	template := serviceTemplate(service)
	imageRef, err := template.ImageRef(imageVersion)
	if err != nil {
		return err
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'deploying', last_error = NULL WHERE id = ?", serviceID); err != nil {
		return err
	}
	storages, err := h.readStorages(r.Context(), serviceID, false)
	if err != nil {
		return err
	}
	ports, err := h.readPorts(r.Context(), serviceID)
	if err != nil {
		return err
	}
	command, err := h.sshDockerCommand(service, template, imageRef, action, storages, ports)
	if err == nil {
		_, err = h.executeSSH(r.Context(), serverID, command)
	}
	if err != nil {
		_, _ = h.db.ExecContext(r.Context(), "UPDATE database_services SET status = 'failed', last_error = ? WHERE id = ?", err.Error(), serviceID)
		return err
	}
	status := "running"
	switch action {
	case "stop":
		status = "stopped"
	case "delete":
		status = "deleted"
	case "remove_container", "remove_container_image":
		status = "stopped"
	}
	_, err = h.db.ExecContext(r.Context(), "UPDATE database_services SET status = ?, last_error = NULL WHERE id = ?", status, serviceID)
	return err
}

func (h *Handler) startServiceDeploySSH(r *http.Request, serverID, serviceID, imageVersion string) error {
	service, err := h.readServiceByID(r, serviceID)
	if err != nil {
		return err
	}
	template := serviceTemplate(service)
	image, err := template.ImageRef(imageVersion)
	if err != nil {
		return err
	}
	storages, err := h.readStorages(r.Context(), serviceID, true)
	if err != nil {
		return err
	}
	ports, err := h.readPorts(r.Context(), serviceID)
	if err != nil {
		return err
	}
	command, err := h.sshDockerCommand(service, template, image, "deploy", storages, ports)
	if err != nil {
		return err
	}
	logFile, statusFile := deploymentLogFiles(serviceID)
	pidFile := deploymentProcessFile(serviceID)
	script := "echo $$ > " + shellQuote(pidFile) + "; (" + command + ") > " + shellQuote(logFile) + " 2>&1; code=$?; rm -f " + shellQuote(pidFile) + "; printf '%s' \"$code\" > " + shellQuote(statusFile+".tmp") + "; mv " + shellQuote(statusFile+".tmp") + " " + shellQuote(statusFile)
	launch := ": > " + shellQuote(logFile) + "; rm -f " + shellQuote(statusFile) + " " + shellQuote(pidFile) + "; nohup setsid sh -c " + shellQuote(script) + " </dev/null >/dev/null 2>&1 &"
	_, err = h.executeSSH(r.Context(), serverID, launch)
	return err
}

func deploymentLogFiles(serviceID string) (string, string) {
	base := "/tmp/kymber-service-" + serviceID
	return base + ".log", base + ".exit"
}

func deploymentProcessFile(serviceID string) string {
	return "/tmp/kymber-service-" + serviceID + ".pid"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (h *Handler) readServiceByID(r *http.Request, id string) (databaseService, error) {
	var service databaseService
	var encrypted []byte
	err := h.db.QueryRowContext(r.Context(), "SELECT id, workspace_id, project_id, engine, name, image_version, image, port, COALESCE(domain, ''), data_path, COALESCE(server_id, ''), database_name, database_user, encrypted_config, compose_file IS NOT NULL, COALESCE(CAST(compose_file AS CHAR), '') FROM database_services WHERE id = ?", id).Scan(&service.ID, &service.WorkspaceID, &service.ProjectID, &service.Engine, &service.Name, &service.ImageVersion, &service.Image, &service.Port, &service.Domain, &service.DataPath, &service.ServerID, &service.DatabaseName, &service.DatabaseUser, &encrypted, &service.Compose, &service.ComposeFile)
	if err != nil {
		return service, err
	}
	if service.Compose {
		return service, nil
	}
	plain, err := secrets.Decrypt(encrypted)
	if err != nil {
		return service, err
	}
	if err := json.Unmarshal(plain, &service.Environment); err != nil {
		return service, err
	}
	return service, nil
}

func (h *Handler) sshDockerCommand(s databaseService, t templates.Template, image, action string, storages []serviceStorage, ports []servicePort) (string, error) {
	name := t.ContainerName(s.Name, s.ID)
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
	if action == "stop" {
		return "docker stop " + quote(name), nil
	}
	if action == "start" {
		return "docker start " + quote(name), nil
	}
	if action == "restart" {
		return "docker restart " + quote(name), nil
	}
	if action == "remove_container" {
		return "docker rm -f " + quote(name), nil
	}
	if action == "remove_container_image" {
		return "docker rm -f " + quote(name) + " >/dev/null 2>&1 || true; docker image rm " + quote(image), nil
	}
	if action == "delete" {
		command := "docker rm -f " + quote(name) + " >/dev/null 2>&1 || true; " + storageDeleteCommand(s.ID) + ":"
		if t.DataPath != "" {
			command += "; docker volume rm " + quote(t.VolumeName(s.ID))
		}
		return command, nil
	}
	if action != "deploy" {
		return "", fmt.Errorf("unsupported action")
	}
	if !templates.ValidDataPath(t.DataPath) {
		return "", fmt.Errorf("invalid data path")
	}
	config, ok := templates.DockerArgs(s.Engine, s.Environment)
	if !ok {
		return "", fmt.Errorf("unsupported engine")
	}
	var env strings.Builder
	for key, value := range s.Environment {
		env.WriteString("printf '%s\\n' ")
		env.WriteString(quote(key + "=" + value))
		env.WriteString(" >> \"$f\"; ")
	}
	prepare, storageArgs := storageDeployCommand(s.ID, storages)
	portCheck, portArgs := portDeployCommand(ports)
	command := "set -eu; n=" + quote(name) + "; f=$(mktemp); trap 'rm -f \"$f\"' EXIT; " + portCheck + env.String() + prepare + "if docker image inspect " + quote(image) + " >/dev/null 2>&1; then echo 'Image found locally.'; else echo 'Image not found locally; pulling…'; docker pull " + quote(image) + "; fi; docker network inspect " + quote("kymber-workspace-"+s.WorkspaceID) + " >/dev/null 2>&1 || docker network create --label kymber.managed=true " + quote("kymber-workspace-"+s.WorkspaceID) + " >/dev/null; docker rm -f \"$n\" >/dev/null 2>&1 || true; docker run --pull=never -d --name \"$n\" --restart unless-stopped --network " + quote("kymber-workspace-"+s.WorkspaceID) + " --network-alias " + quote(t.InternalHost(s.ID)) + " --label kymber.managed=true --label " + quote("kymber.service="+s.ID)
	if s.Domain != "" {
		if t.Port < 1 || t.Port > 65535 {
			return "", fmt.Errorf("public domain requires an HTTP port")
		}
		if len(s.Domain) > 253 || !serviceDomainPattern.MatchString(s.Domain) || net.ParseIP(s.Domain) != nil {
			return "", fmt.Errorf("invalid public domain")
		}
		name := "kymber-" + s.ID
		host := "Host(`" + s.Domain + "`)"
		for _, label := range []string{
			"traefik.enable=true",
			"traefik.docker.network=kymber",
			"traefik.http.routers." + name + "-http.rule=" + host,
			"traefik.http.routers." + name + "-http.entrypoints=web",
			"traefik.http.routers." + name + "-http.middlewares=" + name + "-https",
			"traefik.http.routers." + name + "-http.service=" + name,
			"traefik.http.middlewares." + name + "-https.redirectscheme.scheme=https",
			"traefik.http.routers." + name + ".rule=" + host,
			"traefik.http.routers." + name + ".entrypoints=websecure",
			"traefik.http.routers." + name + ".tls.certresolver=letsencrypt",
			"traefik.http.routers." + name + ".service=" + name,
			fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=%d", name, t.Port),
		} {
			command += " --label " + quote(label)
		}
		command = strings.Replace(command, "; docker rm -f", "; docker network inspect kymber >/dev/null 2>&1 || { echo 'Traefik network kymber is missing; start the Kymber proxy first' >&2; exit 1; }; docker rm -f", 1)
	}
	if t.DataPath != "" {
		command += " --volume " + quote(t.VolumeName(s.ID)+":"+t.DataPath)
	}
	if t.HealthCmd != "" {
		command += " --health-cmd " + quote(t.HealthCmd) + " --health-interval 10s --health-timeout 5s --health-retries 10 --health-start-period 30s"
	}
	command += storageArgs + portArgs + " --env-file \"$f\" " + quote(image)
	for _, arg := range config {
		command += " " + quote(arg)
	}
	if s.Domain != "" {
		command += "; docker network connect kymber \"$n\""
	}
	if t.HealthCmd != "" {
		command += "; i=0; while test \"$i\" -lt 90; do state=$(docker inspect --format '{{.State.Health.Status}}' \"$n\"); if test \"$state\" = healthy; then break; fi; if test \"$state\" = unhealthy; then docker rm -f \"$n\" >/dev/null 2>&1 || true; echo 'Container health check failed.' >&2; exit 1; fi; i=$((i+1)); sleep 2; done; test \"$(docker inspect --format '{{.State.Health.Status}}' \"$n\")\" = healthy || { docker rm -f \"$n\" >/dev/null 2>&1 || true; echo 'Container did not become healthy within 3 minutes.' >&2; exit 1; }"
	}
	return command, nil
}
