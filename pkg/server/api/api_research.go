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

func (w *persistenceStreamWriter) SendReasoning(turnID string, content string) error {
	return w.WriteCEEvent(protocol.CEEvent{
		TurnID:  turnID,
		Object:  "research.reasoning.delta",
		Content: content,
	})
}

func (w *persistenceStreamWriter) SendTurnStarted(turnID string, query string, timestamp int64) error {
	return w.WriteCEEvent(protocol.CEEvent{
		TurnID:    turnID,
		Object:    "research.turn.started",
		Query:     query,
		Timestamp: timestamp,
	})
}

func (w *persistenceStreamWriter) SendStepUpdate(turnID string, stepID string, label string, status protocol.StepStatus) error {
	return w.WriteCEEvent(protocol.CEEvent{
		TurnID: turnID,
		Object: "research.step.update",
		StepID: stepID,
		Label:  label,
		Status: status,
	})
}

func (w *persistenceStreamWriter) SendSourceAdded(turnID string, source protocol.SourceMaterial) error {
	return w.WriteCEEvent(protocol.CEEvent{
		TurnID: turnID,
		Object: "research.source.added",
		Source: &source,
	})
}

func (w *persistenceStreamWriter) SendResourceMaterial(turnID string, resource protocol.SourceMaterial) error {
	return w.WriteCEEvent(protocol.CEEvent{
		TurnID:   turnID,
		Object:   "resource.material",
		Resource: &resource,
	})
}

func (w *persistenceStreamWriter) SendToolCall(turnID string, tool string, params any) error {
	return w.WriteCEEvent(protocol.CEEvent{
		TurnID: turnID,
		Object: "tool.call.request",
		Tool:   tool,
		Params: params,
	})
}

func (w *persistenceStreamWriter) SendToolResponse(turnID string, tool string, response any) error {
	return w.WriteCEEvent(protocol.CEEvent{
		TurnID:   turnID,
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

	sess, err := db.GetStore().GetResearchSession(r.Context(), req.SessionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Session not found", err)
		return
	}

	codebase, err := db.GetStore().GetCodebaseByID(r.Context(), sess.CodebaseID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Codebase not found", err)
		return
	}

	llmCfg := config.Get().System.LLM
	if llmCfg == nil {
		writeError(w, http.StatusInternalServerError, "LLM config not found", nil)
		return
	}

	llmInstance, err := llm.BuildLLM(llmCfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build LLM")
		writeError(w, http.StatusInternalServerError, "Failed to build LLM", err)
		return
	}

	// maxWorkers := 3
	// maxIterations := 5
	// runner := agentworkflow.NewPEEWorkflowRunner(llmInstance, tools.GetGlobalToolRegistry(), maxWorkers, maxIterations)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sw := protocol.NewStreamWriter(w)
	var finalSw protocol.IStreamWriter = sw

	if req.SessionID != "" {
		finalSw = &persistenceStreamWriter{
			StreamWriter: sw,
			sessionID:    req.SessionID,
			store:        db.GetStore(),
		}
	}

	query := fmt.Sprintf("%s\n\nContext: target codebase id=%s\ntarget basedir=%s", req.Query, codebase.ID, codebase.RootPath)
	log.Info().Str("query", query).Str("session", req.SessionID).Msg("Handling agent research request")

	runner := agentworkflow.NewReactWorkflowRunner(llmInstance, tools.GetGlobalToolRegistry(), agentworkflow.ReactWithMaxIterations(999))
	_, err = runner.Run(r.Context(), query, finalSw)
	if err != nil {
		log.Error().Err(err).Msg("react workflow failed")
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
	}

	maxReports := config.Get().Research.MaxReportsPerSession
	if maxReports <= 0 {
		maxReports = 50
	}
	_ = db.GetStore().PruneReportsBySession(r.Context(), req.SessionID, maxReports)
}

func (h *ApiHandler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	codebaseId := r.URL.Query().Get("codebaseId")
	includeArchived := r.URL.Query().Get("includeArchived") == "true"

	var sessions []db.ResearchSession
	var err error
	if codebaseId != "" {
		sessions, err = db.GetStore().GetResearchSessionsByCodebase(r.Context(), codebaseId, includeArchived)
	} else {
		sessions, err = db.GetStore().ListResearchSessions(r.Context(), includeArchived)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list sessions", err)
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (h *ApiHandler) handleGetSessionsPaginated(w http.ResponseWriter, r *http.Request) {
	codebaseId := r.URL.Query().Get("codebaseId")
	page := getIntParam(r, "page", 1)
	pageSize := getIntParam(r, "pageSize", 10)

	sessions, total, err := db.GetStore().GetResearchSessionsPaginated(r.Context(), codebaseId, page, pageSize)
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
	id := r.PathValue("id")
	reports, err := db.GetStore().GetResearchReportsBySession(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get reports", err)
		return
	}
	writeJSON(w, http.StatusOK, reports)
}

func (h *ApiHandler) handleSaveSession(w http.ResponseWriter, r *http.Request) {
	var sess db.ResearchSession
	if err := json.NewDecoder(r.Body).Decode(&sess); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := db.GetStore().SaveResearchSession(r.Context(), &sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save session", err)
		return
	}

	// Prune sessions for this codebase if limit is reached
	maxSessions := config.Get().Research.MaxReportsPerCodebase
	if maxSessions <= 0 {
		maxSessions = 10
	}
	_ = db.GetStore().PruneSessionsByCodebase(r.Context(), sess.CodebaseID, maxSessions)

	writeJSON(w, http.StatusOK, sess)
}

func (h *ApiHandler) handleSummarizeSession(w http.ResponseWriter, r *http.Request) {
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
	id := r.PathValue("id")
	sess, err := db.GetStore().GetResearchSession(r.Context(), id)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "Session not found", nil)
		return
	}

	now := time.Now().UnixMilli()
	sess.ArchivedAt = &now
	if err := db.GetStore().SaveResearchSession(r.Context(), sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to archive session", err)
		return
	}

	writeJSON(w, http.StatusOK, sess)
}

func (h *ApiHandler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := db.GetStore().DeleteResearchSession(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete session", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ApiHandler) handleDeleteReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	turnId := r.PathValue("turnId")
	log.Info().Str("sessionId", id).Str("turnId", turnId).Msg("Deleting research report")
	if err := db.GetStore().DeleteResearchReport(r.Context(), turnId); err != nil {
		log.Error().Err(err).Str("turnId", turnId).Msg("Failed to delete research report")
		writeError(w, http.StatusInternalServerError, "Failed to delete report", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
