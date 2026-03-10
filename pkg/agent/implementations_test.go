package agent

import (
	"context"
	"strings"
	"testing"
)

func TestPromptTemplateStep(t *testing.T) {
	template := "Hello, {{name}}! Your task is: {{input}}"
	vars := map[string]string{"name": "Alice"}
	step := NewPromptTemplateStep(template, vars)

	ctx := context.Background()
	got, err := step.Execute(ctx, "Write a test")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	expected := "Hello, Alice! Your task is: Write a test"
	if got != expected {
		t.Errorf("got %q; want %q", got, expected)
	}
}

func TestRouterStep(t *testing.T) {
	stepA := NewPipelineStepFromFunc("stepA", func(ctx context.Context, input string) (string, error) {
		return "Handled by A: " + input, nil
	})
	stepB := NewPipelineStepFromFunc("stepB", func(ctx context.Context, input string) (string, error) {
		return "Handled by B: " + input, nil
	})

	routes := map[string]PipelineStep{
		"apple": stepA,
		"banana": stepB,
	}
	router := NewRouterStep(routes)

	ctx := context.Background()

	t.Run("RouteA", func(t *testing.T) {
		got, _ := router.Execute(ctx, "I like apples")
		if !strings.Contains(got, "Handled by A") {
			t.Errorf("expected Route A, got %q", got)
		}
	})

	t.Run("RouteB", func(t *testing.T) {
		got, _ := router.Execute(ctx, "Bananas are great")
		if !strings.Contains(got, "Handled by B") {
			t.Errorf("expected Route B, got %q", got)
		}
	})

	t.Run("Default", func(t *testing.T) {
		got, _ := router.Execute(ctx, "Cherry is nice")
		if got != "Cherry is nice" {
			t.Errorf("expected original input, got %q", got)
		}
	})
}
