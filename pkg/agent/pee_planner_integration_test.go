//go:build integration

package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/llm"
)

func TestPEELLMPlannerIntegration(t *testing.T) {
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

	planner, err := NewPEELLMPlannerWithJSONFormat(llmInstance, nil)
	if err != nil {
		t.Fatalf("Failed to create planner: %v", err)
	}

	ctx := context.Background()

	t.Run("Simple Goal Planning", func(t *testing.T) {
		req := PEEPlanRequest{
			Goal:      "Search for information about Go programming",
			Iteration: 1,
		}

		dag, err := planner.Plan(ctx, req)
		if err != nil {
			t.Fatalf("Planner failed: %v", err)
		}

		if dag == nil {
			t.Fatal("Expected non-nil DAG")
		}

		tasks := dag.tasks
		if len(tasks) == 0 {
			t.Fatal("Expected at least one task in the DAG")
		}

		t.Logf("Planned %d tasks:", len(tasks))
		for id, task := range tasks {
			t.Logf("  - Task: %s, Tool: %s", id, task.Tool)
			if task.ID == "" {
				t.Error("Task ID should not be empty")
			}
			if task.Tool == "" {
				t.Error("Task tool should not be empty")
			}
		}
	})

	t.Run("Multi-step Goal Planning", func(t *testing.T) {
		req := PEEPlanRequest{
			Goal:      "First search for Go best practices, then summarize the results",
			Iteration: 1,
		}

		dag, err := planner.Plan(ctx, req)
		if err != nil {
			t.Fatalf("Planner failed: %v", err)
		}

		if dag == nil {
			t.Fatal("Expected non-nil DAG")
		}

		tasks := dag.tasks
		if len(tasks) < 2 {
			t.Logf("Note: Only %d task(s) planned", len(tasks))
		}

		t.Logf("Planned %d tasks:", len(tasks))
		for id, task := range tasks {
			t.Logf("  - Task: %s, Tool: %s, DependsOn: %v", id, task.Tool, task.DependsOn)
		}
	})

	t.Run("Planning With Prior Context", func(t *testing.T) {
		req := PEEPlanRequest{
			Goal: "Find files related to user authentication",
			PriorOutputs: map[string]any{
				"search_results": "Found 5 relevant files in src/auth/",
			},
			Iteration: 2,
		}

		dag, err := planner.Plan(ctx, req)
		if err != nil {
			t.Fatalf("Planner failed: %v", err)
		}

		if dag == nil {
			t.Fatal("Expected non-nil DAG")
		}

		t.Logf("Planned %d tasks with prior context:", len(dag.tasks))
		for id, task := range dag.tasks {
			t.Logf("  - Task: %s, Tool: %s", id, task.Tool)
		}
	})

	t.Run("Planning After Failed Tasks", func(t *testing.T) {
		req := PEEPlanRequest{
			Goal:      "Read and analyze the main configuration file",
			Iteration: 3,
			FailedTasks: []*Task{
				{ID: "read_config", Err: nil, Status: StatusFailed},
			},
			MissingInfo: []string{"The config file path was not found"},
		}

		dag, err := planner.Plan(ctx, req)
		if err != nil {
			t.Fatalf("Planner failed: %v", err)
		}

		if dag == nil {
			t.Fatal("Expected non-nil DAG")
		}

		t.Logf("Replanned %d tasks after failure:", len(dag.tasks))
		for id, task := range dag.tasks {
			t.Logf("  - Task: %s, Tool: %s", id, task.Tool)
		}
	})
}

func TestPEELLMPlannerJSONOutputIntegration(t *testing.T) {
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

	planner, err := NewPEELLMPlannerWithJSONFormat(llmInstance, nil)
	if err != nil {
		t.Fatalf("Failed to create planner: %v", err)
	}

	ctx := context.Background()

	t.Run("Verify JSON Parsing", func(t *testing.T) {
		req := PEEPlanRequest{
			Goal:      "Calculate 5 + 3 using the calculate tool",
			Iteration: 1,
		}

		dag, err := planner.Plan(ctx, req)
		if err != nil {
			t.Fatalf("Planner failed: %v", err)
		}

		for id, task := range dag.tasks {
			if task.Input != nil {
				inputJSON, err := json.Marshal(task.Input)
				if err != nil {
					t.Errorf("Task %s has invalid input: %v", id, err)
				} else {
					t.Logf("Task %s input: %s", id, string(inputJSON))
				}
			}

			if len(task.DependsOn) > 0 {
				t.Logf("Task %s depends on: %v", id, task.DependsOn)
			}
		}
	})
}
