package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type integrationEchoTool struct{}

func (t *integrationEchoTool) Name() string { return "echo" }
func (t *integrationEchoTool) Description() string {
	return "Echoes the input message back"
}
func (t *integrationEchoTool) Clone() agent.Tool { return &integrationEchoTool{} }
func (t *integrationEchoTool) Bind(ctx context.Context, state *map[string]any) error {
	return nil
}
func (t *integrationEchoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string", "description": "Message to echo back"},
		},
		"required": []string{"message"},
	}
}
func (t *integrationEchoTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(input, &req); err != nil {
		return "", err
	}
	return fmt.Sprintf("echo: %s", req.Message), nil
}

type integrationCalculateTool struct{}

func (t *integrationCalculateTool) Name() string { return "calculate" }
func (t *integrationCalculateTool) Description() string {
	return "Performs basic arithmetic operations"
}
func (t *integrationCalculateTool) Clone() agent.Tool { return &integrationCalculateTool{} }
func (t *integrationCalculateTool) Bind(ctx context.Context, state *map[string]any) error {
	return nil
}
func (t *integrationCalculateTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "Operation: add, sub, mul",
				"enum":        []any{"add", "sub", "mul"},
			},
			"a": map[string]any{"type": "integer", "description": "First number"},
			"b": map[string]any{"type": "integer", "description": "Second number"},
		},
		"required": []string{"operation", "a", "b"},
	}
}
func (t *integrationCalculateTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
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
