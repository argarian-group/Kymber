package servers

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/ssh"
	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/monitoring"
	"kymber/backend/internal/secrets"
)

const serverStatusSQL = `CASE
	WHEN s.connection_type = 'ssh' THEN 'online'
	WHEN s.connection_type = 'local' AND s.encrypted_ssh_key IS NOT NULL THEN 'online'
	WHEN s.last_seen IS NULL THEN 'pending'
	WHEN s.last_seen >= UTC_TIMESTAMP(6) - INTERVAL 30 SECOND OR EXISTS(SELECT 1 FROM server_tasks t WHERE t.server_id = s.id AND t.status = 'running') THEN 'online'
	ELSE 'offline'
END`

type serverInfo struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Type          string     `json:"connection_type"`
	Host          string     `json:"host,omitempty"`
	Port          int        `json:"ssh_port,omitempty"`
	SSHUser       string     `json:"ssh_user,omitempty"`
	SSHConfigured bool       `json:"ssh_configured"`
	Status        string     `json:"status"`
	LastSeen      *time.Time `json:"last_seen"`
	CreatedAt     time.Time  `json:"created_at"`
	Apps          int        `json:"apps"`
	Monitoring    bool       `json:"monitoring"`
}

func (h *Handler) requireRoot(w http.ResponseWriter, r *http.Request) bool {
	user, _ := auth.UserFromContext(r)
	if !user.IsRoot {
		writeError(w, http.StatusForbidden, "root access required")
		return false
	}
	return true
}

func (h *Handler) listServers(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT s.id, s.name, s.connection_type, COALESCE(s.host, ''), s.ssh_port, COALESCE(s.ssh_user, ''), s.encrypted_ssh_key IS NOT NULL, "+serverStatusSQL+", s.last_seen, s.created_at, (SELECT COUNT(*) FROM database_services d WHERE d.server_id = s.id AND d.status <> 'deleted'), s.monitoring_installed_at IS NOT NULL FROM servers s ORDER BY s.created_at")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list servers")
		return
	}
	defer rows.Close()
	items := make([]serverInfo, 0)
	for rows.Next() {
		var s serverInfo
		if err := rows.Scan(&s.ID, &s.Name, &s.Type, &s.Host, &s.Port, &s.SSHUser, &s.SSHConfigured, &s.Status, &s.LastSeen, &s.CreatedAt, &s.Apps, &s.Monitoring); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read servers")
			return
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read servers")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]serverInfo{"servers": items})
}

func (h *Handler) listDeployTargets(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, true) {
		return
	}
	rows, err := h.db.QueryContext(r.Context(), "SELECT s.id, s.name FROM servers s WHERE "+serverStatusSQL+" = 'online' ORDER BY s.created_at")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list servers")
		return
	}
	defer rows.Close()
	type target struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	targets := make([]target, 0)
	for rows.Next() {
		var item target
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			writeError(w, http.StatusInternalServerError, "could not read servers")
			return
		}
		targets = append(targets, item)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read servers")
		return
	}
	writeJSON(w, http.StatusOK, map[string][]target{"servers": targets})
}

func (h *Handler) requireServerOnline(w http.ResponseWriter, r *http.Request, id string) bool {
	var online bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM servers s WHERE s.id = ? AND (s.connection_type = 'ssh' OR "+serverStatusSQL+" = 'online'))", id).Scan(&online); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check server")
		return false
	}
	if !online {
		writeError(w, http.StatusConflict, "server is offline")
		return false
	}
	return true
}

func serverName(w http.ResponseWriter, r *http.Request) (string, bool) {
	var input struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &input) {
		return "", false
	}
	name := strings.TrimSpace(input.Name)
	if length := utf8.RuneCountInString(name); length < 1 || length > 120 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 120 characters")
		return "", false
	}
	return name, true
}

