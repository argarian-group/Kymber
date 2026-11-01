package projects

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type projectPayload struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type workspacePayload struct {
	Name string `json:"name"`
}

var entityIDPattern = regexp.MustCompile(`^[0-9a-f]{6}$`)

func pathID(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	id := r.PathValue(name)
	if !entityIDPattern.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return "", false
	}
	return id, true
}

func pathUserID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(r.PathValue("userID"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, "invalid userID")
		return 0, false
	}
	return id, true
}

func projectInput(w http.ResponseWriter, r *http.Request) (projectPayload, bool) {
	var input projectPayload
	if !decodeJSON(w, r, &input) {
		return input, false
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))
	if length := utf8.RuneCountInString(input.Name); length < 1 || length > 120 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 120 characters")
		return input, false
	}
	if len(input.Slug) < 1 || len(input.Slug) > 120 {
		writeError(w, http.StatusBadRequest, "slug must be 1 to 120 characters")
		return input, false
	}
	for _, char := range []byte(input.Slug) {
		if !(char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-') {
			writeError(w, http.StatusBadRequest, "slug may contain only lowercase letters, numbers, and hyphens")
			return input, false
		}
	}
	return input, true
}

func workspaceInput(w http.ResponseWriter, r *http.Request) (workspacePayload, bool) {
	var input workspacePayload
	if !decodeJSON(w, r, &input) {
		return input, false
	}
	input.Name = strings.TrimSpace(input.Name)
	if length := utf8.RuneCountInString(input.Name); length < 1 || length > 120 {
		writeError(w, http.StatusBadRequest, "name must be 1 to 120 characters")
		return input, false
	}
	return input, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request must contain a single JSON value")
		return false
	}
	return true
}
