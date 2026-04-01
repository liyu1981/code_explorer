package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/protocol"
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

func PEEWithPlannerMaxIterations(n int) PEEWorkflowRunnerOption {
	return func(r *PEEWorkflowRunner) {
		r.plannerOpts = append(r.plannerOpts, PEEPlannerWithMaxIterations(n))
	}
}

func PEEWithEvaluatorSystemPrompt(prompt string) PEEWorkflowRunnerOption {
	return func(r *PEEWorkflowRunner) {
		r.evaluatorOpts = append(r.evaluatorOpts, PEEEvaluatorWithSystemPrompt(prompt))
	}
}

func PEEWithEvaluatorMaxIterations(n int) PEEWorkflowRunnerOption {
	return func(r *PEEWorkflowRunner) {
		r.evaluatorOpts = append(r.evaluatorOpts, PEEEvaluatorWithMaxIterations(n))
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

	r.planner = NewPEELLMPlanner(ai, toolRegistry, nil, r.plannerOpts...)
	r.executor = NewPEEExecutor(toolRegistry, maxWorkers)
	r.evaluator = NewPEELLMEvaluator(ai, toolRegistry, nil, r.evaluatorOpts...)
	return r
}

func NewPEEWorkflowRunnerWithJSONFormat(ai llm.LLM, toolRegistry *llm.ToolRegistry, maxWorkers, maxIter int, opts ...PEEWorkflowRunnerOption) (*PEEWorkflowRunner, error) {
	r := &PEEWorkflowRunner{
		maxIter:      maxIter,
		toolRegistry: toolRegistry,
	}
	for _, opt := range opts {
		opt(r)
	}

	tools := toolRegistry.MarshalToolsForLLM()

	planner, err := NewPEELLMPlannerWithJSONFormat(ai, toolRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create planner: %w", err)
	}

	evaluator, err := NewPEELLMEvaluatorWithJSONFormat(ai, toolRegistry, tools)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator: %w", err)
	}

	r.planner = planner
	r.executor = NewPEEExecutor(toolRegistry, maxWorkers)
	r.evaluator = evaluator
	return r, nil
}

