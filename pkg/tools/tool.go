package tools

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/liyu1981/code_explorer/pkg/protocol"
)

var (
	globalToolRegistry     *ToolRegistry
	globalToolRegistryOnce sync.Once
)

type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error)
}

type ToolRegistry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]Tool)}
}

func InitGlobalToolRegistry() {
	globalToolRegistryOnce.Do(func() {
		globalToolRegistry = NewToolRegistry()
	})
}

func GetGlobalToolRegistry() *ToolRegistry {
	if globalToolRegistry == nil {
		InitGlobalToolRegistry()
	}
	return globalToolRegistry
}

func (r *ToolRegistry) RegisterTool(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *ToolRegistry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

func (r *ToolRegistry) MarshalToolsForLLM() []map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
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
