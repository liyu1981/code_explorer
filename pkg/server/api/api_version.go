package api

import (
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/constant"
)

func (h *ApiHandler) handleVersion(w http.ResponseWriter, r *http.Request) {
	maxReports := config.Get().Research.MaxReportsPerCodebase
	if maxReports <= 0 {
		maxReports = 10
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"version":                   constant.Version,
		"max_sessions_per_codebase": maxReports,
	})
}
