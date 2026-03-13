package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
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

	log.Info().Str("query", req.Query).Str("session", req.SessionID).Msg("Handling agent research request")

	ag, err := h.agentFactory.BuildFromConfig(&agent.Config{
		MaxIterations: 10,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to build agent")
		writeError(w, http.StatusInternalServerError, "Failed to build agent", err)
		return
	}

	// Set headers for streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sw := protocol.NewStreamWriter(w)
	var finalSw protocol.IStreamWriter = sw

	// Wrap if persistence is requested
	if req.SessionID != "" && h.index != nil {
		finalSw = &persistenceStreamWriter{
			StreamWriter: sw,
			sessionID:    req.SessionID,
			store:        h.index.GetStore(),
		}
	}

	// Generate turn ID
	turnID := time.Now().Format("20060102150405") // Or use a proper UUID
	finalSw.SendTurnStarted(turnID, req.Query, time.Now().UnixMilli())

	// Send initial steps
	// We can define standard steps here or let the agent emit them
	finalSw.SendStepUpdate(fmt.Sprintf("turn-%s-thinking", turnID), "Thinking about the research plan", protocol.StepActive)

	// Run agent in a goroutine or directly
	// For streaming, we should run it and let it write to sw
	_, err = ag.RunLoop(r.Context(), req.Query, turnID, finalSw)
	if err != nil {
		// In a stream, we might have already started sending data.
		// Errors should ideally be sent as events.
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
	}

	finalSw.SendStepUpdate(fmt.Sprintf("turn-%s-thinking", turnID), "Thinking about the research plan", protocol.StepCompleted)
	finalSw.WriteDone()

	// Prune reports for this session if limit is reached
	maxReports := config.Get().Research.MaxReportsPerSession
	if maxReports <= 0 {
		maxReports = 50
	}
	_ = h.index.GetStore().PruneReportsBySession(r.Context(), req.SessionID, maxReports)
}

func getSystemLLMConfig() map[string]any {
	llmCfg := make(map[string]any)
	if config.Get().System.LLM != nil {
		for k, v := range config.Get().System.LLM {
			llmCfg[k] = v
		}
	} else {
		llmCfg["type"] = "openai"
		llmCfg["model"] = os.Getenv("LLM_MODEL")
		llmCfg["base_url"] = os.Getenv("LLM_BASE_URL")
	}

	if llmCfg["model"] == nil || llmCfg["model"] == "" {
		llmCfg["model"] = "gpt-4o"
	}
	if llmCfg["base_url"] == nil || llmCfg["base_url"] == "" {
		llmCfg["base_url"] = "https://api.openai.com/v1"
	}
	if llmCfg["type"] == nil || llmCfg["type"] == "" {
		llmCfg["type"] = "openai"
	}
	if config.Get().System.ContextLength > 0 {
		llmCfg["context_length"] = config.Get().System.ContextLength
	} else {
		llmCfg["context_length"] = 262144
	}
	return llmCfg
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
