package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/liyu1981/code_explorer/pkg/db"
)

type AgentFactoryInterface interface {
	BuildFromConfig(ctx context.Context, cfg *Config) (*Agent, error)
	GetSkillPrompt(ctx context.Context, name string) (string, error)
	RegisterTool(tool Tool)
}

type AgentFactory struct {
	toolRegistry *ToolRegistry
	store        *db.Store
	defaultLLM   map[string]any
}

func NewAgentFactory(store *db.Store, defaultLLM map[string]any) *AgentFactory {
	return &AgentFactory{
		toolRegistry: NewToolRegistry(),
		store:        store,
		defaultLLM:   defaultLLM,
	}
}

func (f *AgentFactory) RegisterTool(tool Tool) {
	f.toolRegistry.Register(tool)
}

func (f *AgentFactory) Tools() *ToolRegistry {
	return f.toolRegistry
}

func (f *AgentFactory) BuildTestAgent(llm LLM, opts ...AgentOption) *Agent {
	tools := f.toolRegistry
	if tools == nil {
		tools = NewToolRegistry()
	}
	return newAgent(llm, tools, opts...)
}

func (f *AgentFactory) GetSkillPrompt(ctx context.Context, name string) (string, error) {
	if f.store == nil {
		return "", fmt.Errorf("store not initialized in AgentFactory")
	}
	skill, err := f.store.GetSkillByName(ctx, name)
	if err != nil {
		return "", err
	}
	if skill == nil {
		return "", fmt.Errorf("skill %s not found", name)
	}
	return skill.SystemPrompt, nil
}

func (f *AgentFactory) GetSkillTools(ctx context.Context, name string) ([]string, error) {
	if f.store == nil {
		return nil, fmt.Errorf("store not initialized in AgentFactory")
	}
	skill, err := f.store.GetSkillByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if skill == nil {
		return nil, fmt.Errorf("skill %s not found", name)
	}
	if skill.Tools == "" {
		return nil, nil
	}
	return strings.Fields(skill.Tools), nil
}

func (f *AgentFactory) BuildFromConfig(ctx context.Context, cfg *Config) (*Agent, error) {
	llmCfg := cfg.LLM
	if llmCfg == nil {
		llmCfg = f.defaultLLM
	}

	llm, err := f.buildLLM(llmCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build LLM: %w", err)
	}

	contextLength := cfg.ContextLength
	if contextLength <= 0 {
		if cl, ok := llmCfg["context_length"].(int); ok {
			contextLength = cl
		} else if cl, ok := llmCfg["context_length"].(float64); ok {
			contextLength = int(cl)
		}
	}
	if contextLength <= 0 {
		contextLength = 262144
	}

	tools := f.toolRegistry

	if cfg.SkillName != "" && f.store != nil {
		skillTools, err := f.GetSkillTools(ctx, cfg.SkillName)
		if err != nil {
			return nil, fmt.Errorf("failed to get skill tools: %w", err)
		}
		if len(skillTools) > 0 {
			filtered := NewToolRegistry()
			for _, toolName := range skillTools {
				if tool, ok := f.toolRegistry.Get(toolName); ok {
					filtered.Register(tool)
				}
			}
			tools = filtered
		}
	}

	agent := newAgent(llm, tools, WithMaxIterations(cfg.MaxIterations), WithContextLength(contextLength))
	return agent, nil
}

func (f *AgentFactory) buildLLM(cfg map[string]any) (LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("llm config is required")
	}

	llmType, _ := cfg["type"].(string)
	switch llmType {
	case "openai":
		baseURL, _ := cfg["base_url"].(string)
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "qwen3.5:4b"
		}
		apiKey := os.Getenv("LLM_API_KEY")
		if ak, ok := cfg["api_key"].(string); ok {
			apiKey = ak
		}
		return NewHTTPClientLLM(model, baseURL, apiKey), nil

	case "mock":
		model, _ := cfg["model"].(string)
		responses, _ := cfg["responses"].([]any)
		respStrs := make([]string, len(responses))
		for i, r := range responses {
			respStrs[i], _ = r.(string)
		}
		return NewMockLLM(model, respStrs, nil), nil

	default:
		// Fallback for when type is not specified but it looks like an OpenAI-compatible config
		if model, ok := cfg["model"].(string); ok && model != "" {
			baseURL, _ := cfg["base_url"].(string)
			apiKey, _ := cfg["api_key"].(string)
			return NewHTTPClientLLM(model, baseURL, apiKey), nil
		}
		return nil, fmt.Errorf("unknown llm type: %s", llmType)
	}
}
