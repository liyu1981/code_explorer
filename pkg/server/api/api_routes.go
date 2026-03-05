package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
)

// ApiHandler represents the API handler
type ApiHandler struct {
	index *codemogger.CodeIndex
	hub   *WsHub
}

// ApiConfig holds the API handler configuration
type ApiConfig struct {
	Index *codemogger.CodeIndex
}

// NewHandler creates a new API handler instance
func NewHandler(config *ApiConfig) *ApiHandler {
	h := &ApiHandler{
		index: config.Index,
		hub:   NewWsHub(),
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
	mux.HandleFunc("GET /api/codebases", h.handleListCodebases)

	// Files
	mux.HandleFunc("GET /api/files", h.handleListFiles)

	// Indexing
	mux.HandleFunc("POST /api/index", h.handleIndex)

	// Search
	mux.HandleFunc("POST /api/search", h.handleSearch)
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
