package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/rs/zerolog/log"
)

func (h *ApiHandler) handleAgentResearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "Query is required", nil)
		return
	}

	log.Info().Str("query", req.Query).Msg("Handling agent research request")

	// Load agent config from index, env or default
	var llmCfg map[string]any
	if h.index != nil && h.index.Config != nil && h.index.Config.LLM != nil {
		llmCfg = h.index.Config.LLM
	} else {
		llmCfg = map[string]any{
			"type":     "openai",
			"model":    os.Getenv("LLM_MODEL"),
			"endpoint": os.Getenv("LLM_ENDPOINT"),
		}
	}

	agentConfig := &agent.Config{
		LLM:           llmCfg,
		MaxIterations: 10,
	}

	if agentConfig.LLM["model"] == nil || agentConfig.LLM["model"] == "" {
		agentConfig.LLM["model"] = "gpt-4o"
	}
	if agentConfig.LLM["endpoint"] == nil || agentConfig.LLM["endpoint"] == "" {
		agentConfig.LLM["endpoint"] = "https://api.openai.com/v1/chat/completions"
	}
	if agentConfig.LLM["type"] == nil || agentConfig.LLM["type"] == "" {
		agentConfig.LLM["type"] = "openai"
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

	// Send initial steps
	// We can define standard steps here or let the agent emit them
	sw.SendStepUpdate("plan", protocol.StepActive)

	// Run agent in a goroutine or directly
	// For streaming, we should run it and let it write to sw
	_, err = ag.Run(r.Context(), req.Query, sw)
	if err != nil {
		// In a stream, we might have already started sending data.
		// Errors should ideally be sent as events.
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
	}

	sw.WriteDone()
}
