package services

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/ssh"
)

const (
	maxEditableFile = 512 << 10
	maxFileEntries  = 2000
)

// Scripts receive paths as positional arguments, so a path never becomes shell code.
const (
	listFilesScript = `d=$1; [ -e "$d" ] || exit 10; [ -d "$d" ] || exit 11; [ -r "$d" ] && [ -x "$d" ] || exit 14
for f in "$d"/* "$d"/.[!.]* "$d"/..?*; do
  [ -e "$f" ] || [ -L "$f" ] || continue
  if [ -d "$f" ]; then t=d; elif [ -f "$f" ]; then t=f; else t=o; fi
  if [ -L "$f" ]; then l=1; else l=0; fi
  printf '%s %s ' "$t" "$l"; stat -c '%s %Y %a %n' "$f" 2>/dev/null || echo
done | head -n 2001`
	readFileScript = `[ -e "$1" ] || exit 10; [ -f "$1" ] || exit 11; [ "$(stat -c %s "$1")" -le 524288 ] || exit 15; stat -c %Y "$1"; head -c 524289 "$1"`
	// The upload lands in a temp file first, so a dropped connection cannot truncate the real file, and writing with cat keeps its owner and mode.
	writeFileScript = `t=$(mktemp) || exit 1; trap 'rm -f "$t"' EXIT; cat > "$t"; [ "$(wc -c < "$t")" -eq "$3" ] || exit 16
if [ -n "$2" ]; then [ -f "$1" ] || exit 10; [ "$(stat -c %Y "$1")" = "$2" ] || exit 12; fi
cat "$t" > "$1" && stat -c %Y "$1"`
	createFileScript   = `[ ! -e "$1" ] && [ ! -L "$1" ] || exit 13; : > "$1"`
	createFolderScript = `[ ! -e "$1" ] && [ ! -L "$1" ] || exit 13; mkdir -- "$1"`
	moveFileScript     = `[ -e "$1" ] || [ -L "$1" ] || exit 10; [ ! -e "$2" ] && [ ! -L "$2" ] || exit 13; mv -- "$1" "$2"`
	deleteFileScript   = `[ -e "$1" ] || [ -L "$1" ] || exit 10; rm -rf -- "$1"`
)

