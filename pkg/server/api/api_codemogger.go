package api

import (
	"encoding/json"
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
)

func (h *ApiHandler) handleListCodebases(w http.ResponseWriter, r *http.Request) {
	codebases, err := h.index.ListCodebases()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list codebases", err)
		return
	}
	writeJSON(w, http.StatusOK, codebases)
}

func (h *ApiHandler) handleListFiles(w http.ResponseWriter, r *http.Request) {
	files, err := h.index.ListFiles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list files", err)
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *ApiHandler) handleIndex(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir   string   `json:"dir"`
		Langs []string `json:"langs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Dir == "" {
		writeError(w, http.StatusBadRequest, "Directory is required", nil)
		return
	}

	opts := &codemogger.IndexOptions{
		Languages: req.Langs,
	}

	opts.Progress = func(current, total int, stage string) {
		h.Publish("index_progress", map[string]any{
			"current": current,
			"total":   total,
			"stage":   stage,
		})
	}

	go func() {
		res, err := h.index.Index(req.Dir, opts)
		if err != nil {
			h.Publish("index_done", map[string]any{"error": err.Error()})
		} else {
			h.Publish("index_done", res)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "indexing started"})
}

func (h *ApiHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string                `json:"query"`
		Limit int                   `json:"limit"`
		Mode  codemogger.SearchMode `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "Query is required", nil)
		return
	}

	opts := &codemogger.SearchOptions{
		Limit:          req.Limit,
		Mode:           req.Mode,
		IncludeSnippet: true,
	}

	results, err := h.index.Search(req.Query, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Search failed", err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}
