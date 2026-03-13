package db

import (
	"context"
	"testing"
)

func TestKnowledgeStore(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	cb, _ := store.GetOrCreateCodebase(ctx, "/test", "test", "local")

	page := &KnowledgePage{
		CodebaseID: cb.ID,
		Slug:       "test-page",
		Title:      "Test Page",
		Content:    "# Hello",
	}

	// Test Create
	if err := store.CreateKnowledgePage(ctx, page); err != nil {
		t.Fatalf("create knowledge page: %v", err)
	}

	// Test Get
	got, err := store.GetKnowledgePageBySlug(ctx, cb.ID, "test-page")
	if err != nil {
		t.Fatalf("get knowledge page: %v", err)
	}
	if got.Title != "Test Page" {
		t.Errorf("expected title Test Page, got %s", got.Title)
	}

	// Test List
	pages, err := store.ListKnowledgePages(ctx, cb.ID)
	if err != nil {
		t.Fatalf("list knowledge pages: %v", err)
	}
	if len(pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(pages))
	}

	// Test Update
	page.Title = "Updated Title"
	if err := store.UpdateKnowledgePage(ctx, page); err != nil {
		t.Fatalf("update knowledge page: %v", err)
	}
	got, _ = store.GetKnowledgePageBySlug(ctx, cb.ID, "test-page")
	if got.Title != "Updated Title" {
		t.Errorf("expected title Updated Title, got %s", got.Title)
	}

	// Test Delete
	if err := store.DeleteKnowledgePage(ctx, page.ID); err != nil {
		t.Fatalf("delete knowledge page: %v", err)
	}
	got, _ = store.GetKnowledgePageBySlug(ctx, cb.ID, "test-page")
	if got != nil {
		t.Error("expected page to be deleted")
	}
}
