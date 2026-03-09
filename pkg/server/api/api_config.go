package api

import (
	"encoding/json"
	"net/http"

	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/rs/zerolog/log"
)

func (h *ApiHandler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}

	// Create a deep enough copy to prevent masking from affecting the live config
	cfg := *config.Get()

	systemLLM := make(map[string]any)
	if cfg.System.LLM != nil {
		for k, v := range cfg.System.LLM {
			systemLLM[k] = v
		}
		if apiKey, ok := systemLLM["api_key"].(string); ok && apiKey != "" {
			systemLLM["api_key"] = "****"
		}
	}

	// Create a response structure that matches the frontend Config interface
	type systemResp struct {
		DbPath      string         `json:"db_path"`
		IsDefaultDb bool           `json:"is_default_db"`
		LLM         map[string]any `json:"llm"`
	}

	res := struct {
		System     systemResp              `json:"system"`
		Research   config.ResearchConfig   `json:"research"`
		CodeMogger config.CodeMoggerConfig `json:"codemogger"`
	}{
		System: systemResp{
			DbPath:      h.index.GetDbPath(),
			IsDefaultDb: config.Get().System.DBPath == "",
			LLM:         systemLLM,
		},
		Research:   cfg.Research,
		CodeMogger: cfg.CodeMogger,
	}

	// Also mask CodeMogger embedder key if not inheriting
	if !res.CodeMogger.InheritSystemLLM && res.CodeMogger.Embedder.OpenAI.APIKey != "" {
		res.CodeMogger.Embedder.OpenAI.APIKey = "****"
	}

	log.Debug().Interface("res", res).Msg("Sending config to UI")
	writeJSON(w, http.StatusOK, res)
}

func (h *ApiHandler) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	if h.index == nil {
		writeError(w, http.StatusInternalServerError, "Index not initialized", nil)
		return
	}

	var newCfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Update current config singleton
	current := config.Get()

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

	// Save to file via singleton
	if err := config.Save(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save config file", err)
		return
	}

	// Also update the index internal state (e.g. re-init embedder if needed)
	if err := h.index.ReloadConfig(); err != nil {
		log.Error().Err(err).Msg("Failed to reload configuration live")
	}

	writeJSON(w, http.StatusOK, current)
}
