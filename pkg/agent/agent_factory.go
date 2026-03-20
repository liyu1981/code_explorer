package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/rs/zerolog/log"
)

type AgentConfig struct {
	LLM             map[string]any `json:"llm"`
	Tools           []string       `json:"tools"`
	MaxIterations   int            `json:"max_iterations"`
	ContextLength   int            `json:"context_length"`
	AgentPromptName string         `json:"agent_prompt_name"`
	NoThink         bool           `json:"no_think"`
}

type AgentFactoryInterface interface {
	BuildFromConfig(
		ctx context.Context,
		cfg *AgentConfig,
		bindDataProviders ...AgentBindDataProvider,
	) (*Agent, error)
	GetAgentPromptSystemPrompt(ctx context.Context, name string) (string, error)
}

var (
	factoryInstance *AgentFactory
	factoryOnce     sync.Once
	factoryErr      error
	resetFactory    func()
)

type AgentFactory struct {
	toolRegistry *ToolRegistry
	store        *db.Store
	defaultLLM   map[string]any
	mu           sync.RWMutex
}

type AgentBindDataProvider func(m *map[string]any)

func InitAgentFactory(store *db.Store, defaultLLM map[string]any) error {
	factoryOnce.Do(func() {
		factoryInstance = &AgentFactory{
			toolRegistry: NewToolRegistry(),
			store:        store,
			defaultLLM:   defaultLLM,
		}
		factoryInstance.InitTools()
		factoryErr = nil
		resetFactory = func() {
			factoryInstance = nil
			factoryOnce = sync.Once{}
		}
	})
	return factoryErr
}

func GetAgentFactory() *AgentFactory {
	return factoryInstance
}

func ResetAgentFactory() {
	if resetFactory != nil {
		resetFactory()
	}
}

func NewAgentFactoryForTest(store *db.Store, defaultLLM map[string]any) *AgentFactory {
	return &AgentFactory{
		toolRegistry: NewToolRegistry(),
		store:        store,
		defaultLLM:   defaultLLM,
	}
}

// BuildTestAgent is a helper for tests to create an agent with mocked dependencies
func (f *AgentFactory) BuildTestAgent(llm LLM, opts ...AgentOption) *Agent {
	// Create a new tool registry for this agent
	toolRegistry := NewToolRegistry()

	// Register core tools
	f.InitTools()

	return newAgent(llm, "", "", toolRegistry, opts...)
}

func (f *AgentFactory) InitTools() {
	f.registerTool(NewListAgentSkillsTool())
	log.Debug().Msg("Registering tool list_agent_prompts")
	f.registerTool(NewSaveKnowledgeTool())
	log.Debug().Msg("Registering tool save_knowledge")
	f.registerTool(NewQueueTaskTool())
	log.Debug().Msg("Registering tool queue_task")
	f.registerTool(NewCodeMoggerListFilesTool())
	log.Debug().Msg("Registering tool codemogger_list_files")
	f.registerTool(NewCodeMoggerSearchTool())
	log.Debug().Msg("Registering tool codgemogger_search")
	f.registerTool(NewReadFileTool())
	log.Debug().Msg("Registering tool read_file")
	f.registerTool(NewGetTreeTool())
	log.Debug().Msg("Registering tool get_tree")
	f.registerTool(NewGrepSearchTool())
	log.Debug().Msg("Registering tool grep_search")

	tools := f.ToolRegistry().List()
	var toolNames []string
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Name())
	}
	log.Info().Interface("tools", toolNames).Msg("Registered tools")
}

func (f *AgentFactory) registerTool(tool Tool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.toolRegistry.Register(tool)
}

func (f *AgentFactory) ToolRegistry() *ToolRegistry {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.toolRegistry
}

func (f *AgentFactory) GetAgentPromptSystemPrompt(ctx context.Context, name string) (string, error) {
	if f.store == nil {
		return "", fmt.Errorf("store not initialized in AgentFactory")
	}
	p, err := f.store.GetPromptByName(ctx, name)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", fmt.Errorf("agent prompt %s not found", name)
	}
	return p.SystemPrompt, nil
}

func (f *AgentFactory) GetAgentUserPromptTpl(ctx context.Context, name string) (string, error) {
	if f.store == nil {
		return "", fmt.Errorf("store not initialized in AgentFactory")
	}
	p, err := f.store.GetPromptByName(ctx, name)
	if err != nil {
		return "", err
	}
	if p == nil {
		return "", fmt.Errorf("agent prompt %s not found", name)
	}
	return p.UserPromptTpl, nil
}

