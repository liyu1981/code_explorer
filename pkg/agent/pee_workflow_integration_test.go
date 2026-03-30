//go:build integration

package workflow

import (
	"context"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/llm"
)

func TestPEEWorkflowRunnerIntegration(t *testing.T) {
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

	toolRegistry := llm.NewToolRegistry()
	toolRegistry.Register(&integrationEchoTool{})

	runner, err := NewRunnerWithJSONFormat(llmInstance, toolRegistry, 3, 5)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	ctx := context.Background()

	t.Run("Simple Echo Workflow", func(t *testing.T) {
		goal := "Use the echo tool to say 'Hello Workflow'"

		result, err := runner.Run(ctx, goal)
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		t.Logf("Workflow result: %s", result)
	})
}
