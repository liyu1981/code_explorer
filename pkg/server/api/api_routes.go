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
	"github.com/liyu1981/code_explorer/pkg/agent/tools"
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

	factory := agent.NewAgentFactory(store, getSystemLLMConfig())
	if config.Index != nil {
		// Root-based discovery tools
		// We can't easily get the root path here without a specific codebase,
		// but the tool can be registered if it's dynamic or we register them per-task.
		// For now, let's register the ones that don't need rootPath in constructor
		// or will get it from the task context.
		factory.RegisterTool(agent.NewListFilesTool(config.Index))
		factory.RegisterTool(agent.NewSearchTool(config.Index))
		factory.RegisterTool(tools.NewQueueTaskTool(store))
		factory.RegisterTool(tools.NewPollTasksTool(store))
		factory.RegisterTool(tools.NewReadTaskOutputTool(store))
		factory.RegisterTool(tools.NewSaveKnowledgeTool(store))
		factory.RegisterTool(tools.NewListAgentSkillsTool(store))
	}

	h := &ApiHandler{
		index:        config.Index,
		hub:          NewWsHub(),
		agentFactory: factory,
	}

	if config.Index != nil {
		prompt.SyncBuiltinSkills(context.Background(), store)

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
		h.registerQueueHandlers()

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

func (h *ApiHandler) registerQueueHandlers() {
	h.taskManager.RegisterHandler("codemogger-index", h.handleIndexTask)
	h.taskManager.RegisterHandler("knowledge-build", h.handleKnowledgeBuildTask)
	h.taskManager.RegisterHandler("wiki-analyze", h.handleWikiAnalyzeTask)
}

func (h *ApiHandler) handleIndexTask(ctx context.Context, task *db.Task, updateProgress func(progress int, message string)) error {
	return h.index.HandleIndexTask(ctx, task, updateProgress)
}

func (h *ApiHandler) handleKnowledgeBuildTask(ctx context.Context, task *db.Task, updateProgress func(progress int, message string)) error {
	return agent.HandleKnowledgeBuildTask(ctx, h.index, task, h.taskManager, h.agentFactory, updateProgress)
}

func (h *ApiHandler) handleWikiAnalyzeTask(ctx context.Context, task *db.Task, updateProgress func(progress int, message string)) error {
	return agent.HandleKnowledgeWikiAnalyzeTask(ctx, h.index, task, h.agentFactory, updateProgress)
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
	mux.HandleFunc("GET /api/agent_skills", h.handleListSkills)
	mux.HandleFunc("GET /api/agent_skills/get", h.handleGetSkill)
	mux.HandleFunc("PUT /api/agent_skills", h.handleUpdateSkill)
	mux.HandleFunc("POST /api/agent_skills/reset", h.handleResetSkill)
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
