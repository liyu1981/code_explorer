package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

func (h *ApiHandler) handleMockGenerate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query     string `json:"query"`
		SessionID string `json:"sessionId"`
		TurnID    string `json:"turnId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "Query is required", nil)
		return
	}

	if req.TurnID == "" {
		req.TurnID = fmt.Sprintf("mock-turn-%d", time.Now().UnixMilli())
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sw := protocol.NewStreamWriter(req.TurnID, w)
	var finalSw protocol.IStreamWriter = sw

	if req.SessionID != "" {
		finalSw = &persistenceStreamWriter{
			StreamWriter: sw,
			sessionID:    req.SessionID,
			turnID:       req.TurnID,
			store:        db.GetStore(),
		}
	}

	_ = finalSw.SendTurnStarted(req.Query, time.Now().UnixMilli())

	steps := []struct {
		id    string
		label string
	}{
		{"mock-step-1", "Searching codebase"},
		{"mock-step-2", "Analyzing results"},
		{"mock-step-3", "Reading source files"},
		{"mock-step-4", "Synthesizing answer"},
	}

	for _, step := range steps {
		_ = finalSw.SendStepUpdate(step.id, step.label, protocol.StepActive)
		time.Sleep(500 * time.Millisecond)
		_ = finalSw.SendStepUpdate(step.id, step.label, protocol.StepCompleted)
	}

	reasoningLines := []string{
		"Let me analyze the codebase structure...\n",
		"I found several relevant files that match the query.\n",
		"The key files are in the `pkg/server` directory.\n",
		"Based on my analysis, here is what I found.\n",
	}

	for _, line := range reasoningLines {
		_ = finalSw.SendReasoning(line)
		time.Sleep(200 * time.Millisecond)
	}

	sources := []protocol.SourceMaterial{
		{
			ID:        "src-1",
			Path:      "pkg/server/api/api_research.go",
			Snippet:   "func (h *ApiHandler) handleAgentResearch(w http.ResponseWriter, r *http.Request) {",
			StartLine: 115,
			EndLine:   189,
		},
		{
			ID:        "src-2",
			Path:      "pkg/protocol/stream_writer.go",
			Snippet:   "type IStreamWriter interface {",
			StartLine: 11,
			EndLine:   26,
		},
	}

	for _, src := range sources {
		_ = finalSw.SendSourceAdded(src)
		time.Sleep(100 * time.Millisecond)
	}

	report := "## Analysis\n\n" +
		"Based on the codebase, here is what I found:\n\n" +
		"- The research endpoint is defined in `pkg/server/api/api_research.go`\n" +
		"- The streaming protocol is implemented in `pkg/protocol/stream_writer.go`\n" +
		"- Events are sent as Server-Sent Events (SSE) with `ce:` prefix for custom events\n\n" +
		"### Key Components\n\n" +
		"1. **StreamWriter** - handles writing SSE chunks to the response\n" +
		"2. **CEEvent** - custom event type for research-specific messages\n" +
		"3. **IStreamWriter** - interface for writing to the stream\n"

	_ = finalSw.WriteOpenAIChunk("mock-1", "mock-model", report, nil)
	time.Sleep(100 * time.Millisecond)
	_ = finalSw.WriteDone()
}
