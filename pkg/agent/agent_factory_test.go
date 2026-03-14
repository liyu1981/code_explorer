package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

func TestNewAgentFactory(t *testing.T) {
	factory := NewAgentFactory(nil, nil)
	if factory == nil {
		t.Fatal("expected non-nil factory")
	}
	if factory.toolRegistry == nil {
		t.Error("expected tool registry to be initialized")
	}
}

func TestAgentFactory_RegisterTool(t *testing.T) {
	factory := NewAgentFactory(nil, nil)

	tool := &testMockTool{name: "test-tool"}
	factory.RegisterTool(tool)

	got, ok := factory.toolRegistry.Get("test-tool")
	if !ok {
		t.Error("expected tool to be registered")
	}
	if got.Name() != "test-tool" {
		t.Errorf("expected tool name test-tool, got %s", got.Name())
	}
}

func TestAgentFactory_Tools(t *testing.T) {
	factory := NewAgentFactory(nil, nil)

	tool := &testMockTool{name: "test-tool"}
	factory.RegisterTool(tool)

	tools := factory.Tools()
	if tools == nil {
		t.Error("expected non-nil tools")
	}

	list := tools.List()
	if len(list) != 1 {
		t.Errorf("expected 1 tool, got %d", len(list))
	}
}

func TestAgentFactory_GetSkillPrompt_StoreNil(t *testing.T) {
	factory := NewAgentFactory(nil, nil)

	_, err := factory.GetSkillPrompt(context.Background(), "test-skill")
	if err == nil {
		t.Error("expected error when store is nil")
	}
}

func TestAgentFactory_GetSkillTools_StoreNil(t *testing.T) {
	factory := NewAgentFactory(nil, nil)

	_, err := factory.GetSkillTools(context.Background(), "test-skill")
	if err == nil {
		t.Error("expected error when store is nil")
	}
}

func TestAgentFactory_BuildFromConfig_LLMNil(t *testing.T) {
	factory := NewAgentFactory(nil, map[string]any{
		"type":      "mock",
		"model":     "test-model",
		"responses": []any{"response 1"},
	})

	agent, err := factory.BuildFromConfig(context.Background(), &Config{
		MaxIterations: 10,
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

	skill := &db.Skill{
		Name:         "test-skill",
		SystemPrompt: "test prompt",
		Tools:        "tool1 tool2",
	}
	if err := store.CreateSkill(ctx, skill); err != nil {
		t.Fatalf("create skill: %v", err)
	}

	factory := NewAgentFactory(store, map[string]any{
		"type":      "mock",
		"model":     "test-model",
		"responses": []any{"response 1"},
	})

	factory.RegisterTool(&testMockTool{name: "tool1"})
	factory.RegisterTool(&testMockTool{name: "tool2"})

	agent, err := factory.BuildFromConfig(ctx, &Config{
		MaxIterations: 10,
		SkillName:     "test-skill",
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
}

func TestAgentFactory_BuildFromConfig_SkillWithEmptyTools(t *testing.T) {
	store, cleanup := db.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	skill := &db.Skill{
		Name:         "empty-tools-skill",
		SystemPrompt: "test prompt",
		Tools:        "",
	}
	if err := store.CreateSkill(ctx, skill); err != nil {
		t.Fatalf("create skill: %v", err)
	}

	factory := NewAgentFactory(store, map[string]any{
		"type":      "mock",
		"model":     "test-model",
		"responses": []any{"response 1"},
	})

	factory.RegisterTool(&testMockTool{name: "tool1"})

	agent, err := factory.BuildFromConfig(ctx, &Config{
		MaxIterations: 10,
		SkillName:     "empty-tools-skill",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestAgentFactory_BuildFromConfig_ContextLength(t *testing.T) {
	factory := NewAgentFactory(nil, map[string]any{
		"type":      "mock",
		"model":     "test-model",
		"responses": []any{"response 1"},
	})

	agent, err := factory.BuildFromConfig(context.Background(), &Config{
		MaxIterations: 10,
		ContextLength: 100000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestAgentFactory_buildLLM_NilConfig(t *testing.T) {
	factory := NewAgentFactory(nil, nil)

	_, err := factory.buildLLM(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestAgentFactory_buildLLM_UnknownType(t *testing.T) {
	factory := NewAgentFactory(nil, nil)

	_, err := factory.buildLLM(map[string]any{
		"type": "unknown-type",
	})
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

func TestAgentFactory_buildLLM_OpenAI(t *testing.T) {
	factory := NewAgentFactory(nil, nil)

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
	factory := NewAgentFactory(nil, nil)

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
	factory := NewAgentFactory(nil, nil)

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

func (m *testMockTool) Name() string               { return m.name }
func (m *testMockTool) Description() string        { return "mock description" }
func (m *testMockTool) Parameters() map[string]any { return nil }
func (m *testMockTool) Execute(ctx context.Context, input json.RawMessage, stream protocol.IStreamWriter) (string, error) {
	return "result", nil
}
