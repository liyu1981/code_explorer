//go:build integration

package workflow

import (
	"context"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/llm"
)

func TestDynamicRouterIntegration(t *testing.T) {
	stype, baseURL, model, apiKey, noThink := llm.GetIntegrationTestParams()

	llmCfg := map[string]any{
		"type":     stype,
		"model":    model,
		"base_url": baseURL,
		"api_key":  apiKey,
		"no_think": noThink,
	}
	llm, err := llm.BuildLLM(llmCfg)
	if err != nil {
		t.Fatalf("Failed to build LLM: %v", err)
	}

	registry := llm.NewToolRegistry()
	registry.Register(&integrationEchoTool{})
	registry.Register(&integrationCalculateTool{})

	ctx := context.Background()

	t.Run("Route Simple Question", func(t *testing.T) {
		router := NewDynamicRouter(llm, registry)

		goal := "What is the capital of France?"
		result, err := router.Run(ctx, goal)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
	})

	t.Run("Route Code Investigation", func(t *testing.T) {
		rcRunner, err := NewRCWorkflowRunnerWithJSONFormat(llm, registry)
		if err != nil {
			t.Fatalf("Failed to create RC runner: %v", err)
		}

		router := NewDynamicRouter(llm, registry,
			DynamicWithRCWorkflowRunner(rcRunner),
		)

		goal := "Use the echo tool to say 'Hello Dynamic Routing'"
		result, err := router.Run(ctx, goal)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
	})

	t.Run("Route Complex Task", func(t *testing.T) {
		peeRunner, err := NewRunnerWithJSONFormat(llm, registry, 3, 5)
		if err != nil {
			t.Fatalf("Failed to create PEE runner: %v", err)
		}

		router := NewDynamicRouter(llm, registry,
			DynamicWithPEERunner(peeRunner),
		)

		goal := "Use the echo tool to say 'Hello Complex Task'"
		result, err := router.Run(ctx, goal)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
	})
}
