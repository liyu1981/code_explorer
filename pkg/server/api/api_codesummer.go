package api

import (
	"encoding/json"
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/db"
)

type CodesummerSummaryResponse struct {
	ID              string `json:"id"`
	CodesummerID    string `json:"codesummerId"`
	NodePath        string `json:"nodePath"`
	NodeType        string `json:"nodeType"`
	Language        string `json:"language"`
	Summary         string `json:"summary"`
	Definitions     any    `json:"definitions"`
	Dependencies    any    `json:"dependencies"`
	DataManipulated any    `json:"dataManipulated"`
	DataFlow        any    `json:"dataFlow"`
	IndexedAt       int64  `json:"indexedAt"`
}

func toCodesummerSummaryResponse(s db.CodesummerSummary) CodesummerSummaryResponse {
	var defs, deps, manip, flow any
	json.Unmarshal([]byte(s.Definitions), &defs)
	json.Unmarshal([]byte(s.Dependencies), &deps)
	json.Unmarshal([]byte(s.DataManipulated), &manip)
	json.Unmarshal([]byte(s.DataFlow), &flow)

	return CodesummerSummaryResponse{
		ID:              s.ID,
		CodesummerID:    s.CodesummerID,
		NodePath:        s.NodePath,
		NodeType:        s.NodeType,
		Language:        s.Language,
		Summary:         s.Summary,
		Definitions:     defs,
		Dependencies:    deps,
		DataManipulated: manip,
		DataFlow:        flow,
		IndexedAt:       s.IndexedAt,
	}
}

func (h *ApiHandler) handleListCodesummerSummaries(w http.ResponseWriter, r *http.Request) {
	codebaseID := r.URL.Query().Get("codebase_id")
	if codebaseID == "" {
		writeError(w, http.StatusBadRequest, "codebase_id is required", nil)
		return
	}

	metadata, err := db.GetStore().CodesummerGetMetadataByCodebase(r.Context(), codebaseID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get codesummer metadata", err)
		return
	}

	if metadata == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"summaries": []CodesummerSummaryResponse{},
			"total":     0,
			"indexedAt": 0,
		})
		return
	}

	summaries, err := db.GetStore().CodesummerListSummaries(r.Context(), metadata.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list summaries", err)
		return
	}

	responses := make([]CodesummerSummaryResponse, len(summaries))
	for i, s := range summaries {
		responses[i] = toCodesummerSummaryResponse(s)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"summaries": responses,
		"total":     len(responses),
		"indexedAt": metadata.IndexedAt,
	})
}
