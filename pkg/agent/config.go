package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

type AgentFactory struct {
	toolRegistry *ToolRegistry
}

func NewAgentFactory() *AgentFactory {
	return &AgentFactory{
		toolRegistry: NewToolRegistry(),
	}
}

func (f *AgentFactory) RegisterTool(tool Tool) {
	f.toolRegistry.Register(tool)
}

func (f *AgentFactory) Tools() *ToolRegistry {
	return f.toolRegistry
}

func (f *AgentFactory) BuildFromConfig(cfg *Config) (*Agent, error) {
	llm, err := f.buildLLM(cfg.LLM)
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
		endpoint, _ := cfg["endpoint"].(string)
		if endpoint == "" {
			endpoint = "https://api.openai.com/v1/chat/completions"
		}
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "gpt-4"
		}
		apiKey := os.Getenv("LLM_API_KEY")
		if ak, ok := cfg["api_key"].(string); ok {
			apiKey = ak
		}
		return NewHTTPClientLLM(model, endpoint, apiKey), nil

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

var configTypeMap = map[string]reflect.Type{
	"string":   reflect.TypeOf(""),
	"int":      reflect.TypeOf(0),
	"float64":  reflect.TypeOf(float64(0)),
	"bool":     reflect.TypeOf(false),
	"[]string": reflect.TypeOf([]string{}),
	"[]any":    reflect.TypeOf([]any{}),
}
