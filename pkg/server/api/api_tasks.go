package api

import (
	"net/http"
)

func (h *ApiHandler) handleListTasks(w http.ResponseWriter, r *http.Request) {
	page := getIntParam(r, "page", 1)
	pageSize := getIntParam(r, "pageSize", 10)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	tasks, total, err := h.index.GetStore().GetTasks(r.Context(), pageSize, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list tasks", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tasks":    tasks,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
