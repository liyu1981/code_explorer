package api

import (
	"encoding/json"
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func (h *ApiHandler) handleListKnowledgePages(w http.ResponseWriter, r *http.Request) {
	codebaseID := r.URL.Query().Get("codebase_id")
	if codebaseID == "" {
		http.Error(w, "codebase_id is required", http.StatusBadRequest)
		return
	}

	pages, err := h.index.GetStore().ListKnowledgePages(r.Context(), codebaseID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(pages)
}

func (h *ApiHandler) handleGetKnowledgePage(w http.ResponseWriter, r *http.Request) {
	codebaseID := r.URL.Query().Get("codebase_id")
	slug := r.URL.Query().Get("slug")
	if codebaseID == "" || slug == "" {
		http.Error(w, "codebase_id and slug are required", http.StatusBadRequest)
		return
	}

	page, err := h.index.GetStore().GetKnowledgePageBySlug(r.Context(), codebaseID, slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if page == nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(page)
}

func (h *ApiHandler) handleCreateKnowledgePage(w http.ResponseWriter, r *http.Request) {
	var page db.KnowledgePage
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if page.CodebaseID == "" || page.Slug == "" || page.Title == "" || page.Content == "" {
		http.Error(w, "codebase_id, slug, title, and content are required", http.StatusBadRequest)
		return
	}

	if err := h.index.GetStore().CreateKnowledgePage(r.Context(), &page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(page)
}

func (h *ApiHandler) handleUpdateKnowledgePage(w http.ResponseWriter, r *http.Request) {
	var page db.KnowledgePage
	if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if page.ID == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := h.index.GetStore().UpdateKnowledgePage(r.Context(), &page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(page)
}

func (h *ApiHandler) handleDeleteKnowledgePage(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := h.index.GetStore().DeleteKnowledgePage(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
