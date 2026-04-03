package api

import (
	"encoding/json"
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/zoekt"
	"github.com/rs/zerolog/log"
)

func (h *ApiHandler) handleZoektListCodebases(w http.ResponseWriter, r *http.Request) {
	codebases, err := h.cmIndex.ListCodebases(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list codebases", err)
		return
	}

	type ZoektCodebaseInfo struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		RootPath  string `json:"rootPath"`
		IndexedAt int64  `json:"indexedAt"`
		FileCount int    `json:"fileCount"`
	}

	var result []ZoektCodebaseInfo
	for _, cb := range codebases {
		metadata, err := h.zIndex.GetStore().ZoektGetMetadataByCodebase(r.Context(), cb.ID)
		if err != nil {
			log.Warn().Str("codebase_id", cb.ID).Err(err).Msg("Failed to get zoekt metadata")
			continue
		}
		if metadata == nil {
			continue
		}

		files, err := h.zIndex.ListFiles(r.Context(), cb.ID)
		if err != nil {
			log.Warn().Str("codebase_id", cb.ID).Err(err).Msg("Failed to list zoekt files")
			continue
		}

		result = append(result, ZoektCodebaseInfo{
			ID:        cb.ID,
			Name:      cb.Name,
			RootPath:  cb.RootPath,
			IndexedAt: metadata.IndexedAt,
			FileCount: len(files),
		})
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ApiHandler) handleZoektStatus(w http.ResponseWriter, r *http.Request) {
	codebaseID := r.URL.Query().Get("codebase_id")
	if codebaseID == "" {
		writeError(w, http.StatusBadRequest, "codebase_id is required", nil)
		return
	}

	metadata, err := h.zIndex.GetStore().ZoektGetMetadataByCodebase(r.Context(), codebaseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get zoekt metadata", err)
		return
	}

	if metadata == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "not_indexed"})
		return
	}

	files, err := h.zIndex.ListFiles(r.Context(), codebaseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list files", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "indexed",
		"indexedAt": metadata.IndexedAt,
		"fileCount": len(files),
	})
}

func (h *ApiHandler) handleZoektListFiles(w http.ResponseWriter, r *http.Request) {
	codebaseID := r.URL.Query().Get("codebase_id")
	if codebaseID == "" {
		writeError(w, http.StatusBadRequest, "codebase_id is required", nil)
		return
	}

	files, err := h.zIndex.ListFiles(r.Context(), codebaseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list files", err)
		return
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *ApiHandler) handleZoektIndex(w http.ResponseWriter, r *http.Request) {
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

	log.Info().Str("dir", req.Dir).Interface("langs", req.Langs).Msg("Zoekt indexing request received")

	taskId, err := h.taskManager.Submit(r.Context(), "zoekt-index", req, 3)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to submit zoekt indexing task", err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "zoekt_indexing_queued",
		"taskId": taskId,
	})
}

func (h *ApiHandler) handleZoektSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query      string `json:"query"`
		CodebaseID string `json:"codebaseID"`
		Limit      int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "Query is required", nil)
		return
	}

	if req.CodebaseID == "" {
		writeError(w, http.StatusBadRequest, "codebaseID is required", nil)
		return
	}

	opts := &zoekt.SearchOptions{
		MaxMatchCount: req.Limit,
	}

	results, err := h.zIndex.Search(r.Context(), req.CodebaseID, req.Query, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Search failed", err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (h *ApiHandler) handleDeleteZoektCodebase(w http.ResponseWriter, r *http.Request) {
	codebaseID := r.URL.Query().Get("codebase_id")
	if codebaseID == "" {
		writeError(w, http.StatusBadRequest, "codebase_id is required", nil)
		return
	}

	log.Info().Str("codebaseID", codebaseID).Msg("Deleting zoekt codebase entries")

	if err := h.zIndex.GetStore().ZoektDeleteCodebase(r.Context(), codebaseID); err != nil {
		log.Error().Err(err).Str("codebaseID", codebaseID).Msg("Failed to delete zoekt codebase")
		writeError(w, http.StatusInternalServerError, "Failed to delete zoekt codebase", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
