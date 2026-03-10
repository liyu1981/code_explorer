package api

import (
	"encoding/json"
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/rs/zerolog/log"
)

func (h *ApiHandler) handleListCodebases(w http.ResponseWriter, r *http.Request) {
	codebases, err := h.index.ListCodebases()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list codebases", err)
		return
	}
	writeJSON(w, http.StatusOK, codebases)
}

func (h *ApiHandler) handleGetCodemoggerStatus(w http.ResponseWriter, r *http.Request) {
	codebaseID := r.URL.Query().Get("codebase_id")
	if codebaseID == "" {
		writeError(w, http.StatusBadRequest, "codebase_id is required", nil)
		return
	}

	metadata, err := h.index.GetStore().CodemoggerGetMetadataByCodebase(codebaseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get codemogger metadata", err)
		return
	}

	if metadata == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "not_indexed"})
		return
	}

	files, err := h.index.GetStore().CodemoggerListFiles(metadata.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list files", err)
		return
	}

	chunkCount := 0
	for _, f := range files {
		chunkCount += f.ChunkCount
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "indexed",
		"indexedAt":  metadata.IndexedAt,
		"fileCount":  len(files),
		"chunkCount": chunkCount,
	})
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

	log.Info().Str("dir", req.Dir).Interface("langs", req.Langs).Msg("Indexing request received")

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
		log.Debug().Str("dir", req.Dir).Msg("Starting background indexing goroutine")
		res, err := h.index.Index(req.Dir, opts)
		if err != nil {
			log.Error().Err(err).Str("dir", req.Dir).Msg("Background indexing failed")
			h.Publish("index_done", map[string]any{"error": err.Error()})
		} else {
			log.Info().Str("dir", req.Dir).Msg("Background indexing completed successfully")
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