func (h *Handler) createServer(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	user, _ := auth.UserFromContext(r)
	var input struct {
		Name    string `json:"name"`
		Type    string `json:"connection_type"`
		Host    string `json:"host"`
		Port    int    `json:"ssh_port"`
		User    string `json:"ssh_user"`
		HostKey string `json:"ssh_host_key"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Name, input.Host, input.User = strings.TrimSpace(input.Name), strings.TrimSpace(input.Host), strings.TrimSpace(input.User)
	if input.Type == "" {
		input.Type = "local"
	}
	if n := utf8.RuneCountInString(input.Name); n < 1 || n > 120 || (input.Type != "local" && input.Type != "ssh") {
		writeError(w, http.StatusBadRequest, "valid name and connection_type are required")
		return
	}
	id, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate server id")
		return
	}
	info := serverInfo{ID: id, Name: input.Name, Type: input.Type, Status: "pending", CreatedAt: time.Now().UTC()}
	if input.Type == "local" {
		encrypted, hostKey, err := h.provisionLocalSSH(r)
		if err != nil {
			writeError(w, http.StatusBadGateway, "could not provision local SSH access: "+err.Error())
			return
		}
		if _, err := h.db.ExecContext(r.Context(), "INSERT INTO servers (id, name, token_hash, connection_type, host, ssh_port, ssh_user, encrypted_ssh_key, ssh_host_key, created_by) VALUES (?, ?, NULL, 'local', ?, 22, 'root', ?, ?, ?)", id, input.Name, localSSHHost(), encrypted, hostKey.Marshal(), user.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "could not register server")
			return
		}
		info.SSHConfigured = true
		writeJSON(w, http.StatusCreated, map[string]any{"server": info})
		return
	}
	if input.Port == 0 {
		input.Port = 22
	}
	if input.Port < 1 || input.Port > 65535 || !sshHostPattern.MatchString(input.Host) || !sshUserPattern.MatchString(input.User) {
		writeError(w, http.StatusBadRequest, "valid SSH host, port, and user are required")
		return
	}
	hostKey, err := parseSSHHostKey(input.HostKey)
	if err != nil {
		writeError(w, http.StatusBadRequest, "valid SSH host key is required")
		return
	}
	privateKey, publicKey, err := generateSSHKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate SSH key")
		return
	}
	encrypted, err := secrets.Encrypt(privateKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SSH key encryption is not configured")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO servers (id, name, connection_type, host, ssh_port, ssh_user, encrypted_ssh_key, ssh_host_key, created_by) VALUES (?, ?, 'ssh', ?, ?, ?, ?, ?, ?)", id, input.Name, input.Host, input.Port, input.User, encrypted, hostKey.Marshal(), user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not register server")
		return
	}
	info.Type, info.Host, info.Port, info.SSHUser = "ssh", input.Host, input.Port, input.User
	info.SSHConfigured = true
	writeJSON(w, http.StatusCreated, map[string]any{"server": info, "public_key": publicKey})
}

func (h *Handler) provisionLocalSSH(r *http.Request) ([]byte, ssh.PublicKey, error) {
	privateKey, err := os.ReadFile(os.Getenv("KYMBER_LOCAL_SSH_PRIVATE_KEY_FILE"))
	if err != nil {
		return nil, nil, err
	}
	hostKeyData, err := os.ReadFile(os.Getenv("KYMBER_LOCAL_SSH_HOST_KEY_FILE"))
	if err != nil {
		return nil, nil, err
	}
	hostKey, err := parseSSHHostKey(string(hostKeyData))
	if err != nil {
		return nil, nil, err
	}
	encrypted, err := secrets.Encrypt(privateKey)
	if err != nil {
		return nil, nil, err
	}
	if err := validateSSHAccess(r.Context(), localSSHHost(), 22, "root", privateKey, hostKey); err != nil {
		return nil, nil, err
	}
	return encrypted, hostKey, nil
}

func localSSHHost() string {
	if host := strings.TrimSpace(os.Getenv("KYMBER_LOCAL_SSH_HOST")); host != "" {
		return host
	}
	return "host.docker.internal"
}

func (h *Handler) setupLocalSSH(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	id, ok := pathID(w, r, "serverID")
	if !ok {
		return
	}
	var connectionType string
	var configured bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT connection_type, encrypted_ssh_key IS NOT NULL FROM servers WHERE id = ?", id).Scan(&connectionType, &configured); errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "server not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read server")
		return
	}
	if connectionType != "local" || configured {
		writeError(w, http.StatusConflict, "server is not an unconfigured local server")
		return
	}
	encrypted, hostKey, err := h.provisionLocalSSH(r)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not provision local SSH access: "+err.Error())
		return
	}
	result, err := h.db.ExecContext(r.Context(), "UPDATE servers SET host = ?, ssh_port = 22, ssh_user = 'root', encrypted_ssh_key = ?, ssh_host_key = ? WHERE id = ? AND connection_type = 'local' AND encrypted_ssh_key IS NULL", localSSHHost(), encrypted, hostKey.Marshal(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save local SSH credentials")
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		writeError(w, http.StatusConflict, "server changed during SSH setup")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

var sshHostPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]{0,252}[a-zA-Z0-9]$|^[a-zA-Z0-9]$`)
var sshUserPattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)

