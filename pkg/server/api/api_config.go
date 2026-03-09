package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
)

func (h *ApiHandler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}

	cfg := *h.index.Config
	// Mask API keys
	if cfg.System.LLM != nil {
		if apiKey, ok := cfg.System.LLM["api_key"].(string); ok && apiKey != "" {
			cfg.System.LLM["api_key"] = "****"
		}
	}
	if cfg.CodeMogger.Embedder.OpenAI.APIKey != "" {
		cfg.CodeMogger.Embedder.OpenAI.APIKey = "****"
	}

	// Create a response structure that matches the frontend Config interface
	type systemResp struct {
		DbPath      string         `json:"db_path"`
		IsDefaultDb bool           `json:"is_default_db"`
		LLM         map[string]any `json:"llm"`
	}

	res := struct {
		System     systemResp                  `json:"system"`
		Research   codemogger.ResearchConfig   `json:"research"`
		CodeMogger codemogger.CodeMoggerConfig `json:"codemogger"`
	}{
		System: systemResp{
			DbPath:      h.index.GetDbPath(),
			IsDefaultDb: h.index.Config.System.DBPath == "",
			LLM:         cfg.System.LLM,
		},
		Research:   cfg.Research,
		CodeMogger: cfg.CodeMogger,
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *ApiHandler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}

	var newCfg codemogger.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Update current config
	current := h.index.Config

	// System
	if newCfg.System.LLM != nil {
		if current.System.LLM == nil {
			current.System.LLM = make(map[string]any)
		}
		for k, v := range newCfg.System.LLM {
			if k == "api_key" {
				if vStr, ok := v.(string); ok && vStr != "" && vStr != "****" {
					current.System.LLM[k] = vStr
				}
			} else {
				current.System.LLM[k] = v
			}
		}
	}

	// Research
	if newCfg.Research.MaxReportsPerCodebase > 0 {
		current.Research.MaxReportsPerCodebase = newCfg.Research.MaxReportsPerCodebase
	}

	// CodeMogger
	current.CodeMogger.InheritSystemLLM = newCfg.CodeMogger.InheritSystemLLM
	if newCfg.CodeMogger.ChunkLines > 0 {
		current.CodeMogger.ChunkLines = newCfg.CodeMogger.ChunkLines
	}
	if len(newCfg.CodeMogger.Languages) > 0 {
		current.CodeMogger.Languages = newCfg.CodeMogger.Languages
	}

	// Embedder
	emb := newCfg.CodeMogger.Embedder
	if emb.Type != "" {
		current.CodeMogger.Embedder.Type = emb.Type
	}
	if emb.Model != "" {
		current.CodeMogger.Embedder.Model = emb.Model
	}
	if emb.OpenAI.APIBase != "" {
		current.CodeMogger.Embedder.OpenAI.APIBase = emb.OpenAI.APIBase
	}
	if emb.OpenAI.Model != "" {
		current.CodeMogger.Embedder.OpenAI.Model = emb.OpenAI.Model
	}
	if emb.OpenAI.APIKey != "" && emb.OpenAI.APIKey != "****" {
		current.CodeMogger.Embedder.OpenAI.APIKey = emb.OpenAI.APIKey
	}

	// Save to file
	configPath := h.index.ConfigPath
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".code_explorer", "config.json")
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create config directory", err)
		return
	}

	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to marshal config", err)
		return
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save config file", err)
		return
	}

	writeJSON(w, http.StatusOK, current)
}
