package api

import (
	"encoding/json"
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/db"
)

type SavedReportWithSources struct {
	ID           string `json:"id"`
	SessionID    string `json:"sessionId"`
	CodebaseID   string `json:"codebaseId"`
	Title        string `json:"title"`
	Query        string `json:"query"`
	StreamData   string `json:"streamData"`
	CodebaseName string `json:"codebaseName"`
	CodebasePath string `json:"codebasePath"`
	CreatedAt    int64  `json:"createdAt"`
}

func (h *ApiHandler) handleSaveSavedReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID    string `json:"sessionId"`
		CodebaseID   string `json:"codebaseId"`
		Title        string `json:"title"`
		Query        string `json:"query"`
		TurnID       string `json:"turnId"`
		StreamData   string `json:"streamData"`
		CodebaseName string `json:"codebaseName"`
		CodebasePath string `json:"codebasePath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	streamData := req.StreamData

	// If turnId is provided, fetch stream_data from research_reports
	if req.TurnID != "" && streamData == "" {
		reports, err := db.GetStore().GetResearchReportsBySession(r.Context(), req.SessionID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to get research reports", err)
			return
		}
		for _, rp := range reports {
			if rp.TurnID == req.TurnID {
				streamData = rp.StreamData
				break
			}
		}
	}

	if streamData == "" {
		writeError(w, http.StatusBadRequest, "streamData or turnId is required", nil)
		return
	}

	report := &db.SavedReport{
		SessionID:    req.SessionID,
		CodebaseID:   req.CodebaseID,
		Title:        req.Title,
		Query:        req.Query,
		StreamData:   streamData,
		CodebaseName: req.CodebaseName,
		CodebasePath: req.CodebasePath,
	}

	if err := db.GetStore().SaveSavedReport(r.Context(), report); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save report", err)
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (h *ApiHandler) handleGetSavedReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	report, err := db.GetStore().GetSavedReport(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get report", err)
		return
	}

	if report == nil {
		writeError(w, http.StatusNotFound, "Report not found", nil)
		return
	}

	response := SavedReportWithSources{
		ID:           report.ID,
		SessionID:    report.SessionID,
		CodebaseID:   report.CodebaseID,
		Title:        report.Title,
		Query:        report.Query,
		StreamData:   report.StreamData,
		CodebaseName: report.CodebaseName,
		CodebasePath: report.CodebasePath,
		CreatedAt:    report.CreatedAt,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *ApiHandler) handleListSavedReports(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page := getIntParam(r, "page", 1)
	pageSize := getIntParam(r, "pageSize", 10)

	reports, total, err := db.GetStore().ListSavedReports(r.Context(), page, pageSize, query)
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
	id := r.PathValue("id")
	if err := db.GetStore().DeleteSavedReport(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete report", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
