package db

import (
	"context"
	"testing"
)

func TestPromptStore(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	prompt := &Prompt{
		Name:         "test-prompt",
		SystemPrompt: "test prompt",
		Tags:         "test tags",
		Tools:        "tool1 tool2",
	}

	// Test Create
	if err := store.CreatePrompt(ctx, prompt); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	// Test Get
	got, err := store.GetPromptByName(ctx, "test-prompt")
	if err != nil {
		t.Fatalf("get prompt: %v", err)
	}
	if got.Name != "test-prompt" {
		t.Errorf("expected test-prompt, got %s", got.Name)
	}

	// Test Update
	prompt.Tags = "go devops"
	if err := store.UpdatePrompt(ctx, prompt); err != nil {
		t.Fatalf("update prompt: %v", err)
	}
	got, _ = store.GetPromptByName(ctx, "test-prompt")
	if got.Tags != "go devops" {
		t.Errorf("expected updated tags, got %s", got.Tags)
	}

	// Test List
	prompts, err := store.ListAgentPrompts(ctx)
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}
	if len(prompts) != 1 {
		t.Errorf("expected 1 prompt, got %d", len(prompts))
	}

	// Test Delete
	if err := store.DeletePrompt(ctx, prompt.ID); err != nil {
		t.Fatalf("delete prompt: %v", err)
	}
	got, _ = store.GetPromptByName(ctx, "test-prompt")
	if got != nil {
		t.Error("expected prompt to be deleted")
	}
}
