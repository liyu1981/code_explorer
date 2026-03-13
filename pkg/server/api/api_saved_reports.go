package api

import (
	"encoding/json"
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func (h *ApiHandler) handleSaveSavedReport(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}

	var report db.SavedReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := h.index.GetStore().SaveSavedReport(r.Context(), &report); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save report", err)
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (h *ApiHandler) handleGetSavedReport(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}

	id := r.PathValue("id")
	report, err := h.index.GetStore().GetSavedReport(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get report", err)
		return
	}

	if report == nil {
		writeError(w, http.StatusNotFound, "Report not found", nil)
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (h *ApiHandler) handleListSavedReports(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}

	query := r.URL.Query().Get("q")
	page := getIntParam(r, "page", 1)
	pageSize := getIntParam(r, "pageSize", 10)

	reports, total, err := h.index.GetStore().ListSavedReports(r.Context(), page, pageSize, query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list reports", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"reports":  reports,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *ApiHandler) handleDeleteSavedReport(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}

	id := r.PathValue("id")
	if err := h.index.GetStore().DeleteSavedReport(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete report", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
