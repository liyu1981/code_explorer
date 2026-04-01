package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	agentworkflow "github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/liyu1981/code_explorer/pkg/tools"
	"github.com/rs/zerolog/log"
)

type persistenceStreamWriter struct {
	*protocol.StreamWriter
	sessionID string
	turnID    string
	store     *db.Store
}

func (w *persistenceStreamWriter) saveChunk(prefix, data string) {
	if w.turnID == "" {
		return
	}
	chunk := fmt.Sprintf("%s%s\n\n", prefix, data)
	_ = w.store.SaveResearchReportChunk(context.Background(), w.sessionID, w.turnID, chunk)
}

func (w *persistenceStreamWriter) WriteOpenAIChunk(id, model, content string, finishReason *string) error {
	data, _ := json.Marshal(map[string]any{
		"choices": []any{
			map[string]any{
				"delta": map[string]any{
					"content": content,
				},
			},
		},
	})
	w.saveChunk("data: ", string(data))
	return w.StreamWriter.WriteOpenAIChunk(id, model, content, finishReason)
}

func (w *persistenceStreamWriter) WriteCEEvent(event protocol.CEEvent) error {
	data, _ := json.Marshal(event)
	w.saveChunk("ce: ", string(data))
	return w.StreamWriter.WriteCEEvent(event)
}

func (w *persistenceStreamWriter) WriteDone() error {
	w.saveChunk("data: ", "[DONE]")
	return w.StreamWriter.WriteDone()
}

func (w *persistenceStreamWriter) SendReasoning(content string) error {
	return w.WriteCEEvent(protocol.CEEvent{
		Object:  "research.reasoning.delta",
		Content: content,
	})
}

func (w *persistenceStreamWriter) SendTurnStarted(id string, query string, timestamp int64) error {
	w.turnID = id
	return w.WriteCEEvent(protocol.CEEvent{
		Object:    "research.turn.started",
		ID:        id,
		Query:     query,
		Timestamp: timestamp,
	})
}

func (w *persistenceStreamWriter) SendStepUpdate(id string, label string, status protocol.StepStatus) error {
	return w.WriteCEEvent(protocol.CEEvent{
		Object: "research.step.update",
		ID:     id,
		Label:  label,
		Status: status,
	})
}

func (w *persistenceStreamWriter) SendSourceAdded(source protocol.SourceMaterial) error {
	return w.WriteCEEvent(protocol.CEEvent{
		Object: "research.source.added",
		Source: &source,
	})
}

func (w *persistenceStreamWriter) SendResourceMaterial(resource protocol.SourceMaterial) error {
	return w.WriteCEEvent(protocol.CEEvent{
		Object:   "resource.material",
		Resource: &resource,
	})
}

func (w *persistenceStreamWriter) SendToolCall(tool string, params any) error {
	return w.WriteCEEvent(protocol.CEEvent{
		Object: "tool.call.request",
		Tool:   tool,
		Params: params,
	})
}

func (w *persistenceStreamWriter) SendToolResponse(tool string, response any) error {
	return w.WriteCEEvent(protocol.CEEvent{
		Object:   "tool.call.response",
		Tool:     tool,
		Response: response,
	})
}

