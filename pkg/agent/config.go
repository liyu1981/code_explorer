package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/liyu1981/code_explorer/pkg/db"
)

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

func (f *AgentFactory) BuildFromConfig(cfg *Config) (*Agent, error) {
	llmCfg := cfg.LLM
	if llmCfg == nil {
		llmCfg = f.defaultLLM
	}

	llm, err := f.buildLLM(llmCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build LLM: %w", err)
	}

	agent := NewAgent(llm, f.toolRegistry, WithMaxIterations(cfg.MaxIterations))
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
		return nil, fmt.Errorf("unknown llm type: %s", llmType)
	}
}

func LoadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveConfigToFile(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
