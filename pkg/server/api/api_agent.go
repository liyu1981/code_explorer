package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/liyu1981/code_explorer/pkg/agent"
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
	_ = w.store.SaveResearchReportChunk(w.sessionID, w.turnID, chunk)
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

	// Load agent config from index, env or default
	llmCfg := make(map[string]any)
	if h.index != nil && h.index.Config != nil && h.index.Config.LLM != nil {
		for k, v := range h.index.Config.LLM {
			llmCfg[k] = v
		}
	} else {
		llmCfg["type"] = "openai"
		llmCfg["model"] = os.Getenv("LLM_MODEL")
		llmCfg["endpoint"] = os.Getenv("LLM_ENDPOINT")
	}

	if llmCfg["model"] == nil || llmCfg["model"] == "" {
		llmCfg["model"] = "gpt-4o"
	}
	if llmCfg["endpoint"] == nil || llmCfg["endpoint"] == "" {
		llmCfg["endpoint"] = "https://api.openai.com/v1/chat/completions"
	}
	if llmCfg["type"] == nil || llmCfg["type"] == "" {
		llmCfg["type"] = "openai"
	}

	log.Info().
		Str("type", fmt.Sprintf("%v", llmCfg["type"])).
		Str("model", fmt.Sprintf("%v", llmCfg["model"])).
		Str("endpoint", fmt.Sprintf("%v", llmCfg["endpoint"])).
		Msg("Agent LLM config")

	agentConfig := &agent.Config{
		LLM:           llmCfg,
		MaxIterations: 10,
	}

	ag, err := h.agentFactory.BuildFromConfig(agentConfig)
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
	finalSw.SendStepUpdate("thinking", "Thinking about the research plan", protocol.StepActive)

	// Run agent in a goroutine or directly
	// For streaming, we should run it and let it write to sw
	_, err = ag.Run(r.Context(), req.Query, finalSw)
	if err != nil {
		// In a stream, we might have already started sending data.
		// Errors should ideally be sent as events.
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
	}

	finalSw.SendStepUpdate("thinking", "Thinking about the research plan", protocol.StepCompleted)
	finalSw.WriteDone()
}

func (h *ApiHandler) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	includeArchived := r.URL.Query().Get("includeArchived") == "true"
	sessions, err := h.index.GetStore().ListResearchSessions(includeArchived)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list sessions", err)
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (h *ApiHandler) handleGetSessionReports(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	id := r.PathValue("id")
	reports, err := h.index.GetStore().GetResearchReportsBySession(id)
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

	// Ensure one active session per codebase
	if sess.ArchivedAt == nil {
		existing, _ := h.index.GetStore().GetResearchSessionByCodebase(sess.CodebaseID)
		if existing != nil && existing.ID != sess.ID {
			_ = h.index.GetStore().DeleteResearchSession(existing.ID)
		}
	}

	if err := h.index.GetStore().SaveResearchSession(&sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save session", err)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *ApiHandler) handleArchiveSession(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	id := r.PathValue("id")
	sessions, _ := h.index.GetStore().ListResearchSessions(true)
	var sess *db.ResearchSession
	for _, s := range sessions {
		if s.ID == id {
			sess = &s
			break
		}
	}
	if sess == nil {
		writeError(w, http.StatusNotFound, "Session not found", nil)
		return
	}

	now := time.Now().UnixMilli()
	sess.ArchivedAt = &now
	if err := h.index.GetStore().SaveResearchSession(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to archive session", err)
		return
	}

	// Prune
	maxArchived := 10 // Should fetch from config
	_ = h.index.GetStore().PruneArchivedSessions(maxArchived)

	writeJSON(w, http.StatusOK, sess)
}

func (h *ApiHandler) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	id := r.PathValue("id")
	if err := h.index.GetStore().DeleteResearchSession(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete session", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
