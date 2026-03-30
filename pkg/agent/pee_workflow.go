package workflow

import (
	"context"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/rs/zerolog/log"
)

type PEEWorkflowRunner struct {
	planner       PEEPlanner
	executor      *PEEExecutor
	evaluator     PEEEvaluator
	maxIter       int
	toolRegistry  *llm.ToolRegistry
	plannerOpts   []PEELLMPlannerOption
	evaluatorOpts []PEELLMEvaluatorOption
}

type PEEWorkflowRunnerOption func(*PEEWorkflowRunner)

func PEEWithPlannerSystemPrompt(prompt string) PEEWorkflowRunnerOption {
	return func(r *PEEWorkflowRunner) {
		r.plannerOpts = append(r.plannerOpts, PEEPlannerWithSystemPrompt(prompt))
	}
}

func PEEWithEvaluatorSystemPrompt(prompt string) PEEWorkflowRunnerOption {
	return func(r *PEEWorkflowRunner) {
		r.evaluatorOpts = append(r.evaluatorOpts, PEEEvaluatorWithSystemPrompt(prompt))
	}
}

func NewPEEWorkflowRunner(ai llm.LLM, toolRegistry *llm.ToolRegistry, maxWorkers, maxIter int, opts ...PEEWorkflowRunnerOption) *PEEWorkflowRunner {
	r := &PEEWorkflowRunner{
		maxIter:      maxIter,
		toolRegistry: toolRegistry,
	}
	for _, opt := range opts {
		opt(r)
	}

	tools := toolRegistry.MarshalToolsForLLM()
	return &PEEWorkflowRunner{
		planner:      NewPEELLMPlanner(ai, tools, nil, r.plannerOpts...),
		executor:     NewPEEExecutor(toolRegistry, maxWorkers),
		evaluator:    NewPEELLMEvaluator(ai, tools, nil, r.evaluatorOpts...),
		maxIter:      maxIter,
		toolRegistry: toolRegistry,
	}
}

func NewRunnerWithJSONFormat(ai llm.LLM, toolRegistry *llm.ToolRegistry, maxWorkers, maxIter int, opts ...PEEWorkflowRunnerOption) (*PEEWorkflowRunner, error) {
	r := &PEEWorkflowRunner{
		maxIter:      maxIter,
		toolRegistry: toolRegistry,
	}
	for _, opt := range opts {
		opt(r)
	}

	tools := toolRegistry.MarshalToolsForLLM()

	planner, err := NewPEELLMPlannerWithJSONFormat(ai, tools)
	if err != nil {
		return nil, fmt.Errorf("failed to create planner: %w", err)
	}

	evaluator, err := NewPEELLMEvaluatorWithJSONFormat(ai, tools)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator: %w", err)
	}

	return &PEEWorkflowRunner{
		planner:      planner,
		executor:     NewPEEExecutor(toolRegistry, maxWorkers),
		evaluator:    evaluator,
		maxIter:      maxIter,
		toolRegistry: toolRegistry,
	}, nil
}

func (r *PEEWorkflowRunner) Run(ctx context.Context, goal string) (string, error) {
	req := PEEPlanRequest{Goal: goal}

	for i := range r.maxIter {
		req.Iteration = i + 1

		log.Info().Int("iteration", req.Iteration).Str("goal", goal).Msg("starting workflow iteration")

		dag, err := r.planner.Plan(ctx, req)
		if err != nil {
			return "", fmt.Errorf("iter %d plan: %w", req.Iteration, err)
		}

		if err := r.executor.Execute(ctx, dag); err != nil {
			return "", fmt.Errorf("iter %d execute: %w", req.Iteration, err)
		}

		result, err := r.evaluator.Evaluate(ctx, goal, dag)
		if err != nil {
			return "", fmt.Errorf("iter %d evaluate: %w", req.Iteration, err)
		}

		switch result.Status {
		case PEEEvalDone:
			log.Info().Int("iteration", req.Iteration).Msg("workflow completed successfully")
			return result.FinalAnswer, nil
		case PEEEvalFailed:
			log.Error().Str("hint", result.ReplanHint).Int("iteration", req.Iteration).Msg("workflow failed")
			return "", fmt.Errorf("unrecoverable: %s", result.ReplanHint)
		case PEEEvalReplan:
			log.Info().Str("hint", result.ReplanHint).Int("iteration", req.Iteration).Msg("replanning")
			req.PriorOutputs = dag.Outputs()
			req.FailedTasks = dag.FailedTasks()
			req.MissingInfo = result.MissingInfo
		}
	}

	return "", fmt.Errorf("exceeded max iterations (%d)", r.maxIter)
}
