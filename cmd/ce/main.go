package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/config"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/logger"
	"github.com/liyu1981/code_explorer/pkg/server"
	"github.com/liyu1981/code_explorer/pkg/sqlitefs"
	"github.com/liyu1981/code_explorer/pkg/tools"
	zindex "github.com/liyu1981/code_explorer/pkg/zoekt/index"
	"github.com/rs/zerolog/log"
)

func initTools(r *tools.ToolRegistry, data map[string]any) error {
	if data == nil {
		return fmt.Errorf("data is nil")
	}

	cmIndex, ok := data["codemogger_index"]
	if !ok || cmIndex == nil {
		return fmt.Errorf("codemogger index is required")
	}

	r.RegisterTool(tools.NewCodeMoggerListFilesTool(cmIndex.(*codemogger.CodeIndex)))
	log.Debug().Msg("Load tool codemogger_list_files")
	r.RegisterTool(tools.NewCodeMoggerSearchTool(cmIndex.(*codemogger.CodeIndex)))
	log.Debug().Msg("Load tool codgemogger_search")
	r.RegisterTool(tools.NewReadFileTool())
	log.Debug().Msg("Load global tool read_file")
	r.RegisterTool(tools.NewGetTreeTool())
	log.Debug().Msg("Load global tool get_tree")
	r.RegisterTool(tools.NewGrepSearchTool())
	log.Debug().Msg("Load global tool grep_search")

	tools := r.List()
	var toolNames []string
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name())
	}
	log.Info().Interface("tools", toolNames).Msg("Registered tools")
	return nil
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

	// init zoekt
	zFs := sqlitefs.OpenFS(store)
	zIdx := zindex.NewZoektIndex(store, zFs)

	// init global agent tool registry
	if err := initTools(tools.GetGlobalToolRegistry(), map[string]any{
		"codemogger_index": idx,
	}); err != nil {
		log.Fatal().Err(err).Msg("Failed to init tool registry")
	}

	// init httpSrv
	port := os.Getenv("PORT")
	if port == "" {
		port = "12345"
	}

	srv := server.New(idx, zIdx)
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
