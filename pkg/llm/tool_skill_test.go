package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/protocol"
)

func TestListAgentSkillsTool(t *testing.T) {
	store, cleanup := db.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	stream := &skillMockStreamWriter{}

	// Create some test prompts
	prompt1 := &db.Prompt{
		Name:         "test-prompt-1",
		SystemPrompt: "Test prompt 1 prompt",
		Tags:         "tag1,tag2",
		Tools:        "tool1 tool2",
	}
	if err := store.CreatePrompt(ctx, prompt1); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	prompt2 := &db.Prompt{
		Name:         "test-prompt-2",
		SystemPrompt: "Test prompt 2 prompt",
		Tags:         "tag3",
		Tools:        "tool3",
	}
	if err := store.CreatePrompt(ctx, prompt2); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	tool := NewListAgentSkillsTool()
	state := map[string]any{"store": store}
	if err := tool.Bind(context.Background(), &state); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}

	res, err := tool.Execute(context.Background(), json.RawMessage("{}"), stream)
	if err != nil {
		t.Fatalf("ListAgentSkillsTool failed: %v", err)
	}

	var prompts []map[string]string
	if err := json.Unmarshal([]byte(res), &prompts); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(prompts) != 2 {
		t.Errorf("Expected 2 prompts, got %d", len(prompts))
	}

	// Check that both prompts are present
	promptNames := make(map[string]bool)
	for _, prompt := range prompts {
		promptNames[prompt["name"]] = true
	}

	if !promptNames["test-prompt-1"] {
		t.Error("Expected test-prompt-1 to be present")
	}
	if !promptNames["test-prompt-2"] {
		t.Error("Expected test-prompt-2 to be present")
	}

	// Check tags and tools for first prompt
	for _, prompt := range prompts {
		if prompt["name"] == "test-prompt-1" {
			if prompt["tags"] != "tag1,tag2" {
				t.Errorf("Expected tags 'tag1,tag2' for test-prompt-1, got '%s'", prompt["tags"])
			}
			if prompt["tools"] != "tool1 tool2" {
				t.Errorf("Expected tools 'tool1 tool2' for test-prompt-1, got '%s'", prompt["tools"])
			}
		}
		if prompt["name"] == "test-prompt-2" {
			if prompt["tags"] != "tag3" {
				t.Errorf("Expected tags 'tag3' for test-prompt-2, got '%s'", prompt["tags"])
			}
			if prompt["tools"] != "tool3" {
				t.Errorf("Expected tools 'tool3' for test-prompt-2, got '%s'", prompt["tools"])
			}
		}
	}
}

// mockStreamWriter is a simple implementation of IStreamWriter for testing
type skillMockStreamWriter struct {
	protocol.IStreamWriter
}

func (m *skillMockStreamWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
