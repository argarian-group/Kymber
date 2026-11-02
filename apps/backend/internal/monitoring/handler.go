package monitoring

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/sshclient"
)

const (
	nodeExporterImage = "prom/node-exporter:v1.9.1"
	cadvisorImage     = "gcr.io/cadvisor/cadvisor:v0.55.1"
	removeCommand     = "docker rm -f kymber-node-exporter kymber-cadvisor >/dev/null 2>&1 || true"
)

var exporterPorts = map[string]string{"node": "9100", "cadvisor": "9101"}

const installCommand = "set -e; " + removeCommand + "; " +
	"docker run -d --name kymber-node-exporter --restart unless-stopped --label kymber.managed=true --net=host --pid=host -v /:/host:ro,rslave " + nodeExporterImage + " --path.rootfs=/host --web.listen-address=127.0.0.1:9100 >/dev/null; " +
	"docker run -d --name kymber-cadvisor --restart unless-stopped --label kymber.managed=true --privileged --device /dev/kmsg -p 127.0.0.1:9101:8080 -v /:/rootfs:ro -v /var/run:/var/run:ro -v /sys:/sys:ro -v /var/lib/docker/:/var/lib/docker:ro -v /dev/disk/:/dev/disk:ro " + cadvisorImage + " --docker_only=true --store_container_labels=false --whitelisted_container_labels=kymber.service --enable_metrics=cpu,memory,network --housekeeping_interval=30s >/dev/null"

type Handler struct {
	db    *sql.DB
	token string
}

func New(db *sql.DB) *Handler {
	token := os.Getenv("KYMBER_METRICS_TOKEN")
	if path := os.Getenv("KYMBER_METRICS_TOKEN_FILE"); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			token = string(data)
		}
	}
	return &Handler{db: db, token: strings.TrimSpace(token)}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	for _, route := range []struct {
		pattern string
		handler http.HandlerFunc
	}{
		{"POST /api/servers/{serverID}/monitoring", h.install},
		{"DELETE /api/servers/{serverID}/monitoring", h.uninstall},
		{"GET /api/servers/{serverID}/metrics", h.serverMetrics},
		{"GET /api/monitoring/alerts", h.alerts},
	} {
		mux.Handle(route.pattern, auth.Authenticate(h.db, route.handler))
	}
	mux.HandleFunc("GET /internal/metrics/targets", h.targets)
	mux.HandleFunc("GET /internal/metrics/servers/{serverID}/{exporter}", h.scrape)
}

func Remove(ctx context.Context, db *sql.DB, serverID string) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var installed bool
	if db.QueryRowContext(ctx, "SELECT monitoring_installed_at IS NOT NULL FROM servers WHERE id = ?", serverID).Scan(&installed) != nil || !installed {
		return
	}
	_, _ = sshclient.Run(ctx, db, serverID, removeCommand)
}

func (h *Handler) rootServer(w http.ResponseWriter, r *http.Request) (string, bool) {
	user, _ := auth.UserFromContext(r)
	if !user.IsRoot {
		writeError(w, http.StatusForbidden, "root access required")
		return "", false
	}
	id := r.PathValue("serverID")
	if !idPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid serverID")
		return "", false
	}
	var sshConfigured bool
	err := h.db.QueryRowContext(r.Context(), "SELECT encrypted_ssh_key IS NOT NULL FROM servers WHERE id = ?", id).Scan(&sshConfigured)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "server not found")
		return "", false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read server")
		return "", false
	}
	if !sshConfigured {
		writeError(w, http.StatusConflict, "set up SSH for this server first")
		return "", false
	}
	return id, true
}

