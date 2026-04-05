//go:build integration

package agent

import (
	"context"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/tests"
	"github.com/liyu1981/code_explorer/pkg/tools"
)

func TestReactWorkflowRunnerIntegration(t *testing.T) {
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

	registry := tools.NewToolRegistry()
	registry.RegisterTool(&integrationEchoTool{})
	registry.RegisterTool(&integrationCalculateTool{})

	turnID := "test-react-turn"

	runner := NewReactWorkflowRunner(turnID, llmInstance, registry)

	ctx := context.Background()

	t.Run("Direct Answer", func(t *testing.T) {
		goal := "What is the capital of France? Just say the answer."

		result, err := runner.Run(ctx, goal, nil)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
	})

	t.Run("Single Tool Call", func(t *testing.T) {
		goal := "Use the echo tool to say 'Hello ReAct'"

		result, err := runner.Run(ctx, goal, nil)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
	})

	t.Run("Multiple Tool Calls", func(t *testing.T) {
		goal := "First calculate 15 + 25, then use echo to say the result"

		result, err := runner.Run(ctx, goal, nil)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
	})
}
