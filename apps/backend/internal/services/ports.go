package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"kymber/backend/internal/database"
)

const maxPortsPerService = 20

type servicePort struct {
	ID            string `json:"id"`
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
	Bind          string `json:"bind"`
	Applied       bool   `json:"applied"`
}

func validatePort(hostPort, containerPort int, protocol, bind string) error {
	if hostPort < 1 || hostPort > 65535 || containerPort < 1 || containerPort > 65535 {
		return errors.New("ports must be between 1 and 65535")
	}
	if protocol != "tcp" && protocol != "udp" {
		return errors.New("protocol must be tcp or udp")
	}
	if bind != "0.0.0.0" && bind != "127.0.0.1" {
		return errors.New("bind must be 0.0.0.0 or 127.0.0.1")
	}
	return nil
}

// A deploy removes the old container before starting the new one, so a taken port must stop it first or the app goes down. Ports the app itself holds do not count.
func portDeployCommand(ports []servicePort) (check, args string) {
	for _, p := range ports {
		flag := "-Hltn"
		if p.Protocol == "udp" {
			flag = "-Hlun"
		}
		check += fmt.Sprintf("if command -v ss >/dev/null 2>&1 && [ -n \"$(ss %s 'sport = :%d')\" ] && ! docker port \"$n\" 2>/dev/null | grep -q ':%d$'; then echo 'Port %d/%s is already in use on the server. The running app was not changed.'; exit 1; fi; ", flag, p.HostPort, p.HostPort, p.HostPort, p.Protocol)
		args += fmt.Sprintf(" --publish %s:%d:%d/%s", p.Bind, p.HostPort, p.ContainerPort, p.Protocol)
	}
	return check, args
}

func (h *Handler) readPorts(ctx context.Context, serviceID string) ([]servicePort, error) {
	rows, err := h.db.QueryContext(ctx, "SELECT id, host_port, container_port, protocol, bind_address, applied FROM service_ports WHERE service_id = ? ORDER BY host_port, protocol", serviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ports := make([]servicePort, 0)
	for rows.Next() {
		var p servicePort
		if err := rows.Scan(&p.ID, &p.HostPort, &p.ContainerPort, &p.Protocol, &p.Bind, &p.Applied); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}
	return ports, rows.Err()
}

// portOwner names another app on the server already publishing this port, or returns an empty string.
func (h *Handler) portOwner(ctx context.Context, serverID, serviceID string, hostPort int, protocol string) (string, error) {
	var owner string
	err := h.db.QueryRowContext(ctx, `SELECT d.name FROM service_ports p JOIN database_services d ON d.id = p.service_id
		WHERE d.server_id = ? AND d.id <> ? AND d.status <> 'deleted' AND p.host_port = ? AND p.protocol = ? LIMIT 1`, serverID, serviceID, hostPort, protocol).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return owner, err
}

// An app gets its server at first deploy, so ports added as a draft can only be compared with other apps then.
func (h *Handler) checkPortConflicts(ctx context.Context, serverID, serviceID string, ports []servicePort) error {
	for _, p := range ports {
		owner, err := h.portOwner(ctx, serverID, serviceID, p.HostPort, p.Protocol)
		if err != nil {
			return err
		}
		if owner != "" {
			return fmt.Errorf("port %d/%s is already published by %s on this server", p.HostPort, p.Protocol, owner)
		}
	}
	return nil
}

func (h *Handler) listPorts(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.storageRequest(w, r, false)
	if !ok {
		return
	}
	ports, err := h.readPorts(r.Context(), service.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read ports")
		return
	}
	var host string
	if service.ServerID != "" {
		// A local server's SSH host is an internal address, so the client uses the address it opened Kymber on.
		if err := h.db.QueryRowContext(r.Context(), "SELECT CASE WHEN connection_type = 'ssh' THEN COALESCE(host, '') ELSE '' END FROM servers WHERE id = ?", service.ServerID).Scan(&host); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read server")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ports": ports, "host": host})
}

func (h *Handler) createPort(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.storageRequest(w, r, true)
	if !ok {
		return
	}
	var input struct {
		HostPort      int    `json:"host_port"`
		ContainerPort int    `json:"container_port"`
		Protocol      string `json:"protocol"`
		Bind          string `json:"bind"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	if err := validatePort(input.HostPort, input.ContainerPort, input.Protocol, input.Bind); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var count int
	if err := h.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM service_ports WHERE service_id = ?", service.ID).Scan(&count); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read ports")
		return
	}
	if count >= maxPortsPerService {
		writeError(w, http.StatusConflict, fmt.Sprintf("an app can publish up to %d ports", maxPortsPerService))
		return
	}
	if service.ServerID != "" {
		owner, err := h.portOwner(r.Context(), service.ServerID, service.ID, input.HostPort, input.Protocol)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not check the port")
			return
		}
		if owner != "" {
			writeError(w, http.StatusConflict, fmt.Sprintf("%s already publishes this port on the server", owner))
			return
		}
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate port id")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO service_ports (id, service_id, host_port, container_port, protocol, bind_address) VALUES (?, ?, ?, ?, ?, ?)", id, service.ID, input.HostPort, input.ContainerPort, input.Protocol, input.Bind); err != nil {
		writeError(w, http.StatusConflict, "this app already publishes this port")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]servicePort{"port": {ID: id, HostPort: input.HostPort, ContainerPort: input.ContainerPort, Protocol: input.Protocol, Bind: input.Bind}})
}

func (h *Handler) deletePort(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.storageRequest(w, r, true)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "portID")
	if !ok {
		return
	}
	result, err := h.db.ExecContext(r.Context(), "DELETE FROM service_ports WHERE id = ? AND service_id = ?", id, service.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete port")
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		writeError(w, http.StatusNotFound, "port not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
