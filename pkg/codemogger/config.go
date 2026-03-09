package codemogger

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