func (r *PEEWorkflowRunner) Run(ctx context.Context, goal string, stream protocol.IStreamWriter) (string, error) {
	req := PEEPlanRequest{Goal: goal}

	if stream != nil {
		turnID := fmt.Sprintf("pee-%d", 0)
		stream.SendTurnStarted(turnID, goal, 0)
	}

	for i := range r.maxIter {
		req.Iteration = i + 1

		log.Info().Int("iteration", req.Iteration).Str("goal", goal).Msg("starting workflow iteration")

		if stream != nil {
			planningStepID := fmt.Sprintf("pee-planning-%d", i)
			stream.SendStepUpdate(planningStepID, fmt.Sprintf("Planning iteration %d", req.Iteration), protocol.StepActive)
		}

		dag, err := r.planner.Plan(ctx, req)
		if err != nil {
			if stream != nil {
				stream.SendStepUpdate(fmt.Sprintf("pee-planning-%d", i), fmt.Sprintf("Planning iteration %d", req.Iteration), protocol.StepFailed)
			}
			return "", fmt.Errorf("iter %d plan: %w", req.Iteration, err)
		}
		if stream != nil {
			stream.SendStepUpdate(fmt.Sprintf("pee-planning-%d", i), fmt.Sprintf("Planning iteration %d", req.Iteration), protocol.StepCompleted)
		}

		if stream != nil {
			stream.SendStepUpdate(fmt.Sprintf("pee-execution-%d", i), "Executing tasks", protocol.StepActive)
		}

		if err := r.executeWithStream(ctx, dag, stream); err != nil {
			if stream != nil {
				stream.SendStepUpdate(fmt.Sprintf("pee-execution-%d", i), "Executing tasks", protocol.StepFailed)
			}
			return "", fmt.Errorf("iter %d execute: %w", req.Iteration, err)
		}
		if stream != nil {
			stream.SendStepUpdate(fmt.Sprintf("pee-execution-%d", i), "Executing tasks", protocol.StepCompleted)
		}

		if stream != nil {
			stream.SendStepUpdate(fmt.Sprintf("pee-evaluating-%d", i), "Evaluating results", protocol.StepActive)
		}

		result, err := r.evaluator.Evaluate(ctx, goal, dag)
		if err != nil {
			if stream != nil {
				stream.SendStepUpdate(fmt.Sprintf("pee-evaluating-%d", i), "Evaluating results", protocol.StepFailed)
			}
			return "", fmt.Errorf("iter %d evaluate: %w", req.Iteration, err)
		}
		if stream != nil {
			stream.SendStepUpdate(fmt.Sprintf("pee-evaluating-%d", i), "Evaluating results", protocol.StepCompleted)
		}

		switch result.Status {
		case PEEEvalDone:
			log.Info().Int("iteration", req.Iteration).Msg("workflow completed successfully")
			if stream != nil {
				stream.WriteOpenAIChunk("", "", result.FinalAnswer, nil)
				stream.WriteDone()
			}
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

func (r *PEEWorkflowRunner) executeWithStream(ctx context.Context, dag *DAG, stream protocol.IStreamWriter) error {
	for _, t := range dag.tasks {
		if stream != nil {
			toolStepID := fmt.Sprintf("pee-tool-%s", t.ID)
			stream.SendStepUpdate(toolStepID, fmt.Sprintf("Executing %s", t.Tool), protocol.StepActive)
			stream.SendToolCall(t.Tool, t.Input)
		}

		tool, ok := r.toolRegistry.Get(t.Tool)
		if !ok {
			err := fmt.Errorf("unknown tool: %s", t.Tool)
			dag.SetResult(t.ID, nil, err)
			dag.SetStatus(t.ID, StatusFailed)
			if stream != nil {
				stream.SendToolResponse(t.Tool, err.Error())
				stream.SendStepUpdate(fmt.Sprintf("pee-tool-%s", t.ID), fmt.Sprintf("Executing %s", t.Tool), protocol.StepFailed)
			}
			continue
		}

		inputJSON, err := json.Marshal(t.Input)
		if err != nil {
			err = fmt.Errorf("marshal input: %w", err)
			dag.SetResult(t.ID, nil, err)
			dag.SetStatus(t.ID, StatusFailed)
			if stream != nil {
				stream.SendToolResponse(t.Tool, err.Error())
				stream.SendStepUpdate(fmt.Sprintf("pee-tool-%s", t.ID), fmt.Sprintf("Executing %s", t.Tool), protocol.StepFailed)
			}
			continue
		}

		output, err := tool.Execute(ctx, inputJSON, stream)
		dag.SetResult(t.ID, output, err)
		if err != nil {
			log.Error().Err(err).Str("task", t.ID).Msg("tool execution failed")
			dag.SetStatus(t.ID, StatusFailed)
			if stream != nil {
				stream.SendToolResponse(t.Tool, err.Error())
				stream.SendStepUpdate(fmt.Sprintf("pee-tool-%s", t.ID), fmt.Sprintf("Executing %s", t.Tool), protocol.StepFailed)
			}
		} else {
			log.Info().Str("task", t.ID).Msg("task completed")
			dag.SetStatus(t.ID, StatusDone)
			if stream != nil {
				var structured any
				if err := json.Unmarshal([]byte(output), &structured); err == nil {
					stream.SendToolResponse(t.Tool, structured)
				} else {
					stream.SendToolResponse(t.Tool, output)
				}
				stream.SendStepUpdate(fmt.Sprintf("pee-tool-%s", t.ID), fmt.Sprintf("Executing %s", t.Tool), protocol.StepCompleted)
			}
		}
	}
	return nil
}
