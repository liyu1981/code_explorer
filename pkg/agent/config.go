package agent

import (
	"encoding/json"
	"os"
)

type Config struct {
	LLM             map[string]any `json:"llm"`
	Tools           []string       `json:"tools"`
	MaxIterations   int            `json:"max_iterations"`
	ContextLength   int            `json:"context_length"`
	AgentPromptName string         `json:"agent_prompt_name"`
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
