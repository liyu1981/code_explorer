//go:build integration

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type echoTool struct{}

func (t *echoTool) Name() string                                          { return "echo" }
func (t *echoTool) Description() string                                   { return "Echoes the input back" }
func (t *echoTool) Clone() Tool                                           { return &echoTool{} }
func (t *echoTool) Bind(ctx context.Context, state *map[string]any) error { return nil }
func (t *echoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string", "description": "Message to echo back"},
		},
		"required": []string{"message"},
	}
}
func (t *echoTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}
	return fmt.Sprintf("echo: %s", req.Message), nil
}

type calculateTool struct{}

func (t *calculateTool) Name() string                                          { return "calculate" }
func (t *calculateTool) Description() string                                   { return "Performs basic arithmetic" }
func (t *calculateTool) Clone() Tool                                           { return &calculateTool{} }
func (t *calculateTool) Bind(ctx context.Context, state *map[string]any) error { return nil }
func (t *calculateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "Operation to perform: add, sub, mul",
				"enum":        []any{"add", "sub", "mul"},
			},
			"a": map[string]any{"type": "integer", "description": "First number"},
			"b": map[string]any{"type": "integer", "description": "Second number"},
		},
		"required": []string{"operation", "a", "b"},
	}
}
func (t *calculateTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Operation string `json:"operation"`
		A         int    `json:"a"`
		B         int    `json:"b"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}

	switch req.Operation {
	case "add":
		return fmt.Sprintf("%d", req.A+req.B), nil
	case "sub":
		return fmt.Sprintf("%d", req.A-req.B), nil
	case "mul":
		return fmt.Sprintf("%d", req.A*req.B), nil
	default:
		return "", fmt.Errorf("unknown operation: %s", req.Operation)
	}
}

func TestAgentIntegration(t *testing.T) {
	baseURL := "http://localhost:20003/v1"
	model := "unsloth/Qwen3.5-9B-GGUF:Q4_K_M"

	registry := NewToolRegistry()
	registry.Register(&echoTool{})
	registry.Register(&calculateTool{})

	llm := NewHTTPClientLLM(model, baseURL, "")
	agentInstance := newAgent(llm, "", "", registry, WithMaxIterations(5))

	ctx := context.Background()

	t.Run("Basic Echo", func(t *testing.T) {
		prompt := "Please echo the message 'Hello Integration Test' using the echo tool."
		result, err := agentInstance.RunLoop(ctx, prompt, nil, nil, 5)
		if err != nil {
			t.Fatalf("Agent run failed: %v", err)
		}
		t.Logf("Result: %s", result)
	})

	t.Run("Multi-step Calculation", func(t *testing.T) {
		prompt := "Calculate (12 + 34) * 2 using the calculate tool."
		result, err := agentInstance.RunLoop(ctx, prompt, nil, nil, 5)
		if err != nil {
			t.Fatalf("Agent run failed: %v", err)
		}
		t.Logf("Result: %s", result)
	})
}