func (h *ApiHandler) handleAgentResearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query     string `json:"query"`
		SessionID string `json:"sessionId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "Query is required", nil)
		return
	}

	if req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "SessionID is required", nil)
		return
	}

	log.Info().Str("query", req.Query).Str("session", req.SessionID).Msg("Handling agent research request")

	llmCfg := config.Get().System.LLM
	if llmCfg == nil {
		writeError(w, http.StatusInternalServerError, "LLM config not found", nil)
		return
	}

	ai, err := llm.BuildLLM(llmCfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build LLM")
		writeError(w, http.StatusInternalServerError, "Failed to build LLM", err)
		return
	}

	var toolRegistry *tools.ToolRegistry
	sess, err := h.index.GetStore().GetResearchSession(r.Context(), req.SessionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Session not found", err)
		return
	}

	codebase, err := h.index.GetStore().GetCodebaseByID(r.Context(), sess.CodebaseID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Codebase not found", err)
		return
	}

	toolRegistry, err = tools.GetGlobalToolRegistry().Bind(map[string]any{
		"index":   h.index,
		"baseDir": codebase.RootPath,
	})
	log.Debug().Interface("codebase_index", h.index).Str("codebase_basedir", codebase.RootPath).Msg("Bound tool registry with index and codebase root")

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to bind tools", err)
		return
	}

	maxWorkers := 3
	maxIterations := 5
	runner := agentworkflow.NewPEEWorkflowRunner(ai, toolRegistry, maxWorkers, maxIterations)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sw := protocol.NewStreamWriter(w)
	var finalSw protocol.IStreamWriter = sw

	if req.SessionID != "" && h.index != nil {
		finalSw = &persistenceStreamWriter{
			StreamWriter: sw,
			sessionID:    req.SessionID,
			store:        h.index.GetStore(),
		}
	}

	_, err = runner.Run(r.Context(), req.Query, finalSw)
	if err != nil {
		log.Error().Err(err).Msg("PEE workflow failed")
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
	}

	maxReports := config.Get().Research.MaxReportsPerSession
	if maxReports <= 0 {
		maxReports = 50
	}
	_ = h.index.GetStore().PruneReportsBySession(r.Context(), req.SessionID, maxReports)
}

func (h *ApiHandler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	codebaseId := r.URL.Query().Get("codebaseId")
	includeArchived := r.URL.Query().Get("includeArchived") == "true"

	var sessions []db.ResearchSession
	var err error
	if codebaseId != "" {
		sessions, err = h.index.GetStore().GetResearchSessionsByCodebase(r.Context(), codebaseId, includeArchived)
	} else {
		sessions, err = h.index.GetStore().ListResearchSessions(r.Context(), includeArchived)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list sessions", err)
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (h *ApiHandler) handleGetSessionsPaginated(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}

	codebaseId := r.URL.Query().Get("codebaseId")
	page := getIntParam(r, "page", 1)
	pageSize := getIntParam(r, "pageSize", 10)

	sessions, total, err := h.index.GetStore().GetResearchSessionsPaginated(r.Context(), codebaseId, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get sessions", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessions": sessions,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *ApiHandler) handleGetSessionReports(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	id := r.PathValue("id")
	reports, err := h.index.GetStore().GetResearchReportsBySession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get reports", err)
		return
	}
	writeJSON(w, http.StatusOK, reports)
}

func (h *ApiHandler) handleSaveSession(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	var sess db.ResearchSession
	if err := json.NewDecoder(r.Body).Decode(&sess); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := h.index.GetStore().SaveResearchSession(r.Context(), &sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save session", err)
		return
	}

	// Prune sessions for this codebase if limit is reached
	maxSessions := config.Get().Research.MaxReportsPerCodebase
	if maxSessions <= 0 {
		maxSessions = 10
	}
	_ = h.index.GetStore().PruneSessionsByCodebase(r.Context(), sess.CodebaseID, maxSessions)

	writeJSON(w, http.StatusOK, sess)
}

func (h *ApiHandler) handleSummarizeSession(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	id := r.PathValue("id")

	// Submit background task
	payload := map[string]string{
		"sessionId": id,
	}

	taskID, err := h.taskManager.Submit(r.Context(), "summarize-topic", payload, 3)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to submit summarization task", err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"taskId": taskID,
		"status": "queued",
	})
}

func (h *ApiHandler) handleArchiveSession(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	id := r.PathValue("id")
	sess, err := h.index.GetStore().GetResearchSession(r.Context(), id)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "Session not found", nil)
		return
	}

	now := time.Now().UnixMilli()
	sess.ArchivedAt = &now
	if err := h.index.GetStore().SaveResearchSession(r.Context(), sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to archive session", err)
		return
	}

	writeJSON(w, http.StatusOK, sess)
}

func (h *ApiHandler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	id := r.PathValue("id")
	if err := h.index.GetStore().DeleteResearchSession(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete session", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ApiHandler) handleDeleteReport(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	id := r.PathValue("id")
	turnId := r.PathValue("turnId")
	log.Info().Str("sessionId", id).Str("turnId", turnId).Msg("Deleting research report")
	if err := h.index.GetStore().DeleteResearchReport(r.Context(), turnId); err != nil {
		log.Error().Err(err).Str("turnId", turnId).Msg("Failed to delete research report")
		writeError(w, http.StatusInternalServerError, "Failed to delete report", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