func (f *AgentFactory) GetAgentPromptTools(ctx context.Context, name string) ([]string, error) {
	if f.store == nil {
		return nil, fmt.Errorf("store not initialized in AgentFactory")
	}
	p, err := f.store.GetPromptByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("agent prompt %s not found", name)
	}
	if p.Tools == "" {
		return nil, nil
	}
	return strings.Fields(p.Tools), nil
}

func WithBindData(key string, value any) AgentBindDataProvider {
	return func(m *map[string]any) {
		(*m)[key] = value
	}
}

func (f *AgentFactory) BuildFromConfig(
	ctx context.Context,
	cfg *AgentConfig,
	bindDataProviders ...AgentBindDataProvider,
) (*Agent, error) {
	llmCfg := cfg.LLM
	if llmCfg == nil {
		llmCfg = f.defaultLLM
	}
	if cfg.NoThink {
		llmCfg["no_think"] = true
	}

	llm, err := f.buildLLM(llmCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build LLM: %w", err)
	}

	contextLength := cfg.ContextLength
	if contextLength <= 0 {
		if cl, ok := llmCfg["context_length"].(int); ok {
			contextLength = cl
		} else if cl, ok := llmCfg["context_length"].(float64); ok {
			contextLength = int(cl)
		}
	}
	if contextLength <= 0 {
		contextLength = 262144
	}

	bindData := &map[string]any{}
	for _, bindDataProvider := range bindDataProviders {
		bindDataProvider(bindData)
	}
	log.Debug().Interface("bindData", bindData).Msg("bind data prepared")

	systemPrompt, err := f.GetAgentPromptSystemPrompt(ctx, cfg.AgentPromptName)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent prompt system prompt: %w", err)
	}

	userPromptTpl, err := f.GetAgentUserPromptTpl(ctx, cfg.AgentPromptName)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent prompt user prompt template: %w", err)
	}

	toolRegistry := NewToolRegistry()
	f.mu.RLock()
	if cfg.AgentPromptName != "" && f.store != nil {
		promptTools, err := f.GetAgentPromptTools(ctx, cfg.AgentPromptName)
		if err != nil {
			f.mu.RUnlock()
			return nil, fmt.Errorf("failed to get skill tools: %w", err)
		}
		if len(promptTools) > 0 {
			for _, toolName := range promptTools {
				tool, ok := f.toolRegistry.Get(toolName)
				if !ok {
					f.mu.RUnlock()
					return nil, fmt.Errorf("tool %s not found in registry", toolName)
				}
				boundTool := tool.Clone()
				if err := boundTool.Bind(ctx, bindData); err != nil {
					f.mu.RUnlock()
					return nil, fmt.Errorf("failed to bind tool %s: %w", toolName, err)
				}
				toolRegistry.Register(boundTool)
			}
		}
	}
	f.mu.RUnlock()

	agent := newAgent(
		llm,
		systemPrompt,
		userPromptTpl,
		toolRegistry,
		WithMaxIterations(cfg.MaxIterations),
		WithContextLength(contextLength),
	)
	return agent, nil
}

func (f *AgentFactory) buildLLM(cfg map[string]any) (LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("llm config is required")
	}

	llmType, _ := cfg["type"].(string)
	var llm LLM
	switch llmType {
	case "openai":
		baseURL, _ := cfg["base_url"].(string)
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}
		model, _ := cfg["model"].(string)
		if model == "" {
			model = "qwen3.5:4b"
		}
		apiKey := os.Getenv("LLM_API_KEY")
		if ak, ok := cfg["api_key"].(string); ok {
			apiKey = ak
		}
		httpLLMClient := NewHTTPClientLLM(model, baseURL, apiKey)
		if cfg["no_think"].(bool) {
			httpLLMClient.SetNoThink(true)
		}
		llm = httpLLMClient

	case "mock":
		model, _ := cfg["model"].(string)
		responses, _ := cfg["responses"].([]any)
		respStrs := make([]string, len(responses))
		for i, r := range responses {
			respStrs[i], _ = r.(string)
		}
		llm = NewMockLLM(model, respStrs, nil)

	default:
		// Fallback for when type is not specified but it looks like an OpenAI-compatible config
		if model, ok := cfg["model"].(string); ok && model != "" {
			baseURL, _ := cfg["base_url"].(string)
			apiKey, _ := cfg["api_key"].(string)
			return NewHTTPClientLLM(model, baseURL, apiKey), nil
		}
		return nil, fmt.Errorf("unknown llm type: %s", llmType)
	}

	return llm, nil
}
