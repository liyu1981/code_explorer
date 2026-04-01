//go:build integration

package agent

import (
	"context"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/llm"
)

func TestPEELLMEvaluatorIntegration(t *testing.T) {
	stype, baseURL, model, apiKey, noThink := llm.GetIntegrationTestParams()

	llmCfg := map[string]any{
		"type":     stype,
		"model":    model,
		"base_url": baseURL,
		"api_key":  apiKey,
		"no_think": noThink,
	}
	llmInstance, err := llm.BuildLLM(llmCfg)
	if err != nil {
		t.Fatalf("Failed to build LLM: %v", err)
	}

	evaluator, err := NewPEELLMEvaluatorWithJSONFormat(llmInstance, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	t.Run("Evaluate Completed Goal", func(t *testing.T) {
		tasks := []*Task{
			{
				ID:     "search_task",
				Tool:   "grep_search",
				Status: StatusDone,
				Output: "Found 10 matches for 'authentication'",
			},
			{
				ID:        "read_task",
				Tool:      "read_file",
				DependsOn: []string{"search_task"},
				Status:    StatusDone,
				Output:    "File contents: user authentication logic implemented",
			},
		}

		dag, err := NewDAG(tasks)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		goal := "Find and read authentication-related files"

		result, err := evaluator.Evaluate(ctx, goal, dag)
		if err != nil {
			t.Fatalf("Evaluator failed: %v", err)
		}

		t.Logf("Evaluation status: %s", result.Status)
		t.Logf("Final answer: %s", result.FinalAnswer)
		if result.ReplanHint != "" {
			t.Logf("Replan hint: %s", result.ReplanHint)
		}
		if len(result.MissingInfo) > 0 {
			t.Logf("Missing info: %v", result.MissingInfo)
		}

		if result.Status != PEEEvalDone && result.Status != PEEEvalReplan {
			t.Errorf("Expected Done or Replan, got %s", result.Status)
		}
	})

	t.Run("Evaluate Partial Completion", func(t *testing.T) {
		tasks := []*Task{
			{
				ID:     "task_1",
				Tool:   "echo",
				Status: StatusDone,
				Output: "First step completed",
			},
			{
				ID:     "task_2",
				Tool:   "grep_search",
				Status: StatusFailed,
				Err:    nil,
			},
		}

		dag, err := NewDAG(tasks)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		goal := "Complete a multi-step search and echo process"

		result, err := evaluator.Evaluate(ctx, goal, dag)
		if err != nil {
			t.Fatalf("Evaluator failed: %v", err)
		}

		t.Logf("Evaluation status: %s", result.Status)
		if result.ReplanHint != "" {
			t.Logf("Replan hint: %s", result.ReplanHint)
		}
		if len(result.MissingInfo) > 0 {
			t.Logf("Missing info: %v", result.MissingInfo)
		}
	})

	t.Run("Evaluate All Tasks Failed", func(t *testing.T) {
		tasks := []*Task{
			{
				ID:     "failing_task_1",
				Tool:   "nonexistent",
				Status: StatusFailed,
				Err:    nil,
			},
			{
				ID:        "failing_task_2",
				Tool:      "unknown",
				Status:    StatusSkipped,
				DependsOn: []string{"failing_task_1"},
			},
		}

		dag, err := NewDAG(tasks)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		goal := "Try to execute some tasks"

		result, err := evaluator.Evaluate(ctx, goal, dag)
		if err != nil {
			t.Fatalf("Evaluator failed: %v", err)
		}

		t.Logf("Evaluation status: %s", result.Status)
		if result.Status == PEEEvalFailed {
			t.Logf("Detected as unrecoverable: %s", result.ReplanHint)
		} else if result.ReplanHint != "" {
			t.Logf("Replan hint: %s", result.ReplanHint)
		}
	})

	t.Run("Evaluate Single Task Success", func(t *testing.T) {
		tasks := []*Task{
			{
				ID:          "simple_echo",
				Tool:        "echo",
				Status:      StatusDone,
				Description: "Echo a simple message",
				Output:      "echo: hello world",
			},
		}

		dag, err := NewDAG(tasks)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		goal := "Echo 'hello world'"

		result, err := evaluator.Evaluate(ctx, goal, dag)
		if err != nil {
			t.Fatalf("Evaluator failed: %v", err)
		}

		t.Logf("Evaluation status: %s", result.Status)
		t.Logf("Final answer: %s", result.FinalAnswer)
	})
}

func TestPEELLMEvaluatorJSONFormatIntegration(t *testing.T) {
	stype, baseURL, model, apiKey, noThink := llm.GetIntegrationTestParams()

	llmCfg := map[string]any{
		"type":     stype,
		"model":    model,
		"base_url": baseURL,
		"api_key":  apiKey,
		"no_think": noThink,
	}
	llmInstance, err := llm.BuildLLM(llmCfg)
	if err != nil {
		t.Fatalf("Failed to build LLM: %v", err)
	}

	evaluator, err := NewPEELLMEvaluatorWithJSONFormat(llmInstance, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	ctx := context.Background()

	t.Run("Verify JSON Parsing", func(t *testing.T) {
		tasks := []*Task{
			{
				ID:     "task_1",
				Tool:   "calculate",
				Status: StatusDone,
				Output: "42",
			},
			{
				ID:     "task_2",
				Tool:   "echo",
				Status: StatusDone,
				Output: "echo: done",
			},
		}

		dag, err := NewDAG(tasks)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		goal := "Complete calculations and echo results"

		result, err := evaluator.Evaluate(ctx, goal, dag)
		if err != nil {
			t.Fatalf("Evaluator failed: %v", err)
		}

		t.Logf("Status: %s", result.Status)
		t.Logf("FinalAnswer: %s", result.FinalAnswer)
		t.Logf("ReplanHint: %s", result.ReplanHint)
		t.Logf("MissingInfo: %v", result.MissingInfo)

		if result.Status != PEEEvalDone && result.Status != PEEEvalReplan && result.Status != PEEEvalFailed {
			t.Errorf("Unexpected status: %s", result.Status)
		}
	})
}
