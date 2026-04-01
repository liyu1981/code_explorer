package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/tools"
)

type PEEEvalStatus string

const (
	PEEEvalDone   PEEEvalStatus = "done"
	PEEEvalReplan PEEEvalStatus = "replan"
	PEEEvalFailed PEEEvalStatus = "failed"
)

type PEEEvalResult struct {
	Status      PEEEvalStatus `json:"status"`
	FinalAnswer string        `json:"final_answer"`
	ReplanHint  string        `json:"replan_hint"`
	MissingInfo []string      `json:"missing_info"`
}

type PEEEvaluator interface {
	Evaluate(ctx context.Context, goal string, d *DAG) (*PEEEvalResult, error)
}

const DefaultPEEEvaluatorSystemPrompt = `You are a result evaluator. Given a goal and task outcomes, decide:
- "done": goal fully achieved, provide final_answer
- "replan": goal not met, explain replan_hint and missing_info
- "failed": unrecoverable error`

type PEELLMEvaluator struct {
	generator      *llm.Generator
	toolRegistry   *tools.ToolRegistry
	tools          []map[string]any
	responseFormat *llm.ResponseFormat
	systemPrompt   string
	lastHint       string
}

type PEELLMEvaluatorOption func(*PEELLMEvaluator)

func PEEEvaluatorWithSystemPrompt(prompt string) PEELLMEvaluatorOption {
	return func(e *PEELLMEvaluator) {
		e.systemPrompt = prompt
	}
}

func PEEEvaluatorWithMaxIterations(n int) PEELLMEvaluatorOption {
	return func(e *PEELLMEvaluator) {
		e.generator.Options(llm.WithGeneratorMaxIterations(n))
	}
}

func NewPEELLMEvaluator(ai llm.LLM, toolRegistry *tools.ToolRegistry, responseFormat *llm.ResponseFormat, opts ...PEELLMEvaluatorOption) *PEELLMEvaluator {
	tools := toolRegistry.MarshalToolsForLLM()
	e := &PEELLMEvaluator{
		generator:      llm.NewGenerator(ai, llm.WithGeneratorToolRegistry(toolRegistry)),
		toolRegistry:   toolRegistry,
		tools:          tools,
		responseFormat: responseFormat,
		systemPrompt:   DefaultPEEEvaluatorSystemPrompt,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func NewPEELLMEvaluatorWithJSONFormat(ai llm.LLM, toolRegistry *tools.ToolRegistry, toolsList []map[string]any) (*PEELLMEvaluator, error) {
	responseFormat, err := llm.ResponseFormatFromStruct[PEEEvalResult]("evaluation_result")
	if err != nil {
		return nil, fmt.Errorf("failed to create response format: %w", err)
	}
	return NewPEELLMEvaluator(ai, toolRegistry, responseFormat), nil
}

func (ev *PEELLMEvaluator) Evaluate(ctx context.Context, goal string, d *DAG) (*PEEEvalResult, error) {
	system := ev.systemPrompt

	user := ev.buildPrompt(goal, d)

	messages := []llm.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}

	raw, _, err := ev.generator.Generate(ctx, messages, ev.tools, ev.responseFormat)
	if err != nil {
		return nil, fmt.Errorf("evaluator llm: %w", err)
	}

	var res PEEEvalResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return nil, fmt.Errorf("evaluator parse: %w, raw: %s", err, raw)
	}

	if res.Status == PEEEvalReplan && res.ReplanHint == ev.lastHint {
		res.Status = PEEEvalFailed
		res.ReplanHint = "stagnation detected: " + res.ReplanHint
	}
	ev.lastHint = res.ReplanHint

	return &res, nil
}

func (ev *PEELLMEvaluator) buildPrompt(goal string, d *DAG) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Goal: %s\n\n", goal))
	sb.WriteString("Task outcomes:\n\n")

	for id, t := range d.tasks {
		sb.WriteString(fmt.Sprintf("Task: %s\n", id))
		sb.WriteString(fmt.Sprintf("  Status: %s\n", t.Status))
		sb.WriteString(fmt.Sprintf("  Tool: %s\n", t.Tool))
		if t.Description != "" {
			sb.WriteString(fmt.Sprintf("  Description: %s\n", t.Description))
		}
		if t.Output != nil {
			sb.WriteString(fmt.Sprintf("  Output: %v\n", t.Output))
		}
		if t.Err != nil {
			sb.WriteString(fmt.Sprintf("  Error: %v\n", t.Err))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
