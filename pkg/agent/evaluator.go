package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/llm"
)

type EvalStatus string

const (
	EvalDone   EvalStatus = "done"
	EvalReplan EvalStatus = "replan"
	EvalFailed EvalStatus = "failed"
)

type EvalResult struct {
	Status      EvalStatus `json:"status"`
	FinalAnswer string     `json:"final_answer"`
	ReplanHint  string     `json:"replan_hint"`
	MissingInfo []string   `json:"missing_info"`
}

type Evaluator interface {
	Evaluate(ctx context.Context, goal string, d *DAG) (*EvalResult, error)
}

const DefaultEvaluatorSystemPrompt = `You are a result evaluator. Given a goal and task outcomes, decide:
- "done": goal fully achieved, provide final_answer
- "replan": goal not met, explain replan_hint and missing_info
- "failed": unrecoverable error`

type LLMEvaluator struct {
	llm            llm.LLM
	tools          []map[string]any
	responseFormat *llm.ResponseFormat
	systemPrompt   string
	lastHint       string
}

type LLMEvaluatorOption func(*LLMEvaluator)

func EvaluatorWithSystemPrompt(prompt string) LLMEvaluatorOption {
	return func(e *LLMEvaluator) {
		e.systemPrompt = prompt
	}
}

func NewLLMEvaluator(ai llm.LLM, tools []map[string]any, responseFormat *llm.ResponseFormat, opts ...LLMEvaluatorOption) *LLMEvaluator {
	e := &LLMEvaluator{
		llm:            ai,
		tools:          tools,
		responseFormat: responseFormat,
		systemPrompt:   DefaultEvaluatorSystemPrompt,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func NewLLMEvaluatorWithJSONFormat(ai llm.LLM, tools []map[string]any) (*LLMEvaluator, error) {
	responseFormat, err := llm.ResponseFormatFromStruct[EvalResult]("evaluation_result")
	if err != nil {
		return nil, fmt.Errorf("failed to create response format: %w", err)
	}
	return NewLLMEvaluator(ai, tools, responseFormat), nil
}

func (ev *LLMEvaluator) Evaluate(ctx context.Context, goal string, d *DAG) (*EvalResult, error) {
	system := ev.systemPrompt

	user := ev.buildPrompt(goal, d)

	messages := []llm.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}

	raw, _, err := ev.llm.Generate(ctx, messages, ev.tools, ev.responseFormat)
	if err != nil {
		return nil, fmt.Errorf("evaluator llm: %w", err)
	}

	var res EvalResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return nil, fmt.Errorf("evaluator parse: %w, raw: %s", err, raw)
	}

	if res.Status == EvalReplan && res.ReplanHint == ev.lastHint {
		res.Status = EvalFailed
		res.ReplanHint = "stagnation detected: " + res.ReplanHint
	}
	ev.lastHint = res.ReplanHint

	return &res, nil
}

func (ev *LLMEvaluator) buildPrompt(goal string, d *DAG) string {
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
