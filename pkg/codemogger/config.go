package codemogger

type Config struct {
	DBPath     string         `json:"db_path,omitempty"`
	Embedder   EmbedderConfig `json:"embedder"`
	Languages  []string       `json:"languages,omitempty"`
	ChunkLines int            `json:"chunk_lines,omitempty"`
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
		ChunkLines: 150,
		Embedder: EmbedderConfig{
			Type:  "local",
			Model: "all-MiniLM-L6-v2",
		},
		Languages: []string{"go", "rust", "python", "typescript", "javascript"},
	}
}
