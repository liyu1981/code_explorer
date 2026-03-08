package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/logger"
	"github.com/liyu1981/code_explorer/pkg/server"
	"github.com/rs/zerolog/log"
)

func getIndex(dbPath string) (*codemogger.CodeIndex, error) {
	cfg := codemogger.DefaultConfig()

	// Try to load config from file
	configPath := os.Getenv("CODE_EXPLORER_CONFIG")
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".code_explorer", "config.json")
	}

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err == nil {
			var fileCfg codemogger.Config
			if err := json.Unmarshal(data, &fileCfg); err == nil {
				// Merge configs (simplified)
				if fileCfg.DBPath != "" {
					cfg.DBPath = fileCfg.DBPath
				}
				if fileCfg.Embedder.Type != "" {
					cfg.Embedder.Type = fileCfg.Embedder.Type
				}
				if fileCfg.Embedder.Model != "" {
					cfg.Embedder.Model = fileCfg.Embedder.Model
				}
				if fileCfg.Embedder.OpenAI.APIBase != "" {
					cfg.Embedder.OpenAI.APIBase = fileCfg.Embedder.OpenAI.APIBase
				}
				if fileCfg.Embedder.OpenAI.APIKey != "" {
					cfg.Embedder.OpenAI.APIKey = fileCfg.Embedder.OpenAI.APIKey
				}
				if fileCfg.Embedder.OpenAI.Model != "" {
					cfg.Embedder.OpenAI.Model = fileCfg.Embedder.OpenAI.Model
				}
				if fileCfg.LLM != nil {
					cfg.LLM = fileCfg.LLM
				}
			}
		}
	}

	if dbPath == "" {
		if cfg.DBPath != "" {
			dbPath = cfg.DBPath
		} else {
			dbPath = codemogger.ProjectDbPath(".")
		}
	}

	// Create .codemogger if it doesn't exist
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	return codemogger.NewCodeIndex(dbPath, cfg)
}

func main() {
	logger.Init()

	port := os.Getenv("PORT")
	if port == "" {
		port = "12345"
	}

	idx, err := getIndex("")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open index")
	}
	defer idx.Close()

	srv := server.New(idx)
	log.Info().Msgf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatal().Err(err).Msg("Server failed")
	}
}
