package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
	"kymber/backend/internal/auth"
	"kymber/backend/internal/logger"
)

const (
	maxTerminalsPerUser = 5
	terminalIdleLimit   = 30 * time.Minute
	terminalMaxLifetime = 8 * time.Hour
	terminalPing        = 30 * time.Second
)

// The upgrader's default origin check rejects other sites, which matters because a socket is authenticated by cookie alone.
var terminalUpgrader = websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 4096}

type terminalSlots struct {
	mu   sync.Mutex
	open map[uint64]int
}

var openTerminals = terminalSlots{open: map[uint64]int{}}

func (s *terminalSlots) acquire(userID uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open[userID] >= maxTerminalsPerUser {
		return false
	}
	s.open[userID]++
	return true
}

func (s *terminalSlots) release(userID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.open[userID]--; s.open[userID] <= 0 {
		delete(s.open, userID)
	}
}

func parseTerminalSize(message []byte) (cols, rows int, ok bool) {
	var size struct {
		Cols int `json:"cols"`
		Rows int `json:"rows"`
	}
	if json.Unmarshal(message, &size) != nil || size.Cols < 1 || size.Cols > 1000 || size.Rows < 1 || size.Rows > 500 {
		return 0, 0, false
	}
	return size.Cols, size.Rows, true
}

// docker exec leaves its process running when the client disconnects, so every session is tagged to find and stop it afterwards.
func containerShellCommand(container, sessionID string) string {
	return "docker exec -it -e TERM=xterm-256color -e KYMBER_TERMINAL=" + sessionID + " " + shellQuote(container) + " sh -c " + shellQuote("if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi")
}

// The script's own environment lacks the tag, so it never matches itself.
func containerShellCleanup(container, sessionID string) string {
	script := `pids=; for p in /proc/[0-9]*; do tr '\0' '\n' < "$p/environ" 2>/dev/null | grep -qx "KYMBER_TERMINAL=$1" && pids="$pids ${p#/proc/}"; done; [ -z "$pids" ] && exit 0; kill -HUP $pids 2>/dev/null; sleep 1; kill -KILL $pids 2>/dev/null; exit 0`
	return containerCommand(container, false, script, sessionID)
}

// terminalAllowed answers whether a session may stay open. A database error keeps it open, since it says nothing about the user's access.
func (h *Handler) terminalAllowed(r *http.Request, workspaceID string) bool {
	user, active, err := auth.ActiveUser(h.db, r)
	if err != nil {
		return true
	}
	if !active {
		return false
	}
	if user.IsRoot {
		return true
	}
	if workspaceID == "" {
		return false
	}
	var role string
	if err := h.db.QueryRowContext(r.Context(), "SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?", workspaceID, user.ID).Scan(&role); err != nil {
		return !errors.Is(err, sql.ErrNoRows)
	}
	return role == "owner" || role == "admin"
}

func (h *Handler) serverTerminal(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r)
	// A server shell reaches the whole machine and every workspace on it.
	if !user.IsRoot {
		writeError(w, http.StatusForbidden, "root access required")
		return
	}
	serverID, ok := pathID(w, r, "serverID")
	if !ok {
		return
	}
	h.runTerminal(w, r, user, "", serverID, "", "")
}

func (h *Handler) serviceTerminal(w http.ResponseWriter, r *http.Request) {
	service, ok := h.fileRequest(w, r)
	if !ok {
		return
	}
	user, _ := auth.UserFromContext(r)
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		writeError(w, http.StatusInternalServerError, "could not start the terminal")
		return
	}
	sessionID := hex.EncodeToString(raw)
	container := serviceTemplate(service).ContainerName(service.Name, service.ID)
	h.runTerminal(w, r, user, service.WorkspaceID, service.ServerID, containerShellCommand(container, sessionID), containerShellCleanup(container, sessionID))
}

// runTerminal bridges a WebSocket to an SSH PTY: binary frames carry terminal bytes and text frames carry resizes. An empty command opens the login shell.
// An empty workspace means a server shell, which only the root account may keep.
func (h *Handler) runTerminal(w http.ResponseWriter, r *http.Request, user auth.User, workspaceID, serverID, command, cleanup string) {
	if !openTerminals.acquire(user.ID) {
		writeError(w, http.StatusTooManyRequests, "too many open terminals; close one and try again")
		return
	}
	defer openTerminals.release(user.ID)
	// Upgrading first means a rejected origin never starts a shell and connection errors reach the terminal.
	conn, err := terminalUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	fail := func(reason string) {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, reason), time.Now().Add(5*time.Second))
	}
	client, err := h.connectSSH(r.Context(), serverID)
	if err != nil {
		fail("could not reach the server")
		return
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		fail("could not open a session on the server")
		return
	}
	defer session.Close()
	stdin, stdinErr := session.StdinPipe()
	stdout, stdoutErr := session.StdoutPipe()
	if err := errors.Join(stdinErr, stdoutErr, session.RequestPty("xterm-256color", 24, 80, ssh.TerminalModes{ssh.ECHO: 1})); err != nil {
		fail("could not open a terminal on the server")
		return
	}
	if command == "" {
		err = session.Shell()
	} else {
		err = session.Start(command)
	}
	if err != nil {
		fail("could not start the shell")
		return
	}
	if cleanup != "" {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if _, err := h.executeSSH(ctx, serverID, cleanup); err != nil {
				logger.Warn("terminal cleanup failed: server=%s: %v", serverID, err)
			}
		}()
	}
	logger.Info("terminal opened: user=%d server=%s container=%t", user.ID, serverID, command != "")
	defer logger.Info("terminal closed: user=%d server=%s", user.ID, serverID)

	var writeMu sync.Mutex
	send := func(kind int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteMessage(kind, data)
	}
	end := func(reason string) {
		_ = send(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason))
		_ = conn.Close()
		_ = client.Close()
	}

	var activityMu sync.Mutex
	lastInput := time.Now()
	done := make(chan struct{})
	go func() {
		defer close(done)
		buffer := make([]byte, 32<<10)
		for {
			n, err := stdout.Read(buffer)
			if n > 0 && send(websocket.BinaryMessage, buffer[:n]) != nil {
				return
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					end("session ended")
				}
				return
			}
		}
	}()
	go func() {
		started := time.Now()
		ticker := time.NewTicker(terminalPing)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				activityMu.Lock()
				idle := time.Since(lastInput)
				activityMu.Unlock()
				if idle > terminalIdleLimit {
					end("closed after 30 minutes without input")
					return
				}
				if time.Since(started) > terminalMaxLifetime {
					end("closed after reaching the 8 hour limit")
					return
				}
				if !h.terminalAllowed(r, workspaceID) {
					end("closed because your access changed")
					return
				}
				if send(websocket.PingMessage, nil) != nil {
					return
				}
			}
		}
	}()

	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(2 * terminalPing))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(2 * terminalPing)) })
	for {
		kind, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * terminalPing))
		if kind == websocket.TextMessage {
			if cols, rows, ok := parseTerminalSize(message); ok {
				_ = session.WindowChange(rows, cols)
			}
			continue
		}
		activityMu.Lock()
		lastInput = time.Now()
		activityMu.Unlock()
		if _, err := stdin.Write(message); err != nil {
			break
		}
	}
	// Closing the client hangs up a server shell; a container shell is stopped by the cleanup above.
	_ = client.Close()
	<-done
}
