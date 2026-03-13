package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/agent/tools"
	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/rs/zerolog/log"
)

type TaskManager interface {
	Submit(ctx context.Context, name string, payload any, maxRetries int) (string, error)
	GetTask(ctx context.Context, id string) (*db.Task, error)
}

type AgentFactoryInterface interface {
	BuildFromConfig(cfg *agent.Config) (*agent.Agent, error)
	GetSkillPrompt(ctx context.Context, name string) (string, error)
	RegisterTool(tool agent.Tool)
}

func HandleKnowledgeBuildTask(ctx context.Context, idx *codemogger.CodeIndex, task *db.Task, taskManager TaskManager, agentFactory AgentFactoryInterface, updateProgress func(progress int, message string)) error {
	var payload struct {
		CodebaseID string `json:"codebaseId"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	log.Info().Str("codebaseId", payload.CodebaseID).Msg("Starting real knowledge build task")
	updateProgress(0, "Starting knowledge build...")

	// Fetch codebase details
	cb, err := idx.GetStore().GetCodebaseByID(ctx, payload.CodebaseID)
	if err != nil {
		return fmt.Errorf("failed to fetch codebase: %w", err)
	}
	if cb == nil {
		return fmt.Errorf("codebase %s not found", payload.CodebaseID)
	}

	// 1. Get Skill
	skill, err := idx.GetStore().GetSkillByName(ctx, "architect-planner")
	if err != nil {
		return fmt.Errorf("failed to get skill architect-planner: %w", err)
	}
	if skill == nil {
		return fmt.Errorf("skill architect-planner not found")
	}

	systemPrompt := skill.SystemPrompt

	// 2. Register tools that need codebase root
	agentFactory.RegisterTool(tools.NewGetTreeTool(cb.RootPath))
	agentFactory.RegisterTool(tools.NewReadFileTool(cb.RootPath))
	agentFactory.RegisterTool(tools.NewGrepSearchTool(cb.RootPath))
	agentFactory.RegisterTool(tools.NewListAgentSkillsTool(idx.GetStore()))

	// 3. Build Agent
	ag, err := agentFactory.BuildFromConfig(&agent.Config{
		MaxIterations: 20,
	})
	if err != nil {
		return fmt.Errorf("failed to build orchestrator agent: %w", err)
	}

	ag.SetSystemPrompt(systemPrompt)

	// 4. Run Agent
	updateProgress(10, "Orchestrator starting analysis...")
	input := fmt.Sprintf("Analyze the codebase at %s and identify key modules and architecture.", cb.RootPath)
	_, err = ag.Run(ctx, input, task.ID, nil)
	if err != nil {
		return fmt.Errorf("orchestrator execution failed: %w", err)
	}

	updateProgress(100, "Knowledge build complete")
	return nil
}

func HandleKnowledgeWikiAnalyzeTask(ctx context.Context, idx *codemogger.CodeIndex, task *db.Task, agentFactory AgentFactoryInterface, updateProgress func(progress int, message string)) error {
	var payload struct {
		CodebaseID string `json:"codebaseId"`
		Path       string `json:"path"`
		SkillName  string `json:"skillName"`
		Goal       string `json:"goal"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	updateProgress(0, fmt.Sprintf("Analyzing module: %s...", payload.Path))

	// Fetch codebase details
	cb, err := idx.GetStore().GetCodebaseByID(ctx, payload.CodebaseID)
	if err != nil {
		return fmt.Errorf("failed to fetch codebase: %w", err)
	}
	if cb == nil {
		return fmt.Errorf("codebase %s not found", payload.CodebaseID)
	}

	// 1. Get Skill
	skillName := payload.SkillName
	if skillName == "" {
		skillName = "generic-analyst"
	}
	systemPrompt, err := agentFactory.GetSkillPrompt(ctx, skillName)
	if err != nil {
		return fmt.Errorf("failed to get skill %s: %w", skillName, err)
	}

	// 2. Register tools scoped to modules
	agentFactory.RegisterTool(tools.NewGetTreeTool(cb.RootPath))
	agentFactory.RegisterTool(tools.NewReadFileTool(cb.RootPath))
	agentFactory.RegisterTool(tools.NewGrepSearchTool(cb.RootPath))
	agentFactory.RegisterTool(tools.NewListAgentSkillsTool(idx.GetStore()))

	// 3. Build Agent
	ag, err := agentFactory.BuildFromConfig(&agent.Config{
		MaxIterations: 10,
	})
	if err != nil {
		return fmt.Errorf("failed to build analyze agent: %w", err)
	}

	ag.SetSystemPrompt(systemPrompt)

	// 4. Run Agent
	goal := payload.Goal
	if goal == "" {
		goal = fmt.Sprintf("Provide a detailed technical summary of the module at %s.", payload.Path)
	}

	input := fmt.Sprintf("Module Path: %s\nGoal: %s\n\nPlease explore the files in this path and provide a concise Markdown summary of its purpose, main components, and key logic.", payload.Path, goal)

	output, err := ag.Run(ctx, input, task.ID, nil)
	if err != nil {
		return fmt.Errorf("analyze agent execution failed: %w", err)
	}

	updateProgress(100, output) // Use output as final message
	return nil
}
