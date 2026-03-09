package api

import (
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/constant"
)

// handleVersion returns the current application version
func (h *ApiHandler) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"version":               constant.Version,
		"max_archived_sessions": 10, // Could be from config later
	})
}
