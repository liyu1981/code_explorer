package agent

import (
	"context"
	"encoding/json"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

type BaseTool struct {
	name        string
	description string
	parameters  map[string]any
	executeFn   func(ctx context.Context, input json.RawMessage) (string, error)
}

func (t *BaseTool) Name() string        { return t.name }
func (t *BaseTool) Description() string { return t.description }
func (t *BaseTool) Parameters() map[string]any {
	if t.parameters == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		}
	}
	return t.parameters
}

func (t *BaseTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	return t.executeFn(ctx, input)
}

func NewBaseTool(name, description string, fn func(ctx context.Context, input json.RawMessage) (string, error)) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		executeFn:   fn,
	}
}
