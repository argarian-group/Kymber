package services

import (
	"net/http"

	"kymber/backend/internal/monitoring"
)

func (h *Handler) serviceMetrics(w http.ResponseWriter, r *http.Request) {
	service, _, ok := h.storageRequest(w, r, "")
	if !ok {
		return
	}
	span, ok := monitoring.ParseRange(r.URL.Query().Get("range"))
	if !ok {
		writeError(w, http.StatusBadRequest, "range must be 1h, 6h, 24h or 7d")
		return
	}
	queries, ok := monitoring.AppQueries(service.ID, service.Domain != "")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid serviceID")
		return
	}
	series, err := monitoring.QueryAll(r.Context(), queries, span)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "metrics are unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"series": series, "http": service.Domain != ""})
}
