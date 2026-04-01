package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/llm"
	"github.com/liyu1981/code_explorer/pkg/logger"
	"github.com/liyu1981/code_explorer/pkg/server"
	"github.com/rs/zerolog/log"
)

func getSystemLLMConfig() map[string]any {
	llmCfg := make(map[string]any)
	if config.Get().System.LLM != nil {
		for k, v := range config.Get().System.LLM {
			llmCfg[k] = v
		}
	} else {
		llmCfg["type"] = "openai"
		llmCfg["model"] = os.Getenv("LLM_MODEL")
		llmCfg["base_url"] = os.Getenv("LLM_BASE_URL")
	}

	if llmCfg["model"] == nil || llmCfg["model"] == "" {
		llmCfg["model"] = "gpt-4o"
	}
	if llmCfg["base_url"] == nil || llmCfg["base_url"] == "" {
		llmCfg["base_url"] = "https://api.openai.com/v1"
	}
	if llmCfg["type"] == nil || llmCfg["type"] == "" {
		llmCfg["type"] = "openai"
	}
	if config.Get().System.ContextLength > 0 {
		llmCfg["context_length"] = config.Get().System.ContextLength
	} else {
		llmCfg["context_length"] = 262144
	}
	return llmCfg
}

func main() {
	logger.Init()

	// init config
	// Try to load config from file via singleton
	configPath := os.Getenv("CODE_EXPLORER_CONFIG")
	if err := config.Load(configPath); err != nil {
		log.Warn().Err(err).Msg("Failed to load configuration, using defaults")
	}
	cfg := config.Get()

	// init db
	_, _, store, err := db.InitDb(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open db")
	}

	// init codemogger
	idx, err := codemogger.NewCodeIndex(cfg, store)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to init codemogger")
	}
	defer idx.Close()

	// init global agent tool registry
	llm.GetGlobalToolRegistry()

	// init httpSrv
	port := os.Getenv("PORT")
	if port == "" {
		port = "12345"
	}

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
