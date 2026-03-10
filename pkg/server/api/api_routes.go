package api

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/agent/tools"
	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/task"
	"github.com/liyu1981/code_explorer/pkg/util"
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
	factory := agent.NewAgentFactory()
	if config.Index != nil {
		factory.RegisterTool(tools.NewListFilesTool(config.Index))
		factory.RegisterTool(tools.NewSearchTool(config.Index))
	}

	h := &ApiHandler{
		index:        config.Index,
		hub:          NewWsHub(),
		agentFactory: factory,
	}

	if config.Index != nil {
		h.taskManager = task.NewManager(config.Index.GetStore(), runtime.NumCPU()-1, h.Publish)
		h.registerQueueHandlers()
		h.taskManager.StartWorkers(context.Background(), util.IsDev())
	}

	go h.hub.run()
	return h
}

func (h *ApiHandler) registerQueueHandlers() {
	h.taskManager.RegisterHandler("codemogger-index", h.index.HandleIndexTask)
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