func (h *Handler) install(w http.ResponseWriter, r *http.Request) {
	id, ok := h.rootServer(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	if output, err := sshclient.Run(ctx, h.db, id, installCommand); err != nil {
		writeError(w, http.StatusBadGateway, strings.TrimSpace("could not start monitoring on the server: "+string(output)))
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE servers SET monitoring_installed_at = UTC_TIMESTAMP(6) WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save monitoring status")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) uninstall(w http.ResponseWriter, r *http.Request) {
	id, ok := h.rootServer(w, r)
	if !ok {
		return
	}
	if output, err := sshclient.Run(r.Context(), h.db, id, removeCommand); err != nil {
		writeError(w, http.StatusBadGateway, strings.TrimSpace("could not stop monitoring on the server: "+string(output)))
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE servers SET monitoring_installed_at = NULL WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save monitoring status")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) serverMetrics(w http.ResponseWriter, r *http.Request) {
	id, ok := h.rootServer(w, r)
	if !ok {
		return
	}
	span, ok := ParseRange(r.URL.Query().Get("range"))
	if !ok {
		writeError(w, http.StatusBadRequest, "range must be 1h, 6h, 24h or 7d")
		return
	}
	queries, _ := ServerQueries(id)
	series, err := QueryAll(r.Context(), queries, span)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "metrics are unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": series})
}

type alertView struct {
	Name        string    `json:"name"`
	ServerID    string    `json:"server_id,omitempty"`
	ServerName  string    `json:"server_name,omitempty"`
	ServiceID   string    `json:"service_id,omitempty"`
	ServiceName string    `json:"service_name,omitempty"`
	ProjectID   string    `json:"project_id,omitempty"`
	Since       time.Time `json:"since"`
}

func (h *Handler) alerts(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	firing, err := firingAlerts(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "metrics are unavailable")
		return
	}
	visible := make([]alertView, 0, len(firing))
	for _, alert := range firing {
		view := alertView{Name: alert.Name, Since: alert.Since}
		switch {
		case alert.ServiceID != "":
			err = h.db.QueryRowContext(r.Context(), `SELECT d.id, d.name, d.project_id FROM database_services d WHERE d.id = ? AND d.status <> 'deleted'
				AND (? OR EXISTS(SELECT 1 FROM workspace_members m WHERE m.workspace_id = d.workspace_id AND m.user_id = ?))`, alert.ServiceID, user.IsRoot, user.ID).Scan(&view.ServiceID, &view.ServiceName, &view.ProjectID)
		case alert.ServerID != "" && user.IsRoot:
			err = h.db.QueryRowContext(r.Context(), "SELECT id, name FROM servers WHERE id = ?", alert.ServerID).Scan(&view.ServerID, &view.ServerName)
		default:
			continue
		}
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not read alerts")
			return
		}
		visible = append(visible, view)
	}
	writeJSON(w, http.StatusOK, map[string][]alertView{"alerts": visible})
}

func (h *Handler) authorizedScraper(w http.ResponseWriter, r *http.Request) bool {
	if h.token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+h.token)) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid metrics token")
		return false
	}
	return true
}

func (h *Handler) targets(w http.ResponseWriter, r *http.Request) {
	if !h.authorizedScraper(w, r) {
		return
	}
	exporter := r.URL.Query().Get("exporter")
	if _, ok := exporterPorts[exporter]; !ok {
		writeError(w, http.StatusBadRequest, "exporter must be node or cadvisor")
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT id, name FROM servers WHERE encrypted_ssh_key IS NOT NULL AND monitoring_installed_at IS NOT NULL ORDER BY id")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list servers")
		return
	}
	defer rows.Close()
	groups := make([]map[string]any, 0)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read servers")
			return
		}
		groups = append(groups, map[string]any{"targets": []string{r.Host}, "labels": map[string]string{"server_id": id, "server_name": name, "__metrics_path__": "/internal/metrics/servers/" + id + "/" + exporter}})
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read servers")
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *Handler) scrape(w http.ResponseWriter, r *http.Request) {
	if !h.authorizedScraper(w, r) {
		return
	}
	id := r.PathValue("serverID")
	port, ok := exporterPorts[r.PathValue("exporter")]
	if !idPattern.MatchString(id) || !ok {
		writeError(w, http.StatusNotFound, "exporter not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	connection, err := sshclient.Connect(ctx, h.db, id)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not reach the server")
		return
	}
	defer connection.Close()
	transport := &http.Transport{DisableKeepAlives: true, DialContext: func(context.Context, string, string) (net.Conn, error) {
		return connection.Dial("tcp", "127.0.0.1:"+port)
	}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+port+"/metrics", nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not build the scrape request")
		return
	}
	request.Header.Set("Accept", r.Header.Get("Accept"))
	response, err := transport.RoundTrip(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "the exporter did not respond")
		return
	}
	defer response.Body.Close()
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
