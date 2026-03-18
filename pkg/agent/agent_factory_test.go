package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

func TestNewAgentFactoryForTest(t *testing.T) {
	factory := NewAgentFactoryForTest(nil, nil)
	if factory == nil {
		t.Fatal("expected non-nil factory")
	}
	if factory.toolRegistry == nil {
		t.Error("expected tool registry to be initialized")
	}
}

func TestAgentFactory_RegisterTool(t *testing.T) {
	factory := NewAgentFactoryForTest(nil, nil)

	tool := &testMockTool{name: "test-tool"}
	factory.registerTool(tool)

	got, ok := factory.toolRegistry.Get("test-tool")
	if !ok {
		t.Error("expected tool to be registered")
	}
	if got.Name() != "test-tool" {
		t.Errorf("expected tool name test-tool, got %s", got.Name())
	}
}

func TestAgentFactory_Tools(t *testing.T) {
	factory := NewAgentFactoryForTest(nil, nil)

	tool := &testMockTool{name: "test-tool"}
	factory.registerTool(tool)

	tools := factory.ToolRegistry()
	if tools == nil {
		t.Error("expected non-nil tools")
	}

	list := tools.List()
	if len(list) != 1 {
		t.Errorf("expected 1 tool, got %d", len(list))
	}
}

func TestAgentFactory_GetSkillPrompt_StoreNil(t *testing.T) {
	factory := NewAgentFactoryForTest(nil, nil)

	_, err := factory.GetAgentPromptSystemPrompt(context.Background(), "test-skill")
	if err == nil {
		t.Error("expected error when store is nil")
	}
}

func TestAgentFactory_GetSkillTools_StoreNil(t *testing.T) {
	factory := NewAgentFactoryForTest(nil, nil)

	_, err := factory.GetAgentPromptTools(context.Background(), "test-skill")
	if err == nil {
		t.Error("expected error when store is nil")
	}
}

