//go:build integration

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type testEchoTool struct{}

func (t *testEchoTool) Name() string { return "echo" }
func (t *testEchoTool) Description() string {
	return "Echoes the input message back"
}
func (t *testEchoTool) Clone() agent.Tool { return &testEchoTool{} }
func (t *testEchoTool) Bind(ctx context.Context, state *map[string]any) error {
	return nil
}
func (t *testEchoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string", "description": "Message to echo back"},
		},
		"required": []string{"message"},
	}
}
func (t *testEchoTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}
	return fmt.Sprintf("echo: %s", req.Message), nil
}

func TestPEEWorkflowRunnerIntegration(t *testing.T) {
	baseURL, model, _ := agent.GetIntegrationTestParams()

	llmCfg := map[string]any{
		"type":     "openai",
		"model":    model,
		"base_url": baseURL,
		"api_key":  "",
	}
	llm, err := agent.BuildLLM(llmCfg)
	if err != nil {
		t.Fatalf("Failed to build LLM: %v", err)
	}

	toolRegistry := agent.NewToolRegistry()
	toolRegistry.Register(&testEchoTool{})

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
