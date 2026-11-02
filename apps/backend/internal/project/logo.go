package project

import (
	"bytes"
	"database/sql"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strings"

	"kymber/backend/internal/auth"
	"kymber/backend/internal/database"
	"kymber/backend/internal/permissions"
)

const maxLogoBytes = 5 << 20

func detectLogoType(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png"
	case bytes.HasPrefix(data, []byte("\xff\xd8\xff")):
		return "image/jpeg"
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		return "image/gif"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp"
	case isAVIF(data):
		return "image/avif"
	case bytes.HasPrefix(data, []byte("BM")):
		return "image/bmp"
	case bytes.HasPrefix(data, []byte("\x00\x00\x01\x00")):
		return "image/x-icon"
	case isSVG(data):
		return "image/svg+xml"
	}
	return ""
}

func isAVIF(data []byte) bool {
	if len(data) < 12 || string(data[4:8]) != "ftyp" {
		return false
	}
	size := int(data[0])<<24 | int(data[1])<<16 | int(data[2])<<8 | int(data[3])
	if size < 16 || size > len(data) {
		size = min(len(data), 64)
	}
	for offset := 8; offset+4 <= size; offset += 4 {
		if brand := string(data[offset : offset+4]); brand == "avif" || brand == "avis" {
			return true
		}
	}
	return false
}

func isSVG(data []byte) bool {
	decoder := xml.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
	decoder.Strict = false
	for {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		if element, ok := token.(xml.StartElement); ok {
			return strings.EqualFold(element.Name.Local, "svg")
		}
	}
}

func (h *Handler) authorizeProjectAccess(w http.ResponseWriter, r *http.Request, permission string) (string, string, bool) {
	user, _ := auth.UserFromContext(r)
	workspaceID, ok := pathID(w, r, "workspaceID")
	if !ok || !h.authorize(w, r, user, workspaceID, permission) {
		return "", "", false
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok || !h.authorizeProject(w, r, workspaceID, projectID) {
		return "", "", false
	}
	return workspaceID, projectID, true
}

func (h *Handler) uploadProjectLogo(w http.ResponseWriter, r *http.Request) {
	workspaceID, projectID, ok := h.authorizeProjectAccess(w, r, permissions.ProjectsManage)
	if !ok {
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxLogoBytes))
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "logo must be 5 MB or smaller")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read logo")
		return
	}
	contentType := detectLogoType(data)
	if contentType == "" {
		writeError(w, http.StatusUnsupportedMediaType, "logo must be a PNG, JPEG, WebP, GIF, AVIF, SVG, ICO or BMP file")
		return
	}
	version, err := database.NewShortID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save logo")
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "INSERT INTO project_logos (project_id, content_type, version, data) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE content_type = VALUES(content_type), version = VALUES(version), data = VALUES(data)", projectID, contentType, version, data); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save logo")
		return
	}
	project, err := h.findProject(r, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read project")
		return
	}
	writeJSON(w, http.StatusOK, map[string]Project{"project": project})
}

func (h *Handler) getProjectLogo(w http.ResponseWriter, r *http.Request) {
	_, projectID, ok := h.authorizeProjectAccess(w, r, "")
	if !ok {
		return
	}
	var contentType string
	var data []byte
	err := h.db.QueryRowContext(r.Context(), "SELECT content_type, data FROM project_logos WHERE project_id = ?", projectID).Scan(&contentType, &data)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "project has no logo")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read logo")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	_, _ = w.Write(data)
}

func (h *Handler) deleteProjectLogo(w http.ResponseWriter, r *http.Request) {
	workspaceID, projectID, ok := h.authorizeProjectAccess(w, r, permissions.ProjectsManage)
	if !ok {
		return
	}
	if _, err := h.db.ExecContext(r.Context(), "DELETE FROM project_logos WHERE project_id = ?", projectID); err != nil {
		writeError(w, http.StatusInternalServerError, "could not remove logo")
		return
	}
	project, err := h.findProject(r, workspaceID, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read project")
		return
	}
	writeJSON(w, http.StatusOK, map[string]Project{"project": project})
}
