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
	state       map[string]any
	bindFn      func(state map[string]any) error
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

func (t *BaseTool) Bind(ctx context.Context, state map[string]any) error {
	if t.bindFn != nil {
		return t.bindFn(state)
	}
	return nil
}

func (t *BaseTool) Clone() *BaseTool {
	return NewBaseTool(t.name, t.description, t.executeFn, t.bindFn)
}

func NewBaseTool(
	name,
	description string,
	executeFn func(ctx context.Context, input json.RawMessage) (string, error),
	bindFn func(state map[string]any) error,
) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		executeFn:   executeFn,
		bindFn:      bindFn,
	}
}

type BaseToolRegistry struct {
	tools map[string]*BaseTool
}

func NewBaseToolRegistry() *BaseToolRegistry {
	return &BaseToolRegistry{
		tools: make(map[string]*BaseTool),
	}
}

func (t *BaseToolRegistry) RegisterTool(tool *BaseTool) {
	t.tools[tool.Name()] = tool
}

func (t *BaseToolRegistry) GetTool(name string) (*BaseTool, bool) {
	tool, ok := t.tools[name]
	return tool, ok
}

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error)
}

type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) List() []Tool {
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

func (r *ToolRegistry) MarshalToolsForLLM() []map[string]any {
	result := make([]map[string]any, 0, len(r.tools))
	for _, t := range r.tools {
		params := t.Parameters()
		if params == nil {
			params = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			}
		}
		result = append(result, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  params,
			},
		})
	}
	return result
}
