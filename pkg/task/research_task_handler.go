package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

func HandleSummarizeTopicTask(
	ctx context.Context,
	idx *codemogger.CodeIndex,
	task *db.Task,
	updateProgress func(progress int, message string),
	notifyUpdated func(sessionId string, title string),
) error {
	var payload struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	updateProgress(10, "Fetching research reports...")

	// Get reports to find the first question and part of the report
	reports, err := idx.GetStore().GetResearchReportsBySession(ctx, payload.SessionID)
	if err != nil {
		return fmt.Errorf("failed to get reports: %w", err)
	}
	if len(reports) == 0 {
		return fmt.Errorf("no reports found for session %s", payload.SessionID)
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
				} `choices`
			}
			if err := json.Unmarshal([]byte(line[6:]), &chunk); err == nil && len(chunk.Choices) > 0 {
				firstReport += chunk.Choices[0].Delta.Content
			}
		}
	}

	updateProgress(40, "Generating summary...")

	// Build Agent using the skill
	ag, err := agent.NewAgentFromConfig(ctx, &agent.AgentConfig{
		MaxIterations:   1,
		AgentPromptName: "concise-topic-summarizer",
	})
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	userInput := fmt.Sprintf("Generate a concise title (strictly maximum 5 words) for this research.\n\nQuery: %s\n\nPartial Report: %s", firstQuery, firstReport)

	title, err := ag.Run(ctx, userInput, nil, nil)

	if err != nil {
		return fmt.Errorf("failed to generate title: %w", err)
	}

	title = strings.Trim(title, "\" \n\r")
	if title == "" {
		return fmt.Errorf("agent returned empty title")
	}

	updateProgress(80, "Updating session title...")

	// Update session title
	sess, err := idx.GetStore().GetResearchSession(ctx, payload.SessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if sess == nil {
		return fmt.Errorf("session %s not found", payload.SessionID)
	}

	sess.Title = title
	if err := idx.GetStore().SaveResearchSession(ctx, sess); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	if notifyUpdated != nil {
		notifyUpdated(payload.SessionID, title)
	}

	updateProgress(100, fmt.Sprintf("Session summarized: %s", title))
	return nil
}
