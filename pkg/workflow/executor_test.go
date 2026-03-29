package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type integrationEchoTool struct{}

func (t *integrationEchoTool) Name() string { return "echo" }
func (t *integrationEchoTool) Description() string {
	return "Echoes the input message back"
}
func (t *integrationEchoTool) Clone() agent.Tool { return &integrationEchoTool{} }
func (t *integrationEchoTool) Bind(ctx context.Context, state *map[string]any) error {
	return nil
}
func (t *integrationEchoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string", "description": "Message to echo back"},
		},
		"required": []string{"message"},
	}
}
func (t *integrationEchoTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}
	return fmt.Sprintf("echo: %s", req.Message), nil
}

type integrationCalculateTool struct{}

func (t *integrationCalculateTool) Name() string { return "calculate" }
func (t *integrationCalculateTool) Description() string {
	return "Performs basic arithmetic operations"
}
func (t *integrationCalculateTool) Clone() agent.Tool { return &integrationCalculateTool{} }
func (t *integrationCalculateTool) Bind(ctx context.Context, state *map[string]any) error {
	return nil
}
func (t *integrationCalculateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "Operation: add, sub, mul",
				"enum":        []any{"add", "sub", "mul"},
			},
			"a": map[string]any{"type": "integer", "description": "First number"},
			"b": map[string]any{"type": "integer", "description": "Second number"},
		},
		"required": []string{"operation", "a", "b"},
	}
}
func (t *integrationCalculateTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Operation string `json:"operation"`
		A         int    `json:"a"`
		B         int    `json:"b"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}
	switch req.Operation {
	case "add":
		return fmt.Sprintf("%d", req.A+req.B), nil
	case "sub":
		return fmt.Sprintf("%d", req.A-req.B), nil
	case "mul":
		return fmt.Sprintf("%d", req.A*req.B), nil
	default:
		return "", fmt.Errorf("unknown operation: %s", req.Operation)
	}
}

func TestExecutor(t *testing.T) {
	registry := agent.NewToolRegistry()
	registry.Register(&integrationEchoTool{})
	registry.Register(&integrationCalculateTool{})

	executor := NewExecutor(registry, 3)

	ctx := context.Background()

	t.Run("Execute Single Task", func(t *testing.T) {
		tasks := []*Task{
			{
				ID:   "echo_task",
				Tool: "echo",
				Input: map[string]any{
					"message": "Hello Executor Test",
				},
				Status: StatusPending,
			},
		}

		dag, err := NewDAG(tasks)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		err = executor.Execute(ctx, dag)
		if err != nil {
			t.Fatalf("Executor failed: %v", err)
		}

		task, ok := dag.GetTask("echo_task")
		if !ok {
			t.Fatal("Task not found")
		}

		if task.Status != StatusDone {
			t.Errorf("Expected status Done, got %s", task.Status)
		}

		if task.Output == nil {
			t.Fatal("Expected non-nil output")
		}

		t.Logf("Task output: %v", task.Output)
	})

	t.Run("Execute Multiple Independent Tasks", func(t *testing.T) {
		tasks := []*Task{
			{
				ID:   "calc_1",
				Tool: "calculate",
				Input: map[string]any{
					"operation": "add",
					"a":         10,
					"b":         20,
				},
				Status: StatusPending,
			},
			{
				ID:   "calc_2",
				Tool: "calculate",
				Input: map[string]any{
					"operation": "mul",
					"a":         5,
					"b":         6,
				},
				Status: StatusPending,
			},
		}

		dag, err := NewDAG(tasks)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		err = executor.Execute(ctx, dag)
		if err != nil {
			t.Fatalf("Executor failed: %v", err)
		}

		task1, _ := dag.GetTask("calc_1")
		task2, _ := dag.GetTask("calc_2")

		if task1.Status != StatusDone || task2.Status != StatusDone {
			t.Errorf("Expected both tasks Done, got %s and %s", task1.Status, task2.Status)
		}

		t.Logf("calc_1 output: %v", task1.Output)
		t.Logf("calc_2 output: %v", task2.Output)
	})

	t.Run("Execute Tasks With Dependencies", func(t *testing.T) {
		tasks := []*Task{
			{
				ID:   "first_calc",
				Tool: "calculate",
				Input: map[string]any{
					"operation": "add",
					"a":         5,
					"b":         3,
				},
				Status: StatusPending,
			},
			{
				ID:   "second_calc",
				Tool: "calculate",
				Input: map[string]any{
					"operation": "mul",
					"a":         10,
					"b":         2,
				},
				DependsOn: []string{"first_calc"},
				Status:    StatusPending,
			},
		}

		dag, err := NewDAG(tasks)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		err = executor.Execute(ctx, dag)
		if err != nil {
			t.Fatalf("Executor failed: %v", err)
		}

		task1, _ := dag.GetTask("first_calc")
		task2, _ := dag.GetTask("second_calc")

		if task1.Status != StatusDone {
			t.Errorf("Expected first_calc Done, got %s", task1.Status)
		}
		if task2.Status != StatusDone {
			t.Errorf("Expected second_calc Done, got %s", task2.Status)
		}

		t.Logf("first_calc output: %v", task1.Output)
		t.Logf("second_calc output: %v", task2.Output)
	})

	t.Run("Execute Unknown Tool", func(t *testing.T) {
		tasks := []*Task{
			{
				ID:   "unknown_task",
				Tool: "nonexistent_tool",
				Input: map[string]any{
					"data": "test",
				},
				Status: StatusPending,
			},
		}

		dag, err := NewDAG(tasks)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		err = executor.Execute(ctx, dag)
		if err != nil {
			t.Fatalf("Executor failed: %v", err)
		}

		task, _ := dag.GetTask("unknown_task")
		if task.Status != StatusFailed {
			t.Errorf("Expected status Failed, got %s", task.Status)
		}

		if task.Err == nil {
			t.Error("Expected non-nil error")
		} else {
			t.Logf("Expected error: %v", task.Err)
		}
	})

	t.Run("Skip Dependents On Failure", func(t *testing.T) {
		tasks := []*Task{
			{
				ID:   "failing_task",
				Tool: "nonexistent",
				Input: map[string]any{
					"data": "test",
				},
				Status: StatusPending,
			},
			{
				ID:        "dependent_task_1",
				Tool:      "echo",
				Input:     map[string]any{"message": "should be skipped"},
				DependsOn: []string{"failing_task"},
				Status:    StatusPending,
			},
			{
				ID:        "dependent_task_2",
				Tool:      "calculate",
				Input:     map[string]any{"operation": "add", "a": 1, "b": 2},
				DependsOn: []string{"dependent_task_1"},
				Status:    StatusPending,
			},
		}

		dag, err := NewDAG(tasks)
		if err != nil {
			t.Fatalf("Failed to create DAG: %v", err)
		}

		err = executor.Execute(ctx, dag)
		if err != nil {
			t.Fatalf("Executor failed: %v", err)
		}

		task1, _ := dag.GetTask("failing_task")
		task2, _ := dag.GetTask("dependent_task_1")
		task3, _ := dag.GetTask("dependent_task_2")

		if task1.Status != StatusFailed {
			t.Errorf("Expected failing_task Failed, got %s", task1.Status)
		}
		if task2.Status != StatusSkipped {
			t.Errorf("Expected dependent_task_1 Skipped, got %s", task2.Status)
		}
		if task3.Status != StatusSkipped {
			t.Errorf("Expected dependent_task_2 Skipped, got %s", task3.Status)
		}
	})
}
