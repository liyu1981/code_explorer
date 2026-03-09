package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

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
		if _, err := os.Stat(".config.json"); err == nil {
			configPath = ".config.json"
		} else {
			home, _ := os.UserHomeDir()
			configPath = filepath.Join(home, ".code_explorer", "config.json")
		}
	}

	if _, err := os.Stat(configPath); err == nil {
		log.Info().Str("path", configPath).Msg("Loading configuration")
		data, err := os.ReadFile(configPath)
		if err == nil {
			var fileCfg codemogger.Config
			if err := json.Unmarshal(data, &fileCfg); err == nil {
				log.Info().Interface("config", fileCfg).Msg("Configuration loaded successfully")
				// Merge configs
				if fileCfg.System.DBPath != "" {
					cfg.System.DBPath = fileCfg.System.DBPath
				}
				if fileCfg.System.LLM != nil {
					cfg.System.LLM = fileCfg.System.LLM
				}
				if fileCfg.Research.MaxReportsPerCodebase > 0 {
					cfg.Research.MaxReportsPerCodebase = fileCfg.Research.MaxReportsPerCodebase
				}
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
				cfg.CodeMogger.InheritSystemLLM = fileCfg.CodeMogger.InheritSystemLLM
				if fileCfg.CodeMogger.ChunkLines > 0 {
					cfg.CodeMogger.ChunkLines = fileCfg.CodeMogger.ChunkLines
				}
				if len(fileCfg.CodeMogger.Languages) > 0 {
					cfg.CodeMogger.Languages = fileCfg.CodeMogger.Languages
				}
			}
		}
	}

	if dbPath == "" {
		if cfg.System.DBPath != "" {
			dbPath = cfg.System.DBPath
		} else {
			dbPath = codemogger.ProjectDbPath(".")
		}
	}

	// Create db directory if it doesn't exist
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	log.Info().Interface("final_config", cfg).Msg("Final configuration for NewCodeIndex")
	return codemogger.NewCodeIndex(dbPath, cfg, configPath)
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
	httpSrv := &http.Server{
		Addr:    ":" + port,
		Handler: srv,
	}

	// Channel to listen for errors during startup
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		log.Info().Msgf("Starting server on :%s", port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking wait for either an error or a shutdown signal
	select {
	case err := <-serverErrors:
		log.Fatal().Err(err).Msg("Server failed to start")

	case sig := <-shutdown:
		log.Info().Msgf("Starting graceful shutdown, signal: %v", sig)

		// Give the server time to finish current requests
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpSrv.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Graceful shutdown failed, forcing close")
			if err := httpSrv.Close(); err != nil {
				log.Error().Err(err).Msg("Force close failed")
			}
		}
		log.Info().Msg("Graceful shutdown complete")
	}
}
