package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/rs/zerolog/log"
)

type Intent string

const (
	IntentSimple      Intent = "simple"
	IntentInvestigate Intent = "investigate"
	IntentComplex     Intent = "complex"
)

type RouteResult struct {
	Intent            Intent
	Confidence        float64
	SuggestedWorkflow string
	Reasoning         string
}

type IntentDetection struct {
	Intent     Intent  `json:"intent"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

const DefaultDynamicSystemPrompt = `Analyze the user request and determine the appropriate workflow intent.

Categories:
- "simple": Factual questions, definitions, direct answers (e.g., "what is X", "who is Y", "define Z")
- "investigate": Code/file exploration, search, reading (e.g., "find files", "search for X", "read file Y", "grep Z")
- "complex": Multi-step tasks, reports, comparisons (e.g., "summarize X", "compare Y and Z", "generate report", "analyze and explain")

Respond with JSON only.`

type DynamicRouter struct {
	llm          llm.LLM
	toolRegistry *llm.ToolRegistry
	systemPrompt string
	reactRunner  *ReactWorkflowRunner
	rcRunner     *RCWorkflowRunner
	peeRunner    *PEEWorkflowRunner
}

type DynamicRouterOption func(*DynamicRouter)

func DynamicWithSystemPrompt(prompt string) DynamicRouterOption {
	return func(d *DynamicRouter) {
		d.systemPrompt = prompt
	}
}

func DynamicWithReactWorkflowRunner(runner *ReactWorkflowRunner) DynamicRouterOption {
	return func(d *DynamicRouter) {
		d.reactRunner = runner
	}
}

func DynamicWithRCWorkflowRunner(runner *RCWorkflowRunner) DynamicRouterOption {
	return func(d *DynamicRouter) {
		d.rcRunner = runner
	}
}

func DynamicWithPEERunner(runner *PEEWorkflowRunner) DynamicRouterOption {
	return func(d *DynamicRouter) {
		d.peeRunner = runner
	}
}

func NewDynamicRouter(ai llm.LLM, toolRegistry *llm.ToolRegistry, opts ...DynamicRouterOption) *DynamicRouter {
	d := &DynamicRouter{
		llm:          ai,
		toolRegistry: toolRegistry,
		systemPrompt: DefaultDynamicSystemPrompt,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

const MinConfidenceThreshold = 0.7

func (d *DynamicRouter) Route(ctx context.Context, goal string) (*RouteResult, error) {
	intentFormat, err := llm.ResponseFormatFromStruct[IntentDetection]("intent_detection")
	if err != nil {
		return nil, fmt.Errorf("failed to create intent format: %w", err)
	}

	messages := []llm.Message{
		{Role: "system", Content: d.systemPrompt},
		{Role: "user", Content: goal},
	}

	response, _, err := d.llm.Generate(ctx, messages, nil, intentFormat)
	if err != nil {
		return nil, fmt.Errorf("intent detection llm: %w", err)
	}

	var detected IntentDetection
	if err := json.Unmarshal([]byte(response), &detected); err != nil {
		log.Warn().Err(err).Str("response", response).Msg("failed to parse intent, defaulting to investigate")
		return &RouteResult{
			Intent:            IntentInvestigate,
			Confidence:        0.5,
			SuggestedWorkflow: "react",
			Reasoning:         "Failed to parse intent detection response",
		}, nil
	}

	workflow := d.chooseWorkflow(detected.Intent)

	return &RouteResult{
		Intent:            detected.Intent,
		Confidence:        detected.Confidence,
		SuggestedWorkflow: workflow,
		Reasoning:         detected.Reasoning,
	}, nil
}

func (d *DynamicRouter) chooseWorkflow(intent Intent) string {
	switch intent {
	case IntentSimple:
		return "simple"
	case IntentInvestigate:
		if d.rcRunner != nil {
			return "reflect-critic"
		}
		if d.reactRunner != nil {
			return "react"
		}
		return "react"
	case IntentComplex:
		if d.peeRunner != nil {
			return "pee"
		}
		if d.rcRunner != nil {
			return "reflect-critic"
		}
		return "reflect-critic"
	default:
		return "react"
	}
}

func (d *DynamicRouter) Run(ctx context.Context, goal string) (string, error) {
	route, err := d.Route(ctx, goal)
	if err != nil {
		return "", fmt.Errorf("routing failed: %w", err)
	}

	log.Info().
		Str("intent", string(route.Intent)).
		Float64("confidence", route.Confidence).
		Str("workflow", route.SuggestedWorkflow).
		Str("reasoning", route.Reasoning).
		Msg("Routing decision")

	if route.Confidence < MinConfidenceThreshold && d.peeRunner != nil {
		log.Info().Float64("confidence", route.Confidence).Msg("Low confidence, falling back to PEE")
		return d.peeRunner.Run(ctx, goal)
	}

	switch route.SuggestedWorkflow {
	case "simple":
		return d.directLLM(ctx, goal)
	case "react":
		if d.reactRunner != nil {
			return d.reactRunner.Run(ctx, goal)
		}
		if d.rcRunner != nil {
			return d.rcRunner.Run(ctx, goal)
		}
		return d.directLLM(ctx, goal)
	case "reflect-critic":
		if d.rcRunner != nil {
			return d.rcRunner.Run(ctx, goal)
		}
		if d.reactRunner != nil {
			return d.reactRunner.Run(ctx, goal)
		}
		return d.directLLM(ctx, goal)
	case "pee":
		if d.peeRunner != nil {
			return d.peeRunner.Run(ctx, goal)
		}
		if d.rcRunner != nil {
			return d.rcRunner.Run(ctx, goal)
		}
		return d.directLLM(ctx, goal)
	default:
		return d.directLLM(ctx, goal)
	}
}

func (d *DynamicRouter) directLLM(ctx context.Context, goal string) (string, error) {
	messages := []llm.Message{
		{Role: "user", Content: goal},
	}

	response, _, err := d.llm.Generate(ctx, messages, nil, nil)
	if err != nil {
		return "", fmt.Errorf("direct llm: %w", err)
	}
	return response, nil
}
