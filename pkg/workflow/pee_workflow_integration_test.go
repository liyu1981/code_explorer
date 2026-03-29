//go:build integration

package workflow

import (
	"context"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/agent"
)

func TestPEEWorkflowRunnerIntegration(t *testing.T) {
	stype, baseURL, model, apiKey, noThink := agent.GetIntegrationTestParams()

	llmCfg := map[string]any{
		"type":     stype,
		"model":    model,
		"base_url": baseURL,
		"api_key":  apiKey,
		"no_think": noThink,
	}
	llm, err := agent.BuildLLM(llmCfg)
	if err != nil {
		t.Fatalf("Failed to build LLM: %v", err)
	}

	toolRegistry := agent.NewToolRegistry()
	toolRegistry.Register(&integrationEchoTool{})

	runner, err := NewRunnerWithJSONFormat(llm, toolRegistry, 3, 5)
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
