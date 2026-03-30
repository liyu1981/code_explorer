//go:build integration

package workflow

import (
	"context"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/llm"
)

func TestSimpleWorkflowRunnerIntegration(t *testing.T) {
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

	runner := NewSimpleWorkflowRunner(llmInstance)

	ctx := context.Background()

	t.Run("Simple Question", func(t *testing.T) {
		goal := "What is the capital of France?"

		result, err := runner.Run(ctx, goal)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if result == "" {
			t.Fatal("Expected non-empty result")
		}

		t.Logf("Result: %s", result)
	})

	t.Run("Definition", func(t *testing.T) {
		goal := "Define: photosynthesis"

		result, err := runner.Run(ctx, goal)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if result == "" {
			t.Fatal("Expected non-empty result")
		}

		t.Logf("Result: %s", result)
	})

	t.Run("Factual Query", func(t *testing.T) {
		goal := "Who was the first person to walk on the moon?"

		result, err := runner.Run(ctx, goal)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if result == "" {
			t.Fatal("Expected non-empty result")
		}

		t.Logf("Result: %s", result)
	})
}
