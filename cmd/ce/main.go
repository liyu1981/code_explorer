package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/logger"
	"github.com/liyu1981/code_explorer/pkg/server"
	"github.com/rs/zerolog/log"
)

func getIndex(dbPath string) (*codemogger.CodeIndex, error) {
	// Try to load config from file via singleton
	configPath := os.Getenv("CODE_EXPLORER_CONFIG")
	if err := config.Load(configPath); err != nil {
		log.Warn().Err(err).Msg("Failed to load configuration, using defaults")
	}

	cfg := config.Get()

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

	return codemogger.NewCodeIndex(dbPath)
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
