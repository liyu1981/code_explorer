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

type AgentFactoryInterface interface {
	BuildFromConfig(ctx context.Context, cfg *Config) (*Agent, error)
	GetSkillPrompt(ctx context.Context, name string) (string, error)
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
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Create a new tool registry for this agent
	toolRegistry := NewToolRegistry()

	// Register core tools
	f.InitTools()

	return newAgent(llm, toolRegistry, opts...)
}

// registerToolToRegistry registers a tool to the given registry
func (f *AgentFactory) registerToolToRegistry(registry *ToolRegistry, tool Tool) {
	registry.Register(tool)
}

func (f *AgentFactory) InitTools() {
	f.registerTool(NewListAgentSkillsTool())
	log.Debug().Msg("Registering tool list_agent_skills")
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

func (f *AgentFactory) GetSkillPrompt(ctx context.Context, name string) (string, error) {
	if f.store == nil {
		return "", fmt.Errorf("store not initialized in AgentFactory")
	}
	skill, err := f.store.GetSkillByName(ctx, name)
	if err != nil {
		return "", err
	}
	if skill == nil {
		return "", fmt.Errorf("skill %s not found", name)
	}
	return skill.SystemPrompt, nil
}

func (f *AgentFactory) GetSkillTools(ctx context.Context, name string) ([]string, error) {
	if f.store == nil {
		return nil, fmt.Errorf("store not initialized in AgentFactory")
	}
	skill, err := f.store.GetSkillByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if skill == nil {
		return nil, fmt.Errorf("skill %s not found", name)
	}
	if skill.Tools == "" {
		return nil, nil
	}
	return strings.Fields(skill.Tools), nil
}

func (f *AgentFactory) BuildFromConfig(
	ctx context.Context,
	cfg *Config,
) (*Agent, error) {
	return f.buildFromConfigInternal(ctx, cfg)
}

// buildFromConfigInternal is the internal implementation that accepts bind data providers
func (f *AgentFactory) buildFromConfigInternal(
	ctx context.Context,
	cfg *Config,
	bindDataProviders ...AgentBindDataProvider,
) (*Agent, error) {
	llmCfg := cfg.LLM
	if llmCfg == nil {
		llmCfg = f.defaultLLM
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

	f.mu.RLock()
	toolRegistry := NewToolRegistry()

	if cfg.SkillName != "" && f.store != nil {
		skillTools, err := f.GetSkillTools(ctx, cfg.SkillName)
		if err != nil {
			f.mu.RUnlock()
			return nil, fmt.Errorf("failed to get skill tools: %w", err)
		}
		if len(skillTools) > 0 {
			for _, toolName := range skillTools {
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

	agent := newAgent(llm, toolRegistry, WithMaxIterations(cfg.MaxIterations), WithContextLength(contextLength))
	return agent, nil
}

func (f *AgentFactory) buildLLM(cfg map[string]any) (LLM, error) {
	if cfg == nil {
		return nil, fmt.Errorf("llm config is required")
	}

	llmType, _ := cfg["type"].(string)
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
		return NewHTTPClientLLM(model, baseURL, apiKey), nil

	case "mock":
		model, _ := cfg["model"].(string)
		responses, _ := cfg["responses"].([]any)
		respStrs := make([]string, len(responses))
		for i, r := range responses {
			respStrs[i], _ = r.(string)
		}
		return NewMockLLM(model, respStrs, nil), nil

	default:
		// Fallback for when type is not specified but it looks like an OpenAI-compatible config
		if model, ok := cfg["model"].(string); ok && model != "" {
			baseURL, _ := cfg["base_url"].(string)
			apiKey, _ := cfg["api_key"].(string)
			return NewHTTPClientLLM(model, baseURL, apiKey), nil
		}
		return nil, fmt.Errorf("unknown llm type: %s", llmType)
	}
}