type fileEntry struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Symlink  bool   `json:"symlink,omitempty"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
	Mode     string `json:"mode"`
}

type fileMount struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

func cleanContainerPath(raw string) (string, bool) {
	if raw == "" || raw[0] != '/' || len(raw) > 4096 || path.Clean(raw) != raw || strings.ContainsAny(raw, "\n\x00") {
		return "", false
	}
	return raw, true
}

// shortcut: a file name containing a line break is left out of the listing, parse NUL-separated output if that ever matters.
func parseFileEntries(output string, limit int) ([]fileEntry, bool) {
	entries := make([]fileEntry, 0)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	truncated := len(lines) > limit
	if truncated {
		lines = lines[:limit]
	}
	types := map[string]string{"d": "directory", "f": "file", "o": "other"}
	for _, line := range lines {
		fields := strings.SplitN(line, " ", 6)
		if len(fields) != 6 || types[fields[0]] == "" {
			continue
		}
		size, sizeErr := strconv.ParseInt(fields[2], 10, 64)
		modified, modifiedErr := strconv.ParseInt(fields[3], 10, 64)
		if sizeErr != nil || modifiedErr != nil {
			continue
		}
		entries = append(entries, fileEntry{Name: path.Base(fields[5]), Type: types[fields[0]], Symlink: fields[1] == "1", Size: size, Modified: modified, Mode: fields[4]})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if (entries[i].Type == "directory") != (entries[j].Type == "directory") {
			return entries[i].Type == "directory"
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, truncated
}

func containerCommand(container string, stdin bool, script string, args ...string) string {
	command := "docker exec "
	if stdin {
		command += "-i "
	}
	command += shellQuote(container) + " sh -c " + shellQuote(script) + " sh"
	for _, arg := range args {
		command += " " + shellQuote(arg)
	}
	return command
}

func (h *Handler) fileRequest(w http.ResponseWriter, r *http.Request) (databaseService, bool) {
	service, _, ok := h.storageRequest(w, r, true)
	if !ok {
		return service, false
	}
	if service.Status != "running" || service.ServerID == "" {
		writeError(w, http.StatusConflict, "start the app to browse its files")
		return service, false
	}
	return service, true
}

// runInContainer executes a script and writes the error response itself when it fails.
func (h *Handler) runInContainer(w http.ResponseWriter, r *http.Request, service databaseService, stdin []byte, script string, args ...string) ([]byte, bool) {
	container := serviceTemplate(service).ContainerName(service.Name, service.ID)
	output, err := h.executeSSHInput(r.Context(), service.ServerID, containerCommand(container, stdin != nil, script, args...), stdin)
	if err == nil {
		return output, true
	}
	var exit *ssh.ExitError
	if !errors.As(err, &exit) {
		writeError(w, http.StatusBadGateway, "could not reach the server")
		return nil, false
	}
	switch exit.ExitStatus() {
	case 10:
		writeError(w, http.StatusNotFound, "this path does not exist")
	case 11:
		writeError(w, http.StatusConflict, "this path has the wrong type for this action")
	case 12:
		writeError(w, http.StatusConflict, "the file changed on the server since you opened it")
	case 13:
		writeError(w, http.StatusConflict, "something already exists at this path")
	case 14:
		writeError(w, http.StatusForbidden, "the container's user cannot read this folder")
	case 15:
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("files over %d KB cannot be edited here", maxEditableFile>>10))
	case 16:
		writeError(w, http.StatusBadGateway, "the upload was interrupted and the file was not changed")
	case 126, 127:
		writeError(w, http.StatusConflict, "this image has no shell, so its files cannot be browsed")
	default:
		writeError(w, http.StatusBadGateway, strings.TrimSpace(string(output)))
	}
	return nil, false
}

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	service, ok := h.fileRequest(w, r)
	if !ok {
		return
	}
	directory, ok := cleanContainerPath(r.URL.Query().Get("path"))
	if !ok {
		writeError(w, http.StatusBadRequest, "path must be a clean absolute path")
		return
	}
	output, ok := h.runInContainer(w, r, service, nil, listFilesScript, directory)
	if !ok {
		return
	}
	storages, err := h.readStorages(r.Context(), service.ID, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read storages")
		return
	}
	mounts := make([]fileMount, 0, len(storages)+1)
	if dataPath := serviceTemplate(service).DataPath; dataPath != "" {
		mounts = append(mounts, fileMount{Path: dataPath, Kind: "volume"})
	}
	for _, storage := range storages {
		if storage.Applied {
			mounts = append(mounts, fileMount{Path: storage.MountPath, Kind: storage.Kind})
		}
	}
	entries, truncated := parseFileEntries(string(output), maxFileEntries)
	writeJSON(w, http.StatusOK, map[string]any{"path": directory, "entries": entries, "truncated": truncated, "mounts": mounts})
}

func (h *Handler) readFile(w http.ResponseWriter, r *http.Request) {
	service, ok := h.fileRequest(w, r)
	if !ok {
		return
	}
	file, ok := cleanContainerPath(r.URL.Query().Get("path"))
	if !ok || file == "/" {
		writeError(w, http.StatusBadRequest, "path must be a clean absolute path")
		return
	}
	output, ok := h.runInContainer(w, r, service, nil, readFileScript, file)
	if !ok {
		return
	}
	modified, content, _ := bytes.Cut(output, []byte("\n"))
	if len(content) > maxEditableFile {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("files over %d KB cannot be edited here", maxEditableFile>>10))
		return
	}
	if bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content) {
		writeJSON(w, http.StatusOK, map[string]any{"path": file, "binary": true, "modified": string(modified)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": file, "binary": false, "modified": string(modified), "content": string(content)})
}

func (h *Handler) writeFile(w http.ResponseWriter, r *http.Request) {
	service, ok := h.fileRequest(w, r)
	if !ok {
		return
	}
	var input struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Modified string `json:"modified"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	file, ok := cleanContainerPath(input.Path)
	if !ok || file == "/" {
		writeError(w, http.StatusBadRequest, "path must be a clean absolute path")
		return
	}
	if len(input.Content) > maxEditableFile {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("files over %d KB cannot be edited here", maxEditableFile>>10))
		return
	}
	output, ok := h.runInContainer(w, r, service, []byte(input.Content), writeFileScript, file, input.Modified, strconv.Itoa(len(input.Content)))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"modified": strings.TrimSpace(string(output))})
}

func (h *Handler) createFile(w http.ResponseWriter, r *http.Request) {
	service, ok := h.fileRequest(w, r)
	if !ok {
		return
	}
	var input struct {
		Path string `json:"path"`
		Kind string `json:"kind"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	target, ok := cleanContainerPath(input.Path)
	if !ok || target == "/" || (input.Kind != "file" && input.Kind != "directory") {
		writeError(w, http.StatusBadRequest, "a clean absolute path and a kind are required")
		return
	}
	script := createFileScript
	if input.Kind == "directory" {
		script = createFolderScript
	}
	if _, ok := h.runInContainer(w, r, service, nil, script, target); ok {
		w.WriteHeader(http.StatusCreated)
	}
}

func (h *Handler) moveFile(w http.ResponseWriter, r *http.Request) {
	service, ok := h.fileRequest(w, r)
	if !ok {
		return
	}
	var input struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	from, fromOK := cleanContainerPath(input.From)
	to, toOK := cleanContainerPath(input.To)
	if !fromOK || !toOK || from == "/" || to == "/" {
		writeError(w, http.StatusBadRequest, "clean absolute paths are required")
		return
	}
	if _, ok := h.runInContainer(w, r, service, nil, moveFileScript, from, to); ok {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	service, ok := h.fileRequest(w, r)
	if !ok {
		return
	}
	target, ok := cleanContainerPath(r.URL.Query().Get("path"))
	if !ok || target == "/" {
		writeError(w, http.StatusBadRequest, "this path cannot be deleted")
		return
	}
	if _, ok := h.runInContainer(w, r, service, nil, deleteFileScript, target); ok {
		w.WriteHeader(http.StatusNoContent)
	}
}
