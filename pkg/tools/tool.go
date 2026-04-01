package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/protocol"
	"github.com/rs/zerolog/log"
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

func (r *ToolRegistry) InitTools(data map[string]any) error {
	if data == nil {
		return fmt.Errorf("data is nil")
	}

	cmIndex, ok := data["codemogger_index"]
	if !ok || cmIndex == nil {
		return fmt.Errorf("codemogger index is required")
	}

	r.RegisterTool(NewCodeMoggerListFilesTool(cmIndex.(*codemogger.CodeIndex)))
	log.Debug().Msg("Load tool codemogger_list_files")
	r.RegisterTool(NewCodeMoggerSearchTool(cmIndex.(*codemogger.CodeIndex)))
	log.Debug().Msg("Load tool codgemogger_search")
	r.RegisterTool(NewReadFileTool())
	log.Debug().Msg("Load global tool read_file")
	r.RegisterTool(NewGetTreeTool())
	log.Debug().Msg("Load global tool get_tree")
	r.RegisterTool(NewGrepSearchTool())
	log.Debug().Msg("Load global tool grep_search")

	tools := r.List()
	var toolNames []string
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name())
	}
	log.Info().Interface("tools", toolNames).Msg("Registered tools")
	return nil
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
