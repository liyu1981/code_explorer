package agent

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

	// Create some test skills
	skill1 := &db.Skill{
		Name:         "test-skill-1",
		SystemPrompt: "Test skill 1 prompt",
		Tags:         "tag1,tag2",
		Tools:        "tool1 tool2",
	}
	if err := store.CreateSkill(ctx, skill1); err != nil {
		t.Fatalf("create skill: %v", err)
	}

	skill2 := &db.Skill{
		Name:         "test-skill-2",
		SystemPrompt: "Test skill 2 prompt",
		Tags:         "tag3",
		Tools:        "tool3",
	}
	if err := store.CreateSkill(ctx, skill2); err != nil {
		t.Fatalf("create skill: %v", err)
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

	var skills []map[string]string
	if err := json.Unmarshal([]byte(res), &skills); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(skills))
	}

	// Check that both skills are present
	skillNames := make(map[string]bool)
	for _, skill := range skills {
		skillNames[skill["name"]] = true
	}

	if !skillNames["test-skill-1"] {
		t.Error("Expected test-skill-1 to be present")
	}
	if !skillNames["test-skill-2"] {
		t.Error("Expected test-skill-2 to be present")
	}

	// Check tags and tools for first skill
	for _, skill := range skills {
		if skill["name"] == "test-skill-1" {
			if skill["tags"] != "tag1,tag2" {
				t.Errorf("Expected tags 'tag1,tag2' for test-skill-1, got '%s'", skill["tags"])
			}
			if skill["tools"] != "tool1 tool2" {
				t.Errorf("Expected tools 'tool1 tool2' for test-skill-1, got '%s'", skill["tools"])
			}
		}
		if skill["name"] == "test-skill-2" {
			if skill["tags"] != "tag3" {
				t.Errorf("Expected tags 'tag3' for test-skill-2, got '%s'", skill["tags"])
			}
			if skill["tools"] != "tool3" {
				t.Errorf("Expected tools 'tool3' for test-skill-2, got '%s'", skill["tools"])
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
