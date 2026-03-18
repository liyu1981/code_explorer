package api

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"runtime"
	"strconv"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/prompt"
	"github.com/liyu1981/code_explorer/pkg/task"
)

// ApiHandler represents the API handler
type ApiHandler struct {
	index        *codemogger.CodeIndex
	hub          *WsHub
	agentFactory *agent.AgentFactory
	taskManager  *task.Manager
}

// ApiConfig holds the API handler configuration
type ApiConfig struct {
	Index *codemogger.CodeIndex
}

// NewHandler creates a new API handler instance
func NewHandler(config *ApiConfig) *ApiHandler {
	var store *db.Store
	if config.Index != nil {
		store = config.Index.GetStore()
	}

	if err := agent.InitAgentFactory(store, GetSystemLLMConfig()); err != nil {
		panic("failed to initialize agent factory: " + err.Error())
	}

	factory := agent.GetAgentFactory()

	h := &ApiHandler{
		index:        config.Index,
		hub:          NewWsHub(),
		agentFactory: factory,
	}

	if config.Index != nil {
		prompt.SyncBuiltinPrompts(context.Background(), store)

		numWorkers := runtime.NumCPU() - 1
		isDev := false
		// In tests or dev mode, use fewer workers
		if flag.Lookup("test.v") != nil {
			isDev = true
			numWorkers = 1
		} else if os.Getenv("APP_ENV") == "development" {
			isDev = true
			numWorkers = 2
		}

		h.taskManager = task.NewManager(store, numWorkers, h.Publish)
		task.RegisterQueueHandlers(h.taskManager, config.Index, factory, h.Publish)

		h.taskManager.StartWorkers(context.Background(), isDev)
	}

	go h.hub.run()
	return h
}

// Stop stops the API handler and its background workers
func (h *ApiHandler) Stop() {
	if h.taskManager != nil {
		h.taskManager.Stop()
	}
}

// RegisterRoutes configures all API routes on the provided mux
func (h *ApiHandler) RegisterRoutes(mux *http.ServeMux) {
	// Global Helpers
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/version", h.handleVersion)
	mux.HandleFunc("/api/config", h.handleGetConfig)
	mux.HandleFunc("POST /api/config", h.handleUpdateConfig)
	mux.HandleFunc("/api/ws", h.handleWS)

	// Tasks
	mux.HandleFunc("GET /api/tasks", h.handleListTasks)
	mux.HandleFunc("GET /api/tasks/tree", h.handleGetTaskTree)

	// Codebases
	mux.HandleFunc("GET /api/codebases", h.handleListSystemCodebases)
	mux.HandleFunc("GET /api/codemogger/codebases", h.handleListCodebases)
	mux.HandleFunc("GET /api/codemogger/status", h.handleGetCodemoggerStatus)

	// Files
	mux.HandleFunc("GET /api/codemogger/files", h.handleListFiles)

	// Indexing
	mux.HandleFunc("POST /api/codemogger/index", h.handleIndex)

	// Search
	mux.HandleFunc("POST /api/codemogger/search", h.handleSearch)

	// Agent
	mux.HandleFunc("POST /api/agent/research", h.handleAgentResearch)

	// Research Sessions
	mux.HandleFunc("GET /api/research/sessions", h.handleListSessions)
	mux.HandleFunc("GET /api/research/sessions/manage", h.handleGetSessionsPaginated)
	mux.HandleFunc("GET /api/research/sessions/{id}/reports", h.handleGetSessionReports)
	mux.HandleFunc("POST /api/research/sessions", h.handleSaveSession)
	mux.HandleFunc("POST /api/research/sessions/{id}/summarize", h.handleSummarizeSession)
	mux.HandleFunc("DELETE /api/research/sessions/{id}/reports/{turnId}", h.handleDeleteReport)
	mux.HandleFunc("POST /api/research/sessions/{id}/archive", h.handleArchiveSession)
	mux.HandleFunc("DELETE /api/research/sessions/{id}", h.handleDeleteSession)

	// Saved Reports
	mux.HandleFunc("GET /api/saved_reports", h.handleListSavedReports)
	mux.HandleFunc("GET /api/saved_reports/{id}", h.handleGetSavedReport)
	mux.HandleFunc("POST /api/saved_reports", h.handleSaveSavedReport)
	mux.HandleFunc("DELETE /api/saved_reports/{id}", h.handleDeleteSavedReport)

	// Knowledge Base
	mux.HandleFunc("GET /api/knowledge", h.handleListKnowledgePages)
	mux.HandleFunc("GET /api/knowledge/get", h.handleGetKnowledgePage)
	mux.HandleFunc("POST /api/knowledge", h.handleCreateKnowledgePage)
	mux.HandleFunc("PUT /api/knowledge", h.handleUpdateKnowledgePage)
	mux.HandleFunc("DELETE /api/knowledge", h.handleDeleteKnowledgePage)
	mux.HandleFunc("POST /api/knowledge/build", h.handleBuildKnowledge)

	// Agent Skills
	mux.HandleFunc("GET /api/agent_prompts", h.handleListSkills)
	mux.HandleFunc("GET /api/agent_prompts/get", h.handleGetSkill)
	mux.HandleFunc("PUT /api/agent_prompts", h.handleUpdateSkill)
	mux.HandleFunc("DELETE /api/agent_prompts", h.handleDeleteSkill)
}

// handleHealth returns the health status of the API
func (h *ApiHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "healthy",
	})
}

// --- Helper Functions ---

func getIntParam(r *http.Request, key string, defaultValue int) int {
	valStr := r.URL.Query().Get(key)
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultValue
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string, err error) {
	details := ""
	if err != nil {
		details = err.Error()
	}
	writeJSON(w, status, map[string]any{"error": message, "details": details})
}
