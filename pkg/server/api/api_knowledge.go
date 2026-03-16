package api

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func (h *ApiHandler) handleListKnowledgePages(w http.ResponseWriter, r *http.Request) {
	codebaseID := r.URL.Query().Get("codebase_id")
	if codebaseID == "" {
		writeError(w, http.StatusBadRequest, "codebase_id is required", nil)
		return
	}

	pages, err := h.index.GetStore().ListKnowledgePages(r.Context(), codebaseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list knowledge pages", err)
		return
	}

	writeJSON(w, http.StatusOK, pages)
}

func (h *ApiHandler) handleGetKnowledgePage(w http.ResponseWriter, r *http.Request) {
	codebaseID := r.URL.Query().Get("codebase_id")
	slug := r.URL.Query().Get("slug")

	if codebaseID == "" {
		writeError(w, http.StatusBadRequest, "codebase_id is required", nil)
		return
	}

	// Default to overview if slug not provided
	if slug == "" {
		slug = "overview"
	}

	log.Debug().Str("codebaseID", codebaseID).Str("slug", slug).Msg("handleGetKnowledgePage")

	page, err := h.index.GetStore().GetKnowledgePageBySlug(r.Context(), codebaseID, slug)

	// If page not found and slug is "overview", create an empty overview page
	if (page == nil || err != nil) && slug == "overview" {
		log.Debug().Str("codebaseID", codebaseID).Str("slug", slug).Msg("overview page not found, creating new one")
		newPage := &db.KnowledgePage{
			CodebaseID:        codebaseID,
			Slug:              "overview",
			Title:             "Overview",
			Content:           "",
			BuildInstructions: "",
		}
		if err := h.index.GetStore().CreateKnowledgePage(r.Context(), newPage); err != nil {
			log.Error().Err(err).Str("codebaseID", codebaseID).Str("slug", slug).Msg("failed to create overview page")
			writeError(w, http.StatusInternalServerError, "failed to create overview page", err)
			return
		}
		log.Debug().Str("codebaseID", codebaseID).Str("slug", slug).Msg("overview page created")
		page, err = h.index.GetStore().GetKnowledgePageBySlug(r.Context(), codebaseID, slug)
	}

	if page == nil {
		writeError(w, http.StatusNotFound, "page not found", nil)
		return
	}

	writeJSON(w, http.StatusOK, page)
}

func (h *ApiHandler) handleCreateKnowledgePage(w http.ResponseWriter, r *http.Request) {
	var page db.KnowledgePage
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if page.CodebaseID == "" || page.Slug == "" || page.Title == "" || page.Content == "" {
		writeError(w, http.StatusBadRequest, "codebase_id, slug, title, and content are required", nil)
		return
	}

	if err := h.index.GetStore().CreateKnowledgePage(r.Context(), &page); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create knowledge page", err)
		return
	}

	writeJSON(w, http.StatusCreated, page)
}

func (h *ApiHandler) handleUpdateKnowledgePage(w http.ResponseWriter, r *http.Request) {
	var page db.KnowledgePage
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if page.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required", nil)
		return
	}

	if err := h.index.GetStore().UpdateKnowledgePage(r.Context(), &page); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update knowledge page", err)
		return
	}

	writeJSON(w, http.StatusOK, page)
}

func (h *ApiHandler) handleDeleteKnowledgePage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required", nil)
		return
	}

	if err := h.index.GetStore().DeleteKnowledgePage(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete knowledge page", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ApiHandler) handleBuildKnowledge(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		CodebaseID string `json:"codebaseId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if payload.CodebaseID == "" {
		writeError(w, http.StatusBadRequest, "codebaseId is required", nil)
		return
	}

	// Submit task
	payloadMap := map[string]any{
		"codebaseId": payload.CodebaseID,
	}

	taskID, err := h.taskManager.Submit(r.Context(), "knowledge-wiki-plan", payloadMap, 3)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to submit knowledge wiki plan task", err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"taskId": taskID,
	})
}
