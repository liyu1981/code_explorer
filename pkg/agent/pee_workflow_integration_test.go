//go:build integration

package agent

import (
	"context"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/tests"
	"github.com/liyu1981/code_explorer/pkg/tools"
)

func TestPEEWorkflowRunnerIntegration(t *testing.T) {
	stype, baseURL, model, apiKey, noThink, _ := tests.GetIntegrationTestParams()

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

	toolRegistry := tools.NewToolRegistry()
	toolRegistry.RegisterTool(&integrationEchoTool{})

	runner, err := NewPEEWorkflowRunnerWithJSONFormat(llmInstance, toolRegistry, 3, 5)
	if err != nil {
		t.Fatalf("Failed to create runner: %v", err)
	}

	ctx := context.Background()

	t.Run("Simple Echo Workflow", func(t *testing.T) {
		goal := "Use the echo tool to say 'Hello Workflow'"

		result, err := runner.Run(ctx, goal, nil)
		if err != nil {
			t.Fatalf("Workflow failed: %v", err)
		}

		t.Logf("Workflow result: %s", result)
	})
}