func parseSSHHostKey(line string) (ssh.PublicKey, error) {
	if key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(line))); err == nil {
		return key, nil
	}
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return nil, errors.New("invalid host key")
	}
	data, err := base64.StdEncoding.DecodeString(fields[2])
	if err != nil {
		return nil, err
	}
	key, err := ssh.ParsePublicKey(data)
	if err != nil || key.Type() != fields[1] {
		return nil, errors.New("invalid host key")
	}
	return key, nil
}

func generateSSHKey() ([]byte, string, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", err
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return nil, "", err
	}
	publicKey, err := ssh.NewPublicKey(public)
	if err != nil {
		return nil, "", err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}), string(ssh.MarshalAuthorizedKey(publicKey)), nil
}

func newServerToken() (string, [32]byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", [32]byte{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, sha256.Sum256([]byte(token)), nil
}

func (h *Handler) renameServer(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	id, ok := pathID(w, r, "serverID")
	if !ok {
		return
	}
	name, ok := serverName(w, r)
	if !ok {
		return
	}
	var exists bool
	if err := h.db.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM servers WHERE id = ?)", id).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "could not read server")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "UPDATE servers SET name = ? WHERE id = ?", name, id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not rename server")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) rotateServerToken(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	id, ok := pathID(w, r, "serverID")
	if !ok {
		return
	}
	token, digest, err := newServerToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate server token")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not replace server token")
		return
	}
	defer tx.Rollback()
	var connectionType string
	var current []byte
	err = tx.QueryRowContext(r.Context(), "SELECT connection_type, token_hash FROM servers WHERE id = ? FOR UPDATE", id).Scan(&connectionType, &current)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not replace server token")
		return
	}
	if connectionType != "local" {
		writeError(w, http.StatusConflict, "SSH servers do not use agent tokens")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "UPDATE servers SET token_hash = ?, last_seen = NULL, status = 'pending' WHERE id = ?", digest[:], id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not replace server token")
		return
	}
	if isLocalServer(current) {
		if err := writeTokenFile(os.Getenv("KYMBER_SERVER_TOKEN_FILE"), token); err != nil {
			writeError(w, http.StatusInternalServerError, "could not provision local server token")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not replace server token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *Handler) deleteServer(w http.ResponseWriter, r *http.Request) {
	if !h.requireRoot(w, r) {
		return
	}
	id, ok := pathID(w, r, "serverID")
	if !ok {
		return
	}
	var apps int
	if err := h.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM database_services WHERE server_id = ? AND status <> 'deleted'", id).Scan(&apps); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check server apps")
		return
	}
	if apps > 0 {
		writeError(w, http.StatusConflict, "delete this server's apps first")
		return
	}
	monitoring.Remove(r.Context(), h.db, id)
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete server")
		return
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM database_services WHERE server_id = ? AND status <> 'deleted' FOR UPDATE", id).Scan(&apps); err != nil {
		writeError(w, http.StatusInternalServerError, "could not check server apps")
		return
	}
	if apps > 0 {
		writeError(w, http.StatusConflict, "delete this server's apps first")
		return
	}
	for _, query := range []string{"DELETE FROM server_tasks WHERE server_id = ?", "DELETE FROM database_services WHERE server_id = ? AND status = 'deleted'"} {
		if _, err := tx.ExecContext(r.Context(), query, id); err != nil {
			writeError(w, http.StatusInternalServerError, "could not delete server")
			return
		}
	}
	var current []byte
	var connectionType string
	err = tx.QueryRowContext(r.Context(), "SELECT connection_type, token_hash FROM servers WHERE id = ?", id).Scan(&connectionType, &current)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete server")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "DELETE FROM servers WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete server")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete server")
		return
	}
	if connectionType == "local" && isLocalServer(current) {
		_ = os.Remove(os.Getenv("KYMBER_SERVER_TOKEN_FILE"))
	}
	w.WriteHeader(http.StatusNoContent)
}
