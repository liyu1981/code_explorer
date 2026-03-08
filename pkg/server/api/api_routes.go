package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/agent/tools"
	"github.com/liyu1981/code_explorer/pkg/codemogger"
)

// ApiHandler represents the API handler
type ApiHandler struct {
	index        *codemogger.CodeIndex
	hub          *WsHub
	agentFactory *agent.AgentFactory
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
	go h.hub.run()
	return h
}

// RegisterRoutes configures all API routes on the provided mux
func (h *ApiHandler) RegisterRoutes(mux *http.ServeMux) {
	// Global Helpers
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/version", h.handleVersion)
	mux.HandleFunc("/api/ws", h.handleWS)

	// Codebases
	mux.HandleFunc("GET /api/codemogger/codebases", h.handleListCodebases)

	// Files
	mux.HandleFunc("GET /api/codemogger/files", h.handleListFiles)

	// Indexing
	mux.HandleFunc("POST /api/codemogger/index", h.handleIndex)

	// Search
	mux.HandleFunc("POST /api/codemogger/search", h.handleSearch)

	// Agent
	mux.HandleFunc("POST /api/agent/research", h.handleAgentResearch)
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
