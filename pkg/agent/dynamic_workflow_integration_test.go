//go:build integration

package agent

import (
	"context"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/tests"
	"github.com/liyu1981/code_explorer/pkg/tools"
)

func TestDynamicRouterIntegration(t *testing.T) {
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

	ctx := context.Background()

	t.Run("Route Simple Question", func(t *testing.T) {
		router := NewDynamicRouter(llmInstance, registry)

		goal := "What is the capital of France?"
		result, err := router.Run(ctx, goal, nil)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
	})

	t.Run("Route Code Investigation", func(t *testing.T) {
		rcRunner, err := NewRCWorkflowRunnerWithJSONFormat(llmInstance, registry)
		if err != nil {
			t.Fatalf("Failed to create RC runner: %v", err)
		}

		router := NewDynamicRouter(llmInstance, registry,
			DynamicWithRCWorkflowRunner(rcRunner),
		)

		goal := "Use the echo tool to say 'Hello Dynamic Routing'"
		result, err := router.Run(ctx, goal, nil)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
	})

	t.Run("Route Complex Task", func(t *testing.T) {
		peeRunner, err := NewPEEWorkflowRunnerWithJSONFormat(llmInstance, registry, 3, 5)
		if err != nil {
			t.Fatalf("Failed to create PEE runner: %v", err)
		}

		router := NewDynamicRouter(llmInstance, registry,
			DynamicWithPEERunner(peeRunner),
		)

		goal := "Use the echo tool to say 'Hello Complex Task'"
		result, err := router.Run(ctx, goal, nil)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		t.Logf("Result: %s", result)
	})
}