func TestAgentFactory_BuildFromConfig_LLMNil(t *testing.T) {
	store, cleanup := db.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	prompt := &db.Prompt{
		Name:          "test-prompt",
		SystemPrompt:  "test prompt",
		UserPromptTpl: "test user prompt",
	}
	if err := store.CreatePrompt(ctx, prompt); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	factory := NewAgentFactoryForTest(store, map[string]any{
		"type":      "mock",
		"model":     "test-model",
		"responses": []any{"response 1"},
	})

	agent, err := factory.BuildFromConfig(context.Background(), &Config{
		MaxIterations:   10,
		AgentPromptName: "test-prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestAgentFactory_BuildFromConfig_WithSkillTools(t *testing.T) {
	store, cleanup := db.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	prompt := &db.Prompt{
		Name:         "test-prompt",
		SystemPrompt: "test prompt",
		Tools:        "tool1 tool2",
	}
	if err := store.CreatePrompt(ctx, prompt); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	factory := NewAgentFactoryForTest(store, map[string]any{
		"type":      "mock",
		"model":     "test-model",
		"responses": []any{"response 1"},
	})

	factory.registerTool(&testMockTool{name: "tool1"})
	factory.registerTool(&testMockTool{name: "tool2"})

	agent, err := factory.BuildFromConfig(ctx, &Config{
		MaxIterations:   10,
		AgentPromptName: "test-prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}

	tools := agent.tools.List()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	if !toolNames["tool1"] {
		t.Error("expected tool1 to be present")
	}
	if !toolNames["tool2"] {
		t.Error("expected tool2 to be present")
	}
}

func TestAgentFactory_BuildFromConfig_SkillWithTools_PreservesToolsOnUpdate(t *testing.T) {
	store, cleanup := db.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	prompt := &db.Prompt{
		Name:         "updatable-prompt",
		SystemPrompt: "initial prompt",
		Tools:        "toolA toolB",
	}
	if err := store.CreatePrompt(ctx, prompt); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	factory := NewAgentFactoryForTest(store, map[string]any{
		"type":      "mock",
		"model":     "test-model",
		"responses": []any{"response 1"},
	})

	factory.registerTool(&testMockTool{name: "toolA"})
	factory.registerTool(&testMockTool{name: "toolB"})
	factory.registerTool(&testMockTool{name: "toolC"})

	agent, err := factory.BuildFromConfig(ctx, &Config{
		MaxIterations:   10,
		AgentPromptName: "updatable-prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tools := agent.tools.List()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	if !toolNames["toolA"] {
		t.Error("expected toolA to be present")
	}
	if !toolNames["toolB"] {
		t.Error("expected toolB to be present")
	}
	if toolNames["toolC"] {
		t.Error("expected toolC NOT to be present (not in skill tools)")
	}
}

func TestAgentFactory_BuildFromConfig_SkillWithEmptyTools(t *testing.T) {
	store, cleanup := db.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	prompt := &db.Prompt{
		Name:         "empty-tools-prompt",
		SystemPrompt: "test prompt",
		Tools:        "",
	}
	if err := store.CreatePrompt(ctx, prompt); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	factory := NewAgentFactoryForTest(store, map[string]any{
		"type":      "mock",
		"model":     "test-model",
		"responses": []any{"response 1"},
	})

	factory.registerTool(&testMockTool{name: "tool1"})

	agent, err := factory.BuildFromConfig(ctx, &Config{
		MaxIterations:   10,
		AgentPromptName: "empty-tools-prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestAgentFactory_BuildFromConfig_ContextLength(t *testing.T) {
	store, cleanup := db.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	prompt := &db.Prompt{
		Name:          "test-prompt",
		SystemPrompt:  "test prompt",
		UserPromptTpl: "test user prompt",
	}
	if err := store.CreatePrompt(ctx, prompt); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	factory := NewAgentFactoryForTest(store, map[string]any{
		"type":      "mock",
		"model":     "test-model",
		"responses": []any{"response 1"},
	})

	agent, err := factory.BuildFromConfig(context.Background(), &Config{
		MaxIterations:   10,
		ContextLength:   100000,
		AgentPromptName: "test-prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestAgentFactory_buildLLM_NilConfig(t *testing.T) {
	factory := NewAgentFactoryForTest(nil, nil)

	_, err := factory.buildLLM(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestAgentFactory_buildLLM_UnknownType(t *testing.T) {
	factory := NewAgentFactoryForTest(nil, nil)

	_, err := factory.buildLLM(map[string]any{
		"type": "unknown-type",
	})
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestAgentFactory_buildLLM_OpenAI(t *testing.T) {
	factory := NewAgentFactoryForTest(nil, nil)

	llm, err := factory.buildLLM(map[string]any{
		"type":     "openai",
		"model":    "test-model",
		"base_url": "http://localhost:8080/v1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if llm == nil {
		t.Fatal("expected non-nil LLM")
	}
}

func TestAgentFactory_buildLLM_OpenAI_Default(t *testing.T) {
	factory := NewAgentFactoryForTest(nil, nil)

	llm, err := factory.buildLLM(map[string]any{
		"model": "test-model",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if llm == nil {
		t.Fatal("expected non-nil LLM")
	}
}

func TestAgentFactory_buildLLM_Mock(t *testing.T) {
	factory := NewAgentFactoryForTest(nil, nil)

	llm, err := factory.buildLLM(map[string]any{
		"type":      "mock",
		"model":     "test-model",
		"responses": []any{"response 1", "response 2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if llm == nil {
		t.Fatal("expected non-nil LLM")
	}
}

type testMockTool struct {
	name string
}

func (m *testMockTool) Name() string        { return m.name }
func (m *testMockTool) Description() string { return "mock description" }
func (m *testMockTool) Clone() Tool {
	return &testMockTool{name: m.name}
}
func (m *testMockTool) Parameters() map[string]any { return nil }
func (m *testMockTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	return "result", nil
}
func (m *testMockTool) Bind(ctx context.Context, state *map[string]any) error {
	return nil
}
