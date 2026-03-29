package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/llm"
)

type Planner interface {
	Plan(ctx context.Context, req PlanRequest) (*DAG, error)
}

type PlanRequest struct {
	Goal         string
	Iteration    int
	PriorOutputs map[string]any
	FailedTasks  []*Task
	MissingInfo  []string
}

type TaskPlan struct {
	ID        string         `json:"id"`
	Tool      string         `json:"tool"`
	Input     map[string]any `json:"input"`
	DependsOn []string       `json:"depends_on"`
}

type PlanResponse struct {
	Tasks []TaskPlan `json:"tasks"`
}

const DefaultPlannerSystemPrompt = `You are a task planner. Given a goal, plan a set of tasks to accomplish it.
Tasks may depend on each other. Output a JSON task graph.`

type LLMPlanner struct {
	llm            llm.LLM
	tools          []map[string]any
	responseFormat *llm.ResponseFormat
	systemPrompt   string
}

type LLMPlannerOption func(*LLMPlanner)

func PlannerWithSystemPrompt(prompt string) LLMPlannerOption {
	return func(p *LLMPlanner) {
		p.systemPrompt = prompt
	}
}

func NewLLMPlanner(ai llm.LLM, tools []map[string]any, responseFormat *llm.ResponseFormat, opts ...LLMPlannerOption) *LLMPlanner {
	p := &LLMPlanner{
		llm:            ai,
		tools:          tools,
		responseFormat: responseFormat,
		systemPrompt:   DefaultPlannerSystemPrompt,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func NewLLMPlannerWithJSONFormat(ai llm.LLM, tools []map[string]any) (*LLMPlanner, error) {
	responseFormat, err := llm.ResponseFormatFromStruct[PlanResponse]("task_plan")
	if err != nil {
		return nil, fmt.Errorf("failed to create response format: %w", err)
	}
	return NewLLMPlanner(ai, tools, responseFormat), nil
}

func (p *LLMPlanner) Plan(ctx context.Context, req PlanRequest) (*DAG, error) {
	system := p.buildSystemPrompt()
	user := p.buildUserPrompt(req)

	messages := []llm.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}

	raw, _, err := p.llm.Generate(ctx, messages, p.tools, p.responseFormat)
	if err != nil {
		return nil, fmt.Errorf("planner llm: %w", err)
	}

	var parsed PlanResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("planner parse: %w, raw: %s", err, raw)
	}

	tasks := make([]*Task, len(parsed.Tasks))
	for i, pt := range parsed.Tasks {
		tasks[i] = &Task{
			ID:        pt.ID,
			Tool:      pt.Tool,
			Input:     pt.Input,
			DependsOn: pt.DependsOn,
			Status:    StatusPending,
		}
	}
	return NewDAG(tasks)
}

func (p *LLMPlanner) buildSystemPrompt() string {
	return p.systemPrompt
}

func (p *LLMPlanner) buildUserPrompt(req PlanRequest) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Goal: %s\n", req.Goal))
	sb.WriteString(fmt.Sprintf("Iteration: %d\n\n", req.Iteration))

	if len(req.PriorOutputs) > 0 {
		sb.WriteString("Prior outputs:\n")
		for id, output := range req.PriorOutputs {
			sb.WriteString(fmt.Sprintf("  - %s: %v\n", id, output))
		}
		sb.WriteString("\n")
	}

	if len(req.FailedTasks) > 0 {
		sb.WriteString("Failed tasks from previous attempt:\n")
		for _, t := range req.FailedTasks {
			sb.WriteString(fmt.Sprintf("  - %s: %v\n", t.ID, t.Err))
		}
		sb.WriteString("\n")
	}

	if len(req.MissingInfo) > 0 {
		sb.WriteString("Missing information to consider:\n")
		for _, info := range req.MissingInfo {
			sb.WriteString(fmt.Sprintf("  - %s\n", info))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
