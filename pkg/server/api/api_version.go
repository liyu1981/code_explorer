package api

import (
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/constant"
)

func (h *ApiHandler) handleVersion(w http.ResponseWriter, r *http.Request) {
	maxReports := 10
	if h.index != nil && h.index.Config != nil && h.index.Config.Research.MaxReportsPerCodebase > 0 {
		maxReports = h.index.Config.Research.MaxReportsPerCodebase
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"version":               constant.Version,
		"max_archived_sessions": maxReports,
	})
}
