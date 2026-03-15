package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func TestSaveKnowledgeTool(t *testing.T) {
	store, cleanup := db.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	stream := &mockStreamWriter{}
	tool := NewSaveKnowledgeTool()
	state := map[string]any{"store": store}
	if err := tool.Bind(context.Background(), &state); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}

	cb, err := store.GetOrCreateCodebase(ctx, "/tmp/test", "test", "local")
	if err != nil {
		t.Fatalf("Failed to create codebase: %v", err)
	}

	codebaseID := cb.ID
	slug := "architecture"
	title := "System Architecture"
	content := "This is the system architecture document."

	t.Run("Create new knowledge page", func(t *testing.T) {
		input, _ := json.Marshal(map[string]any{
			"codebase_id": codebaseID,
			"slug":        slug,
			"title":       title,
			"content":     content,
		})
		res, err := tool.Execute(ctx, input, stream)
		if err != nil {
			t.Fatalf("SaveKnowledgeTool failed: %v", err)
		}
		if res != "Knowledge page saved successfully" {
			t.Errorf("Unexpected result: %s", res)
		}

		// Verify page was created
		page, err := store.GetKnowledgePageBySlug(ctx, codebaseID, slug)
		if err != nil {
			t.Fatalf("GetKnowledgePageBySlug failed: %v", err)
		}
		if page == nil {
			t.Fatal("Knowledge page not found in DB")
		}
		if page.Title != title {
			t.Errorf("Expected title %s, got %s", title, page.Title)
		}
		if page.Content != content {
			t.Errorf("Expected content %s, got %s", content, page.Content)
		}
	})

	t.Run("Update existing knowledge page", func(t *testing.T) {
		newTitle := "Updated System Architecture"
		newContent := "Updated content for system architecture."
		input, _ := json.Marshal(map[string]any{
			"codebase_id": codebaseID,
			"slug":        slug,
			"title":       newTitle,
			"content":     newContent,
		})
		res, err := tool.Execute(ctx, input, stream)
		if err != nil {
			t.Fatalf("SaveKnowledgeTool failed: %v", err)
		}
		if res != "Knowledge page saved successfully" {
			t.Errorf("Unexpected result: %s", res)
		}

		// Verify page was updated
		page, err := store.GetKnowledgePageBySlug(ctx, codebaseID, slug)
		if err != nil {
			t.Fatalf("GetKnowledgePageBySlug failed: %v", err)
		}
		if page == nil {
			t.Fatal("Knowledge page not found in DB")
		}
		if page.Title != newTitle {
			t.Errorf("Expected title %s, got %s", newTitle, page.Title)
		}
		if page.Content != newContent {
			t.Errorf("Expected content %s, got %s", newContent, page.Content)
		}
	})
}
