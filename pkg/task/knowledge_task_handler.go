package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/rs/zerolog/log"
)

type TaskManager interface {
	Submit(ctx context.Context, name string, payload any, maxRetries int) (string, error)
	GetTask(ctx context.Context, id string) (*db.Task, error)
}

func HandleKnowledgeWikiPlanTask(ctx context.Context, idx *codemogger.CodeIndex, task *db.Task, taskManager TaskManager, agentFactory agent.AgentFactoryInterface, updateProgress func(progress int, message string)) error {
	var payload struct {
		CodebaseID string `json:"codebaseId"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Info().Str("codebaseId", payload.CodebaseID).Msg("Starting knowledge build plan task")
	updateProgress(0, "Starting knowledge build planning...")

	// Fetch codebase details
	cb, err := idx.GetStore().GetCodebaseByID(ctx, payload.CodebaseID)
	if err != nil {
		return fmt.Errorf("failed to fetch codebase: %w", err)
	}
	if cb == nil {
		return fmt.Errorf("codebase %s not found", payload.CodebaseID)
	}

	// 1. Get Skill
	skill, err := idx.GetStore().GetSkillByName(ctx, "knowledge-base-planner")
	if err != nil {
		return fmt.Errorf("failed to get skill knowledge-base-planner: %w", err)
	}
	if skill == nil {
		return fmt.Errorf("skill knowledge-base-planner not found")
	}

	systemPrompt := skill.SystemPrompt

	// 3. Build Agent
	ag, err := agentFactory.BuildFromConfig(ctx, &agent.Config{
		MaxIterations: 20,
		SkillName:     "knowledge-base-planner",
	})
	if err != nil {
		return fmt.Errorf("failed to build orchestrator agent: %w", err)
	}

	ag.SetSystemPrompt(systemPrompt)

	// 4. Run Agent
	updateProgress(10, "Planner starting analysis...")
	input := fmt.Sprintf("CodebaseID: %s\n\nAnalyze the codebase at %s and generate wiki building tasks", payload.CodebaseID, cb.RootPath)
	_, err = ag.RunLoop(ctx, input, task.ID, nil)
	if err != nil {
		return fmt.Errorf("orchestrator execution failed: %w", err)
	}

	updateProgress(100, "Knowledge build complete")
	return nil
}

func HandleKnowledgeWikiBuildTask(ctx context.Context, idx *codemogger.CodeIndex, task *db.Task, agentFactory agent.AgentFactoryInterface, updateProgress func(progress int, message string)) error {
	var payload struct {
		CodebaseID string `json:"codebaseId"`
		Topic      string `json:"topic"`
		Goal       string `json:"goal"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	updateProgress(0, fmt.Sprintf("Analyzing codebase: %s...", payload.CodebaseID))

	// Fetch codebase details
	cb, err := idx.GetStore().GetCodebaseByID(ctx, payload.CodebaseID)
	if err != nil {
		return fmt.Errorf("failed to fetch codebase: %w", err)
	}
	if cb == nil {
		return fmt.Errorf("codebase %s not found", payload.CodebaseID)
	}

	// 1. Get Skill
	systemPrompt, err := agentFactory.GetSkillPrompt(ctx, "knowledge-base-builder")
	if err != nil {
		return fmt.Errorf("failed to get skill knowledge-base-builder: %w", err)
	}

	// 3. Build Agent
	ag, err := agentFactory.BuildFromConfig(
		ctx,
		&agent.Config{
			MaxIterations: 20,
			SkillName:     "knowledge-base-builder",
		},
		agent.WithBindData("baseDir", cb.RootPath),
	)
	if err != nil {
		return fmt.Errorf("failed to build analyze agent: %w", err)
	}

	ag.SetSystemPrompt(systemPrompt)

	// 4. Run Agent
	input := fmt.Sprintf("Codebase ID: %s\n\nBuild the wiki page with topic: \"%s\" in markdown with goal: %s\n", payload.CodebaseID, payload.Topic, payload.Goal)

	output, err := ag.RunLoop(ctx, input, task.ID, nil)
	if err != nil {
		return fmt.Errorf("wiki build agent execution failed: %w", err)
	}

	updateProgress(100, output) // Use output as final message
	return nil
}
