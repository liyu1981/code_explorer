package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/llm"
)

type PEEPlanner interface {
	Plan(ctx context.Context, req PEEPlanRequest) (*DAG, error)
}

type PEEPlanRequest struct {
	Goal         string
	Iteration    int
	PriorOutputs map[string]any
	FailedTasks  []*Task
	MissingInfo  []string
}

type PEETaskPlan struct {
	ID        string         `json:"id"`
	Tool      string         `json:"tool"`
	Input     map[string]any `json:"input"`
	DependsOn []string       `json:"depends_on"`
}

type PEEPlanResponse struct {
	Tasks []PEETaskPlan `json:"tasks"`
}

const DefaultPEEPlannerSystemPrompt = `You are a task planner. Given a goal, plan a set of tasks to accomplish it.
Tasks may depend on each other. Output a JSON task graph.`

type PEELLMPlanner struct {
	llm            llm.LLM
	tools          []map[string]any
	responseFormat *llm.ResponseFormat
	systemPrompt   string
}

type PEELLMPlannerOption func(*PEELLMPlanner)

func PEEPlannerWithSystemPrompt(prompt string) PEELLMPlannerOption {
	return func(p *PEELLMPlanner) {
		p.systemPrompt = prompt
	}
}

func NewPEELLMPlanner(ai llm.LLM, tools []map[string]any, responseFormat *llm.ResponseFormat, opts ...PEELLMPlannerOption) *PEELLMPlanner {
	p := &PEELLMPlanner{
		llm:            ai,
		tools:          tools,
		responseFormat: responseFormat,
		systemPrompt:   DefaultPEEPlannerSystemPrompt,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func NewPEELLMPlannerWithJSONFormat(ai llm.LLM, tools []map[string]any) (*PEELLMPlanner, error) {
	responseFormat, err := llm.ResponseFormatFromStruct[PEEPlanResponse]("task_plan")
	if err != nil {
		return nil, fmt.Errorf("failed to create response format: %w", err)
	}
	return NewPEELLMPlanner(ai, tools, responseFormat), nil
}

func (p *PEELLMPlanner) Plan(ctx context.Context, req PEEPlanRequest) (*DAG, error) {
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

	var parsed PEEPlanResponse
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

func (p *PEELLMPlanner) buildSystemPrompt() string {
	return p.systemPrompt
}

func (p *PEELLMPlanner) buildUserPrompt(req PEEPlanRequest) string {
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
