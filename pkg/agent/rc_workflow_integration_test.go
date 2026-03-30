//go:build integration

package workflow

import (
	"context"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/llm"
)

func TestRCWorkflowRunnerIntegration(t *testing.T) {
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

	registry := llm.NewToolRegistry()
	registry.Register(&integrationEchoTool{})
	registry.Register(&integrationCalculateTool{})

	ctx := context.Background()

	t.Run("Direct Answer", func(t *testing.T) {
		runner, err := NewRCWorkflowRunnerWithJSONFormat(llmInstance, registry)
		if err != nil {
			t.Fatalf("Failed to create runner: %v", err)
		}
		goal := "What is the capital of Japan? Just answer."

		result, err := runner.Run(ctx, goal)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
	})

	t.Run("Single Tool Call", func(t *testing.T) {
		runner, err := NewRCWorkflowRunnerWithJSONFormat(llmInstance, registry)
		if err != nil {
			t.Fatalf("Failed to create runner: %v", err)
		}
		goal := "Use the echo tool to say 'Hello Reflect-Critic'"

		result, err := runner.Run(ctx, goal)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
		t.Logf("History steps: %d", len(runner.History()))
	})

	t.Run("Tool With Critique", func(t *testing.T) {
		runner, err := NewRCWorkflowRunnerWithJSONFormat(llmInstance, registry,
			RCWithMaxReflections(3),
			RCWithMaxIterations(5),
		)
		if err != nil {
			t.Fatalf("Failed to create runner: %v", err)
		}
		goal := "Calculate 25 + 17 using the calculate tool"

		result, err := runner.Run(ctx, goal)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
		t.Logf("History steps: %d", len(runner.History()))
		for i, step := range runner.History() {
			t.Logf("  Step %d: Draft=%q, Critique.HasIssues=%v", i, step.Draft, step.Critique != nil && step.Critique.HasIssues)
		}
	})
}
