package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
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

	// Load agent config from system config, env or default
	llmCfg := getSystemLLMConfig()

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
	finalSw.SendStepUpdate(fmt.Sprintf("turn-%s-thinking", turnID), "Thinking about the research plan", protocol.StepActive)

	// Run agent in a goroutine or directly
	// For streaming, we should run it and let it write to sw
	_, err = ag.Run(r.Context(), req.Query, turnID, finalSw)
	if err != nil {
		// In a stream, we might have already started sending data.
		// Errors should ideally be sent as events.
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
	}

	finalSw.SendStepUpdate(fmt.Sprintf("turn-%s-thinking", turnID), "Thinking about the research plan", protocol.StepCompleted)
	finalSw.WriteDone()
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
		sessions, err = h.index.GetStore().GetResearchSessionsByCodebase(codebaseId, includeArchived)
	} else {
		sessions, err = h.index.GetStore().ListResearchSessions(includeArchived)
	}

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

	if err := h.index.GetStore().SaveResearchSession(&sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save session", err)
		return
	}

	// Prune sessions for this codebase if limit is reached
	maxSessions := config.Get().Research.MaxReportsPerCodebase
	if maxSessions <= 0 {
		maxSessions = 10
	}
	_ = h.index.GetStore().PruneSessionsByCodebase(sess.CodebaseID, maxSessions)

	writeJSON(w, http.StatusOK, sess)
}

func (h *ApiHandler) handleSummarizeSession(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}
	id := r.PathValue("id")

	// Get reports to find the first question and part of the report
	reports, err := h.index.GetStore().GetResearchReportsBySession(id)
	if err != nil || len(reports) == 0 {
		writeError(w, http.StatusNotFound, "Reports not found for session", err)
		return
	}

	// Reconstruct the first turn context
	var firstQuery string
	var firstReport string
	lines := strings.Split(reports[0].StreamData, "\n\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ce: ") {
			var event protocol.CEEvent
			if err := json.Unmarshal([]byte(line[4:]), &event); err == nil && event.Object == "research.turn.started" {
				firstQuery = event.Query
			}
		} else if strings.HasPrefix(line, "data: ") {
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(line[6:]), &chunk); err == nil && len(chunk.Choices) > 0 {
				firstReport += chunk.Choices[0].Delta.Content
			}
		}
		if len(firstReport) > 500 {
			break
		}
	}

	// Use LLM to summarize
	llmCfg := getSystemLLMConfig()
	model := llmCfg["model"].(string)
	endpoint := llmCfg["endpoint"].(string)
	apiKey := ""
	if v, ok := llmCfg["api_key"].(string); ok {
		apiKey = v
	}

	llm := agent.NewHTTPClientLLM(model, endpoint, apiKey)
	prompt := fmt.Sprintf("Based on the following research query and partial report, generate a concise title (strictly maximum 5 words).\n\nQuery: %s\n\nReport: %s", firstQuery, firstReport)
	title, _, err := llm.Generate(r.Context(), []agent.Message{{Role: "user", Content: prompt}}, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate title with LLM")
		writeError(w, http.StatusInternalServerError, "Failed to generate title", err)
		return
	}

	title = strings.Trim(title, "\" \n\r")

	// Update session title
	sess, err := h.index.GetStore().GetResearchSession(id)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "Session not found", err)
		return
	}

	sess.Title = title
	if err := h.index.GetStore().SaveResearchSession(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update session title", err)
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
	sess, err := h.index.GetStore().GetResearchSession(id)
	if err != nil || sess == nil {
		writeError(w, http.StatusNotFound, "Session not found", nil)
		return
	}

	now := time.Now().UnixMilli()
	sess.ArchivedAt = &now
	if err := h.index.GetStore().SaveResearchSession(sess); err != nil {
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
	if err := h.index.GetStore().DeleteResearchSession(id); err != nil {
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
	if err := h.index.GetStore().DeleteResearchReport(turnId); err != nil {
		log.Error().Err(err).Str("turnId", turnId).Msg("Failed to delete research report")
		writeError(w, http.StatusInternalServerError, "Failed to delete report", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
