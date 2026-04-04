package api

import (
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func (h *ApiHandler) handleListTasks(w http.ResponseWriter, r *http.Request) {
	page := getIntParam(r, "page", 1)
	pageSize := getIntParam(r, "pageSize", 10)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	tasks, total, err := db.GetStore().GetTasks(r.Context(), pageSize, offset)
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

func (h *ApiHandler) handleGetTaskTree(w http.ResponseWriter, r *http.Request) {
	rootID := r.URL.Query().Get("id")
	if rootID == "" {
		writeError(w, http.StatusBadRequest, "Missing task id", nil)
		return
	}

	tasks, err := db.GetStore().GetTaskTree(r.Context(), rootID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch task tree", err)
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}
