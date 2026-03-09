package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/liyu1981/code_explorer/pkg/util"
)

type Config struct {
	System     SystemConfig     `json:"system"`
	Research   ResearchConfig   `json:"research"`
	CodeMogger CodeMoggerConfig `json:"codemogger"`
}

type SystemConfig struct {
	DBPath string         `json:"db_path,omitempty"`
	LLM    map[string]any `json:"llm,omitempty"`
}

type ResearchConfig struct {
	MaxReportsPerCodebase int `json:"max_reports_per_codebase"`
}

type CodeMoggerConfig struct {
	InheritSystemLLM bool           `json:"inherit_system_llm"`
	Embedder         EmbedderConfig `json:"embedder"`
	Languages        []string       `json:"languages,omitempty"`
	ChunkLines       int            `json:"chunk_lines,omitempty"`
}

type EmbedderConfig struct {
	Type   string       `json:"type"` // "local" or "openai"
	Model  string       `json:"model"`
	OpenAI OpenAIConfig `json:"openai,omitempty"`
}

type OpenAIConfig struct {
	APIBase string `json:"api_base,omitempty"`
	Model   string `json:"model,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
}

var (
	instance *Config
	mu       sync.RWMutex
	once     sync.Once
	path     string
)

// Get returns the singleton config instance
func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	if instance == nil {
		return DefaultConfig()
	}
	return instance
}

// Set updates the singleton config instance
func Set(cfg *Config) {
	mu.Lock()
	defer mu.Unlock()
	instance = cfg
}

// GetPath returns the path to the config file
func GetPath() string {
	mu.RLock()
	defer mu.RUnlock()
	return path
}

// Load loads config from file and sets the singleton
func Load(configPath string) error {
	mu.Lock()
	defer mu.Unlock()

	path = configPath
	if configPath == "" {
		if _, err := os.Stat(".config.json"); err == nil {
			path = ".config.json"
		} else {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, ".code_explorer", "config.json")
		}
	}

	cfg := DefaultConfig()
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		var fileCfg Config
		if err := json.Unmarshal(data, &fileCfg); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		// Merge with defaults
		if fileCfg.System.DBPath != "" {
			cfg.System.DBPath = util.ExpandPath(fileCfg.System.DBPath)
		}
		if fileCfg.System.LLM != nil {
			cfg.System.LLM = fileCfg.System.LLM
		}
		if fileCfg.Research.MaxReportsPerCodebase > 0 {
			cfg.Research.MaxReportsPerCodebase = fileCfg.Research.MaxReportsPerCodebase
		}
		cfg.CodeMogger.InheritSystemLLM = fileCfg.CodeMogger.InheritSystemLLM
		if fileCfg.CodeMogger.Embedder.Type != "" {
			cfg.CodeMogger.Embedder.Type = fileCfg.CodeMogger.Embedder.Type
		}
		if fileCfg.CodeMogger.Embedder.Model != "" {
			cfg.CodeMogger.Embedder.Model = fileCfg.CodeMogger.Embedder.Model
		}
		if fileCfg.CodeMogger.Embedder.OpenAI.APIBase != "" {
			cfg.CodeMogger.Embedder.OpenAI.APIBase = fileCfg.CodeMogger.Embedder.OpenAI.APIBase
		}
		if fileCfg.CodeMogger.Embedder.OpenAI.APIKey != "" {
			cfg.CodeMogger.Embedder.OpenAI.APIKey = fileCfg.CodeMogger.Embedder.OpenAI.APIKey
		}
		if fileCfg.CodeMogger.Embedder.OpenAI.Model != "" {
			cfg.CodeMogger.Embedder.OpenAI.Model = fileCfg.CodeMogger.Embedder.OpenAI.Model
		}
		if fileCfg.CodeMogger.ChunkLines > 0 {
			cfg.CodeMogger.ChunkLines = fileCfg.CodeMogger.ChunkLines
		}
		if len(fileCfg.CodeMogger.Languages) > 0 {
			cfg.CodeMogger.Languages = fileCfg.CodeMogger.Languages
		}
	}

	instance = cfg
	return nil
}

// Save persists the current singleton config to file
func Save() error {
	mu.RLock()
	cfg := instance
	configPath := path
	mu.RUnlock()

	if cfg == nil {
		return fmt.Errorf("config instance is nil")
	}
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".code_explorer", "config.json")
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

func DefaultConfig() *Config {
	return &Config{
		System: SystemConfig{
			LLM: map[string]any{
				"type":     "openai",
				"model":    "gpt-4o",
				"endpoint": "https://api.openai.com/v1/chat/completions",
			},
		},
		Research: ResearchConfig{
			MaxReportsPerCodebase: 10,
		},
		CodeMogger: CodeMoggerConfig{
			InheritSystemLLM: true,
			ChunkLines:       150,
			Embedder: EmbedderConfig{
				Type:  "local",
				Model: "all-minilm:l6-v2",
			},
			Languages: []string{"go", "rust", "python", "typescript", "javascript"},
		},
	}
}
