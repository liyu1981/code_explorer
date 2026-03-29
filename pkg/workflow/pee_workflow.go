package workflow

import (
	"context"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/rs/zerolog/log"
)

type PEEWorkflowRunner struct {
	planner      Planner
	executor     *Executor
	evaluator    Evaluator
	maxIter      int
	toolRegistry *agent.ToolRegistry
}

func NewPEEWorkflowRunner(llm agent.LLM, toolRegistry *agent.ToolRegistry, maxWorkers, maxIter int) *PEEWorkflowRunner {
	tools := toolRegistry.MarshalToolsForLLM()
	return &PEEWorkflowRunner{
		planner:      NewLLMPlanner(llm, tools, nil),
		executor:     NewExecutor(toolRegistry, maxWorkers),
		evaluator:    NewLLMEvaluator(llm, tools, nil),
		maxIter:      maxIter,
		toolRegistry: toolRegistry,
	}
}

func NewRunnerWithJSONFormat(llm agent.LLM, toolRegistry *agent.ToolRegistry, maxWorkers, maxIter int) (*PEEWorkflowRunner, error) {
	tools := toolRegistry.MarshalToolsForLLM()

	planner, err := NewLLMPlannerWithJSONFormat(llm, tools)
	if err != nil {
		return nil, fmt.Errorf("failed to create planner: %w", err)
	}

	evaluator, err := NewLLMEvaluatorWithJSONFormat(llm, tools)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator: %w", err)
	}

	return &PEEWorkflowRunner{
		planner:      planner,
		executor:     NewExecutor(toolRegistry, maxWorkers),
		evaluator:    evaluator,
		maxIter:      maxIter,
		toolRegistry: toolRegistry,
	}, nil
}

func (r *PEEWorkflowRunner) Run(ctx context.Context, goal string) (string, error) {
	req := PlanRequest{Goal: goal}

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
		case EvalDone:
			log.Info().Int("iteration", req.Iteration).Msg("workflow completed successfully")
			return result.FinalAnswer, nil
		case EvalFailed:
			log.Error().Str("hint", result.ReplanHint).Int("iteration", req.Iteration).Msg("workflow failed")
			return "", fmt.Errorf("unrecoverable: %s", result.ReplanHint)
		case EvalReplan:
			log.Info().Str("hint", result.ReplanHint).Int("iteration", req.Iteration).Msg("replanning")
			req.PriorOutputs = dag.Outputs()
			req.FailedTasks = dag.FailedTasks()
			req.MissingInfo = result.MissingInfo
		}
	}

	return "", fmt.Errorf("exceeded max iterations (%d)", r.maxIter)
}
